package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Thre4dripper/tidefetch/internal/config"
	"github.com/Thre4dripper/tidefetch/internal/daemon"
	"github.com/Thre4dripper/tidefetch/internal/history"
	"github.com/Thre4dripper/tidefetch/pkg/aria2"
)

// Task is the compact list representation pushed to browsers.
// Heavy fields (bitfield, file lists, peers) are fetched on demand.
type Task struct {
	GID       string  `json:"gid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Total     int64   `json:"total"`
	Done      int64   `json:"done"`
	Uploaded  int64   `json:"uploaded"`
	DownSpeed int64   `json:"downSpeed"`
	UpSpeed   int64   `json:"upSpeed"`
	Conns     int64   `json:"conns"`
	Seeders   int64   `json:"seeders"`
	Seeding   bool    `json:"seeding"`
	Torrent   bool    `json:"torrent"`
	Dir       string  `json:"dir"`
	URI       string  `json:"uri,omitempty"`
	ErrorCode string  `json:"errorCode,omitempty"`
	ErrorMsg  string  `json:"errorMsg,omitempty"`
	NumFiles  int     `json:"numFiles"`
	Progress  float64 `json:"progress"`
	// Speeds is a downsampled lifetime download-speed history (bytes/s),
	// recorded while the task is active and frozen once it stops.
	Speeds []int64 `json:"speeds,omitempty"`
}

func toTask(s aria2.Status) Task {
	return Task{
		GID:       s.GID,
		Name:      s.Name(),
		Status:    s.Status,
		Total:     s.TotalLength.Int(),
		Done:      s.CompletedLength.Int(),
		Uploaded:  s.UploadLength.Int(),
		DownSpeed: s.DownloadSpeed.Int(),
		UpSpeed:   s.UploadSpeed.Int(),
		Conns:     s.Connections.Int(),
		Seeders:   s.NumSeeders.Int(),
		Seeding:   bool(s.Seeder),
		Torrent:   s.IsTorrent(),
		Dir:       s.Dir,
		URI:       s.PrimaryURI(),
		ErrorCode: s.ErrorCode,
		ErrorMsg:  s.ErrorMessage,
		NumFiles:  len(s.Files),
		Progress:  s.Progress(),
	}
}

// Stat is the global transfer statistic snapshot.
type Stat struct {
	DownSpeed  int64 `json:"downSpeed"`
	UpSpeed    int64 `json:"upSpeed"`
	NumActive  int64 `json:"numActive"`
	NumWaiting int64 `json:"numWaiting"`
	NumStopped int64 `json:"numStopped"`
	// SessionDown/SessionUp are bytes moved since this server started,
	// integrated from speed samples between polls.
	SessionDown int64 `json:"sessionDown"`
	SessionUp   int64 `json:"sessionUp"`
	// DiskFree/DiskTotal describe the filesystem of the download directory.
	DiskFree  int64 `json:"diskFree"`
	DiskTotal int64 `json:"diskTotal"`
}

type wsMsg struct {
	Type      string          `json:"type"` // snapshot | delta | conn
	Tasks     []Task          `json:"tasks,omitempty"`
	Updated   []Task          `json:"updated,omitempty"`
	Removed   []string        `json:"removed,omitempty"`
	Stat      *Stat           `json:"stat,omitempty"`
	Connected *bool           `json:"connected,omitempty"`
	Extra     json.RawMessage `json:"extra,omitempty"`
}

// hub owns the aria2 connection, polls state and fans updates out to clients.
type hub struct {
	cfg  *config.Config
	hist *history.Store

	mu        sync.RWMutex
	client    *aria2.Client
	rpcURL    string
	connected bool
	tasks     []Task
	hashes    map[string][32]byte
	stat      Stat
	aria2Ver  string
	speeds    map[string][]int64 // per-gid lifetime speed samples

	sessionDown float64
	sessionUp   float64
	lastPoll    time.Time

	clients   map[*wsClient]struct{}
	clientsMu sync.Mutex

	wake chan struct{}
}

type wsClient struct {
	conn *websocket.Conn
	send chan []byte
}

func newHub(cfg *config.Config, hist *history.Store, res *daemon.Result) *hub {
	return &hub{
		cfg:       cfg,
		hist:      hist,
		client:    res.Client,
		rpcURL:    res.URL,
		connected: true,
		aria2Ver:  res.Version,
		hashes:    map[string][32]byte{},
		speeds:    map[string][]int64{},
		clients:   map[*wsClient]struct{}{},
		wake:      make(chan struct{}, 1),
	}
}

func (h *hub) run(ctx context.Context) {
	interval := time.Duration(h.cfg.PollMS) * time.Millisecond
	if interval < 300*time.Millisecond {
		interval = 300 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	h.poll(ctx)
	notifs := h.currentClient().Notifications()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.poll(ctx)
		case <-h.wake:
			h.poll(ctx)
		case _, ok := <-notifs:
			if !ok {
				// Connection lost: reconnect with backoff.
				h.setConnected(false)
				h.reconnect(ctx)
				notifs = h.currentClient().Notifications()
				continue
			}
			h.poll(ctx)
		}
	}
}

func (h *hub) currentClient() *aria2.Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.client
}

// rpc returns the live client for API handlers.
func (h *hub) rpc() *aria2.Client { return h.currentClient() }

func (h *hub) requestRefresh() {
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

func (h *hub) setConnected(ok bool) {
	h.mu.Lock()
	changed := h.connected != ok
	h.connected = ok
	h.mu.Unlock()
	if changed {
		v := ok
		h.broadcast(wsMsg{Type: "conn", Connected: &v})
	}
}

func (h *hub) reconnect(ctx context.Context) {
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		c, err := daemon.Redial(ctx, h.rpcURL, h.cfg.Secret)
		if err == nil {
			h.mu.Lock()
			h.client = c
			h.mu.Unlock()
			h.setConnected(true)
			h.requestRefresh()
			return
		}
		// Try a full daemon connect (respawn) as fallback.
		if res, err := daemon.Connect(ctx, h.cfg); err == nil {
			h.mu.Lock()
			h.client = res.Client
			h.rpcURL = res.URL
			h.mu.Unlock()
			h.setConnected(true)
			h.requestRefresh()
			return
		}
		if backoff < 15*time.Second {
			backoff *= 2
		}
	}
}

func (h *hub) poll(ctx context.Context) {
	c := h.currentClient()
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	active, err1 := c.TellActive(cctx)
	waiting, err2 := c.TellWaiting(cctx, 0, 1000)
	stopped, err3 := c.TellStopped(cctx, 0, 1000)
	gstat, err4 := c.GetGlobalStat(cctx)
	if err1 != nil && err2 != nil && err3 != nil && err4 != nil {
		h.setConnected(false)
		return
	}
	h.setConnected(true)

	all := make([]aria2.Status, 0, len(active)+len(waiting)+len(stopped))
	all = append(all, active...)
	all = append(all, waiting...)
	all = append(all, stopped...)

	const maxSpeedSamples = 1200
	tasks := make([]Task, 0, len(all))
	h.mu.Lock()
	for _, s := range all {
		t := toTask(s)
		if s.Status == aria2.StatusActive {
			ring := append(h.speeds[s.GID], s.DownloadSpeed.Int())
			if len(ring) > maxSpeedSamples {
				ring = ring[len(ring)-maxSpeedSamples:]
			}
			h.speeds[s.GID] = ring
		}
		t.Speeds = downsampleSpeeds(h.speeds[s.GID], 48)
		tasks = append(tasks, t)
	}
	h.mu.Unlock()
	sortTasks(tasks)

	stat := Stat{
		DownSpeed:  gstat.DownloadSpeed.Int(),
		UpSpeed:    gstat.UploadSpeed.Int(),
		NumActive:  gstat.NumActive.Int(),
		NumWaiting: gstat.NumWaiting.Int(),
		NumStopped: gstat.NumStopped.Int(),
	}
	if free, total, err := diskUsage(h.cfg.DownloadDir); err == nil {
		stat.DiskFree, stat.DiskTotal = free, total
	}

	// Integrate transfer speed into session byte counters.
	now := time.Now()
	h.mu.Lock()
	if !h.lastPoll.IsZero() {
		dt := now.Sub(h.lastPoll).Seconds()
		if dt > 0 && dt < 30 {
			h.sessionDown += float64(stat.DownSpeed) * dt
			h.sessionUp += float64(stat.UpSpeed) * dt
		}
	}
	h.lastPoll = now
	stat.SessionDown = int64(h.sessionDown)
	stat.SessionUp = int64(h.sessionUp)
	h.mu.Unlock()

	h.recordHistory(all)

	// Diff against previous snapshot.
	newHashes := make(map[string][32]byte, len(tasks))
	var updated []Task
	for _, t := range tasks {
		b, _ := json.Marshal(t)
		sum := sha256.Sum256(b)
		newHashes[t.GID] = sum
		if prev, ok := h.hashes[t.GID]; !ok || prev != sum {
			updated = append(updated, t)
		}
	}
	var removed []string
	for gid := range h.hashes {
		if _, ok := newHashes[gid]; !ok {
			removed = append(removed, gid)
		}
	}

	h.mu.Lock()
	h.tasks = tasks
	h.hashes = newHashes
	h.stat = stat
	for _, gid := range removed {
		delete(h.speeds, gid)
	}
	h.mu.Unlock()

	if len(updated) > 0 || len(removed) > 0 {
		h.broadcast(wsMsg{Type: "delta", Updated: updated, Removed: removed, Stat: &stat})
	} else {
		h.broadcast(wsMsg{Type: "delta", Stat: &stat})
	}
}

var statusRank = map[string]int{
	aria2.StatusActive: 0, aria2.StatusWaiting: 1, aria2.StatusPaused: 2,
	aria2.StatusError: 3, aria2.StatusComplete: 4, aria2.StatusRemoved: 5,
}

func sortTasks(ts []Task) {
	sort.SliceStable(ts, func(i, j int) bool {
		if statusRank[ts[i].Status] != statusRank[ts[j].Status] {
			return statusRank[ts[i].Status] < statusRank[ts[j].Status]
		}
		return ts[i].Name < ts[j].Name
	})
}

func (h *hub) recordHistory(all []aria2.Status) {
	changed := false
	for _, s := range all {
		switch s.Status {
		case aria2.StatusComplete, aria2.StatusError, aria2.StatusRemoved:
			if h.hist.Has(s.GID, s.Status) {
				continue
			}
			name := s.Name()
			h.hist.Upsert(history.Entry{
				GID:      s.GID,
				Name:     name,
				URL:      s.PrimaryURI(),
				Dir:      s.Dir,
				Size:     s.TotalLength.Int(),
				Status:   s.Status,
				Category: history.Categorize(name, s.IsTorrent()),
				Added:    time.Now(),
				Finished: time.Now(),
			})
			changed = true
		}
	}
	if changed {
		if err := h.hist.Save(); err != nil {
			log.Printf("history save: %v", err)
		}
	}
}

// snapshot returns the current state for new clients / GET /api/state.
func (h *hub) snapshot() ([]Task, Stat, bool, string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	tasks := make([]Task, len(h.tasks))
	copy(tasks, h.tasks)
	return tasks, h.stat, h.connected, h.aria2Ver
}

// speedHistory returns the full-resolution lifetime speed samples for gid.
func (h *hub) speedHistory(gid string) []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ring := h.speeds[gid]
	out := make([]int64, len(ring))
	copy(out, ring)
	return out
}

// downsampleSpeeds bucket-averages samples into at most n points.
func downsampleSpeeds(samples []int64, n int) []int64 {
	if len(samples) == 0 {
		return nil
	}
	if len(samples) <= n {
		out := make([]int64, len(samples))
		copy(out, samples)
		return out
	}
	out := make([]int64, n)
	for i := 0; i < n; i++ {
		lo := i * len(samples) / n
		hi := (i + 1) * len(samples) / n
		if hi <= lo {
			hi = lo + 1
		}
		var sum int64
		for _, v := range samples[lo:hi] {
			sum += v
		}
		out[i] = sum / int64(hi-lo)
	}
	return out
}

func (h *hub) broadcast(m wsMsg) {
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- b:
		default:
			// Slow client: drop it rather than blocking the hub.
			close(c.send)
			delete(h.clients, c)
		}
	}
}

func (h *hub) addClient(c *wsClient) {
	h.clientsMu.Lock()
	h.clients[c] = struct{}{}
	h.clientsMu.Unlock()
}

func (h *hub) removeClient(c *wsClient) {
	h.clientsMu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.clientsMu.Unlock()
}

func (h *hub) close() {
	h.clientsMu.Lock()
	for c := range h.clients {
		close(c.send)
		delete(h.clients, c)
	}
	h.clientsMu.Unlock()
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "" || sameOrigin(origin, r.Host)
	},
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &wsClient{conn: conn, send: make(chan []byte, 64)}
	s.hub.addClient(client)

	// Initial snapshot.
	tasks, stat, connected, _ := s.hub.snapshot()
	first, _ := json.Marshal(wsMsg{Type: "snapshot", Tasks: tasks, Stat: &stat, Connected: &connected})

	go func() {
		defer func() {
			s.hub.removeClient(client)
			_ = conn.Close()
		}()
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, first); err != nil {
			return
		}
		ping := time.NewTicker(30 * time.Second)
		defer ping.Stop()
		for {
			select {
			case msg, ok := <-client.send:
				if !ok {
					return
				}
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ping.C:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Reader: consume control frames, detect close.
	go func() {
		conn.SetReadLimit(4096)
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				s.hub.removeClient(client)
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		}
	}()
}
