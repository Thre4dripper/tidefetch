package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/turbostart/tidefetch/internal/config"
)

const sessionCookie = "tf_session"

// authenticator implements password login with in-memory bearer sessions.
type authenticator struct {
	mu       sync.Mutex
	enabled  bool
	hash     []byte
	sessions map[string]time.Time
	failures map[string]failWindow
}

type failWindow struct {
	count int
	until time.Time
}

func newAuthenticator(cfg *config.Config, f *Flags, host string) (*authenticator, error) {
	a := &authenticator{
		sessions: map[string]time.Time{},
		failures: map[string]failWindow{},
	}

	if f.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(f.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		cfg.WebPasswordHash = string(h)
		if err := cfg.Save(); err != nil {
			return nil, err
		}
		log.Printf("web UI password updated")
	}

	switch {
	case f.NoAuth:
		a.enabled = false
	case cfg.WebPasswordHash != "":
		a.enabled = true
		a.hash = []byte(cfg.WebPasswordHash)
	default:
		if isLoopback(host) {
			// Local-only bind: auth optional.
			a.enabled = false
			log.Printf("no password set — auth disabled for loopback bind (set one with `tidefetch serve -password ...`)")
		} else {
			return nil, errors.New("refusing to listen on a non-loopback address without a password; " +
				"set one with `tidefetch serve -password ...` or pass -no-auth if an auth proxy fronts this server")
		}
	}
	return a, nil
}

func isLoopback(host string) bool {
	if host == "localhost" || host == "" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (a *authenticator) isEnabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.enabled
}

func (a *authenticator) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.isEnabled() {
			next.ServeHTTP(w, r)
			return
		}
		token := ""
		if c, err := r.Cookie(sessionCookie); err == nil {
			token = c.Value
		} else if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		}
		if !a.valid(token) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *authenticator) valid(token string) bool {
	if token == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(a.sessions, token)
		return false
	}
	return true
}

// handleLogin issues a session cookie after verifying the password.
func (a *authenticator) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !a.isEnabled() {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "authRequired": false})
		return
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	a.mu.Lock()
	fw := a.failures[ip]
	if fw.count >= 8 && time.Now().Before(fw.until) {
		a.mu.Unlock()
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many attempts — try again later"})
		return
	}
	a.mu.Unlock()

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	if bcrypt.CompareHashAndPassword(a.hash, []byte(req.Password)) != nil {
		a.mu.Lock()
		fw := a.failures[ip]
		fw.count++
		fw.until = time.Now().Add(5 * time.Minute)
		a.failures[ip] = fw
		a.mu.Unlock()
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "wrong password"})
		return
	}

	a.mu.Lock()
	delete(a.failures, ip)
	a.mu.Unlock()
	a.issueSession(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "authRequired": true})
}

// issueSession creates a session token and sets its cookie.
func (a *authenticator) issueSession(w http.ResponseWriter) {
	tok := randomToken()
	a.mu.Lock()
	a.sessions[tok] = time.Now().Add(30 * 24 * time.Hour)
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 3600,
	})
}

// setPassword updates the stored hash, enabling auth if it was off.
// It reports whether auth was newly enabled.
func (a *authenticator) setPassword(hash []byte) (newlyEnabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	newlyEnabled = !a.enabled
	a.hash = hash
	a.enabled = true
	return newlyEnabled
}

// verify checks a plaintext password against the current hash.
func (a *authenticator) verify(password string) bool {
	a.mu.Lock()
	hash := a.hash
	enabled := a.enabled
	a.mu.Unlock()
	if !enabled {
		return true
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}

func (a *authenticator) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		a.mu.Lock()
		delete(a.sessions, c.Value)
		a.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
