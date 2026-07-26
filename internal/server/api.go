package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/turbostart/tidefetch/internal/history"
	"github.com/turbostart/tidefetch/pkg/aria2"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func (s *Server) reqCtx(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 15*time.Second)
}

// GET /api/state — initial payload (also the WS-less fallback).
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	tasks, stat, connected, aria2Ver := s.hub.snapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":     s.version,
		"aria2":       aria2Ver,
		"connected":   connected,
		"downloadDir": s.cfg.DownloadDir,
		"authEnabled": s.auth.isEnabled(),
		"tasks":       tasks,
		"stat":        stat,
	})
}

// POST /api/password {current, new} — change (or first-set) the web password.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if len(req.New) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters"})
		return
	}
	if !s.auth.verify(req.Current) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "current password is wrong"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.New), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.cfg.WebPasswordHash = string(hash)
	if err := s.cfg.Save(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if s.auth.setPassword(hash) {
		// Auth just switched on: keep the requesting browser signed in.
		s.auth.issueSession(w)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// POST /api/add {kind, uris?, payload?, filename?, options?}
func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kind    string            `json:"kind"` // uri | torrent | metalink
		URIs    []string          `json:"uris"`
		Payload string            `json:"payload"` // base64 file content
		Options map[string]string `json:"options"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	opts := aria2.Options(req.Options)
	if opts == nil {
		opts = aria2.Options{}
	}
	if opts["dir"] == "" {
		opts["dir"] = s.cfg.DownloadDir
	}

	ctx, cancel := s.reqCtx(r)
	defer cancel()

	var gids []string
	var err error
	switch req.Kind {
	case "torrent":
		var blob []byte
		blob, err = base64.StdEncoding.DecodeString(req.Payload)
		if err == nil {
			var gid string
			gid, err = s.hub.rpc().AddTorrent(ctx, blob, nil, opts)
			gids = []string{gid}
		}
	case "metalink":
		var blob []byte
		blob, err = base64.StdEncoding.DecodeString(req.Payload)
		if err == nil {
			gids, err = s.hub.rpc().AddMetalink(ctx, blob, opts)
		}
	default: // uri — one task per line-group; magnet and http both work
		var gid string
		var out []string
		for _, u := range req.URIs {
			u = strings.TrimSpace(u)
			if u == "" {
				continue
			}
			gid, err = s.hub.rpc().AddURI(ctx, []string{u}, opts)
			if err != nil {
				break
			}
			out = append(out, gid)
		}
		gids = out
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	s.hub.requestRefresh()
	writeJSON(w, http.StatusOK, map[string]any{"gids": gids})
}

// POST /api/tasks/{gid}/action {action, deleteFiles?}
func (s *Server) handleTaskAction(w http.ResponseWriter, r *http.Request) {
	gid := r.PathValue("gid")
	var req struct {
		Action      string `json:"action"`
		DeleteFiles bool   `json:"deleteFiles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := s.reqCtx(r)
	defer cancel()
	c := s.hub.rpc()

	var err error
	switch req.Action {
	case "pause":
		err = c.Pause(ctx, gid)
	case "resume":
		err = c.Unpause(ctx, gid)
	case "remove":
		err = s.removeTask(ctx, gid, req.DeleteFiles)
	case "removeResult":
		err = c.RemoveDownloadResult(ctx, gid)
	case "retry":
		err = s.retryTask(ctx, gid)
	default:
		writeErr(w, http.StatusBadRequest, errBadAction)
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	s.hub.requestRefresh()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

var errBadAction = &apiError{"unknown action"}

type apiError struct{ msg string }

func (e *apiError) Error() string { return e.msg }

func (s *Server) removeTask(ctx context.Context, gid string, deleteFiles bool) error {
	c := s.hub.rpc()
	st, err := c.TellStatus(ctx, gid)
	if err != nil {
		return err
	}
	switch st.Status {
	case aria2.StatusActive, aria2.StatusWaiting, aria2.StatusPaused:
		if err := c.Remove(ctx, gid); err != nil {
			if err := c.ForceRemove(ctx, gid); err != nil {
				return err
			}
		}
		// Removing an in-progress download also leaves a "removed" result entry.
		_ = c.RemoveDownloadResult(ctx, gid)
	default:
		if err := c.RemoveDownloadResult(ctx, gid); err != nil {
			return err
		}
	}
	if deleteFiles {
		for _, f := range st.Files {
			if f.Path == "" || strings.HasPrefix(f.Path, "[") {
				continue
			}
			_ = os.Remove(f.Path)
			_ = os.Remove(f.Path + ".aria2")
		}
	}
	return nil
}

func (s *Server) retryTask(ctx context.Context, gid string) error {
	c := s.hub.rpc()
	st, err := c.TellStatus(ctx, gid)
	if err != nil {
		return err
	}
	uri := st.PrimaryURI()
	if uri == "" {
		return &apiError{"task has no source URI to retry"}
	}
	opts, _ := c.GetOption(ctx, gid)
	if opts == nil {
		opts = aria2.Options{}
	}
	if opts["dir"] == "" {
		opts["dir"] = st.Dir
	}
	delete(opts, "select-file")
	if _, err := c.AddURI(ctx, []string{uri}, opts); err != nil {
		return err
	}
	return c.RemoveDownloadResult(ctx, gid)
}

// POST /api/tasks/actions {action} — bulk operations.
func (s *Server) handleBulkAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"` // pauseAll | resumeAll | purge
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := s.reqCtx(r)
	defer cancel()
	c := s.hub.rpc()

	var err error
	switch req.Action {
	case "pauseAll":
		err = c.PauseAll(ctx)
	case "resumeAll":
		err = c.UnpauseAll(ctx)
	case "purge":
		err = c.PurgeDownloadResult(ctx)
	default:
		writeErr(w, http.StatusBadRequest, errBadAction)
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	s.hub.requestRefresh()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// FileInfo is the per-file detail row.
type FileInfo struct {
	Index    int64  `json:"index"`
	Path     string `json:"path"`
	Length   int64  `json:"length"`
	Done     int64  `json:"done"`
	Selected bool   `json:"selected"`
	URI      string `json:"uri,omitempty"`
}

// PeerInfo is the per-peer detail row.
type PeerInfo struct {
	IP        string  `json:"ip"`
	Port      int64   `json:"port"`
	DownSpeed int64   `json:"downSpeed"`
	UpSpeed   int64   `json:"upSpeed"`
	Seeder    bool    `json:"seeder"`
	Progress  float64 `json:"progress"`
}

// GET /api/tasks/{gid} — full detail including a downsampled piece map.
func (s *Server) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	gid := r.PathValue("gid")
	ctx, cancel := s.reqCtx(r)
	defer cancel()
	c := s.hub.rpc()

	st, err := c.TellStatus(ctx, gid)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}

	files := make([]FileInfo, 0, len(st.Files))
	for _, f := range st.Files {
		fi := FileInfo{
			Index:    f.Index.Int(),
			Path:     f.Path,
			Length:   f.Length.Int(),
			Done:     f.CompletedLength.Int(),
			Selected: bool(f.Selected),
		}
		if len(f.URIs) > 0 {
			fi.URI = f.URIs[0].URI
		}
		files = append(files, fi)
	}

	var peers []PeerInfo
	if st.IsTorrent() && st.Status == aria2.StatusActive {
		if ps, err := c.GetPeers(ctx, gid); err == nil {
			for _, p := range ps {
				pieces := float64(0)
				if st.NumPieces > 0 {
					pieces = bitfieldCompletion(p.Bitfield, int(st.NumPieces.Int()))
				}
				peers = append(peers, PeerInfo{
					IP: p.IP, Port: p.Port.Int(),
					DownSpeed: p.DownloadSpeed.Int(), UpSpeed: p.UploadSpeed.Int(),
					Seeder: bool(p.Seeder), Progress: pieces,
				})
			}
		}
	}

	var servers []string
	if st.Status == aria2.StatusActive && !st.IsTorrent() {
		if groups, err := c.GetServers(ctx, gid); err == nil {
			for _, g := range groups {
				for _, sv := range g.Servers {
					servers = append(servers, sv.CurrentURI)
				}
			}
		}
	}

	detail := map[string]any{
		"task":         toTask(st),
		"files":        files,
		"peers":        peers,
		"servers":      servers,
		"pieces":       downsampleBitfield(st.Bitfield, int(st.NumPieces.Int()), 240),
		"speedHistory": s.hub.speedHistory(gid),
		"bt": map[string]any{
			"infoHash":  st.InfoHash,
			"pieceLen":  st.PieceLength.Int(),
			"numPieces": st.NumPieces.Int(),
		},
	}
	writeJSON(w, http.StatusOK, detail)
}

// downsampleBitfield buckets an aria2 hex bitfield into at most n cells with
// completion levels 0..8, so a million-piece torrent renders as ≤n values.
func downsampleBitfield(hexStr string, numPieces, n int) []int {
	if numPieces <= 0 || hexStr == "" {
		return nil
	}
	if n > numPieces {
		n = numPieces
	}
	buckets := make([]int, n)
	counts := make([]int, n)
	for i := 0; i < numPieces; i++ {
		ci := i / 4
		if ci >= len(hexStr) {
			break
		}
		v := hexVal(hexStr[ci])
		bit := (v >> (3 - uint(i%4))) & 1
		b := i * n / numPieces
		counts[b]++
		buckets[b] += int(bit)
	}
	out := make([]int, n)
	for i := range buckets {
		if counts[i] == 0 {
			continue
		}
		out[i] = buckets[i] * 8 / counts[i]
	}
	return out
}

func bitfieldCompletion(hexStr string, numPieces int) float64 {
	if numPieces <= 0 || hexStr == "" {
		return 0
	}
	set := 0
	for i := 0; i < numPieces; i++ {
		ci := i / 4
		if ci >= len(hexStr) {
			break
		}
		v := hexVal(hexStr[ci])
		set += int((v >> (3 - uint(i%4))) & 1)
	}
	return float64(set) / float64(numPieces)
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// POST /api/tasks/{gid}/files {indices:[...]}
func (s *Server) handleSelectFiles(w http.ResponseWriter, r *http.Request) {
	gid := r.PathValue("gid")
	var req struct {
		Indices []int64 `json:"indices"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	parts := make([]string, len(req.Indices))
	for i, idx := range req.Indices {
		parts[i] = strconv.FormatInt(idx, 10)
	}
	ctx, cancel := s.reqCtx(r)
	defer cancel()
	if err := s.hub.rpc().ChangeOption(ctx, gid, aria2.Options{"select-file": strings.Join(parts, ",")}); err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	s.hub.requestRefresh()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// GET /api/tasks/{gid}/options
func (s *Server) handleGetTaskOptions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.reqCtx(r)
	defer cancel()
	opts, err := s.hub.rpc().GetOption(ctx, r.PathValue("gid"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, opts)
}

// PUT /api/tasks/{gid}/options {key:value,...}
func (s *Server) handlePutTaskOptions(w http.ResponseWriter, r *http.Request) {
	var opts map[string]string
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := s.reqCtx(r)
	defer cancel()
	if err := s.hub.rpc().ChangeOption(ctx, r.PathValue("gid"), aria2.Options(opts)); err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	s.hub.requestRefresh()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// POST /api/tasks/{gid}/position {pos, how}
func (s *Server) handlePosition(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Pos int    `json:"pos"`
		How string `json:"how"` // POS_SET | POS_CUR | POS_END
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.How == "" {
		req.How = "POS_CUR"
	}
	ctx, cancel := s.reqCtx(r)
	defer cancel()
	pos, err := s.hub.rpc().ChangePosition(ctx, r.PathValue("gid"), req.Pos, req.How)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	s.hub.requestRefresh()
	writeJSON(w, http.StatusOK, map[string]int{"pos": pos})
}

// GET /api/options — global aria2 options.
func (s *Server) handleGetGlobalOptions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.reqCtx(r)
	defer cancel()
	opts, err := s.hub.rpc().GetGlobalOption(ctx)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, opts)
}

// PUT /api/options {key:value,...}
func (s *Server) handlePutGlobalOptions(w http.ResponseWriter, r *http.Request) {
	var opts map[string]string
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := s.reqCtx(r)
	defer cancel()
	if err := s.hub.rpc().ChangeGlobalOption(ctx, aria2.Options(opts)); err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	s.hub.requestRefresh()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// GET /api/history?q=&category=
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	cat := r.URL.Query().Get("category")
	all := s.hist.All()
	out := make([]history.Entry, 0, len(all))
	for _, e := range all {
		if cat != "" && e.Category != cat {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(e.Name), q) && !strings.Contains(strings.ToLower(e.URL), q) {
			continue
		}
		out = append(out, e)
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": out, "categories": history.Categories})
}

// DELETE /api/history/{gid}
func (s *Server) handleDeleteHistoryEntry(w http.ResponseWriter, r *http.Request) {
	s.hist.Delete(r.PathValue("gid"))
	_ = s.hist.Save()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// DELETE /api/history
func (s *Server) handleClearHistory(w http.ResponseWriter, r *http.Request) {
	s.hist.Clear()
	_ = s.hist.Save()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// GET /api/browse?path= — server-side directory browser for picking save dirs.
// Walks up to the nearest existing ancestor instead of dead-ending, so stale
// paths (deleted folders, other machines' configs) still open a usable view.
func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	p := r.URL.Query().Get("path")
	if p == "" {
		p = s.cfg.DownloadDir
	}
	p = filepath.Clean(p)
	for {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			break
		}
		parent := filepath.Dir(p)
		if parent == p { // reached the root and it still failed
			p = home
			break
		}
		p = parent
	}

	entries, err := os.ReadDir(p)
	if err != nil {
		// Last resort: home.
		p = home
		if entries, err = os.ReadDir(p); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
	}
	type dirEntry struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	dirs := []dirEntry{}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dirs = append(dirs, dirEntry{Name: e.Name(), Path: filepath.Join(p, e.Name())})
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })

	free, total, _ := diskUsage(p)
	writeJSON(w, http.StatusOK, map[string]any{
		"path":        p,
		"parent":      filepath.Dir(p),
		"dirs":        dirs,
		"free":        free,
		"total":       total,
		"home":        home,
		"downloadDir": s.cfg.DownloadDir,
	})
}

// POST /api/browse/mkdir {path, name} — create a subfolder while browsing.
func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid folder name"})
		return
	}
	newPath := filepath.Join(filepath.Clean(req.Path), name)
	if err := os.Mkdir(newPath, 0o755); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": newPath})
}
