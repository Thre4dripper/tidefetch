// Package server implements the tidefetch web UI backend: a broker that owns
// a single aria2 RPC connection, caches state, pushes WebSocket deltas to
// browsers and exposes an authenticated REST API for every download action.
package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Thre4dripper/tidefetch/internal/config"
	"github.com/Thre4dripper/tidefetch/internal/daemon"
	"github.com/Thre4dripper/tidefetch/internal/history"
	"github.com/Thre4dripper/tidefetch/web"
)

// Flags are the `tidefetch serve` command line options.
type Flags struct {
	Host     string
	Port     int
	Password string
	NoAuth   bool

	URL, Secret, Dir string
	NoSpawn          bool
}

// Server is the web UI HTTP server.
type Server struct {
	cfg     *config.Config
	hist    *history.Store
	hub     *hub
	auth    *authenticator
	version string
	addr    string
}

// New builds a Server from configuration and the established daemon connection.
func New(cfg *config.Config, hist *history.Store, res *daemon.Result, version string, f *Flags) (*Server, error) {
	host := cfg.WebHost
	if f.Host != "" {
		host = f.Host
	}
	port := cfg.WebPort
	if f.Port != 0 {
		port = f.Port
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	auth, err := newAuthenticator(cfg, f, host)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:     cfg,
		hist:    hist,
		hub:     newHub(cfg, hist, res),
		auth:    auth,
		version: version,
		addr:    addr,
	}
	return s, nil
}

// Run serves until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	s.routes(mux)

	httpSrv := &http.Server{
		Addr:              s.addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go s.hub.run(ctx)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("tidefetch web UI listening on http://%s", s.addr)
		if !s.auth.enabled {
			log.Printf("authentication: disabled")
		}
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutCtx)
	s.hub.close()
	return nil
}

func (s *Server) routes(mux *http.ServeMux) {
	// Static SPA.
	dist, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(dist))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			if _, err := fs.Stat(dist, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
				// SPA fallback: unknown paths serve the app shell.
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	})

	// Session endpoints (unauthenticated).
	mux.HandleFunc("POST /api/login", s.auth.handleLogin)
	mux.HandleFunc("POST /api/logout", s.auth.handleLogout)

	// Authenticated API.
	api := func(pattern string, h http.HandlerFunc) {
		mux.Handle(pattern, s.auth.middleware(csrfGuard(h)))
	}
	api("GET /api/state", s.handleState)
	api("GET /api/ws", s.handleWS)
	api("POST /api/add", s.handleAdd)
	api("POST /api/tasks/{gid}/action", s.handleTaskAction)
	api("POST /api/tasks/actions", s.handleBulkAction)
	api("GET /api/tasks/{gid}", s.handleTaskDetail)
	api("POST /api/tasks/{gid}/files", s.handleSelectFiles)
	api("GET /api/tasks/{gid}/options", s.handleGetTaskOptions)
	api("PUT /api/tasks/{gid}/options", s.handlePutTaskOptions)
	api("POST /api/tasks/{gid}/position", s.handlePosition)
	api("GET /api/options", s.handleGetGlobalOptions)
	api("PUT /api/options", s.handlePutGlobalOptions)
	api("GET /api/history", s.handleHistory)
	api("DELETE /api/history/{gid}", s.handleDeleteHistoryEntry)
	api("DELETE /api/history", s.handleClearHistory)
	api("GET /api/browse", s.handleBrowse)
	api("POST /api/browse/mkdir", s.handleMkdir)
	api("GET /api/probe", s.handleProbe)
	api("POST /api/password", s.handleChangePassword)
}

// securityHeaders sets a strict CSP suited to the embedded SPA.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; "+
				"connect-src 'self' ws: wss:; frame-ancestors 'none'; base-uri 'none'")
		next.ServeHTTP(w, r)
	})
}

// csrfGuard rejects state-changing cross-origin requests. The SPA is
// same-origin; browsers always send Origin on cross-origin requests.
func csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			origin := r.Header.Get("Origin")
			if origin != "" && !sameOrigin(origin, r.Host) {
				http.Error(w, "cross-origin request rejected", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func sameOrigin(origin, host string) bool {
	origin = strings.TrimPrefix(origin, "http://")
	origin = strings.TrimPrefix(origin, "https://")
	return origin == host
}
