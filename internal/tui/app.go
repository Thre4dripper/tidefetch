// Package tui implements the tidefetch terminal user interface.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Thre4dripper/tidefetch/internal/config"
	"github.com/Thre4dripper/tidefetch/internal/daemon"
	"github.com/Thre4dripper/tidefetch/internal/history"
	"github.com/Thre4dripper/tidefetch/pkg/aria2"
)

// View identifies the active screen.
type View int

const (
	viewDownloads View = iota
	viewAdd
	viewDetails
	viewHistory
	viewSettings
	viewHelp
	viewFiles
)

type toastLevel int

const (
	toastInfo toastLevel = iota
	toastOK
	toastErr
)

type toast struct {
	text    string
	level   toastLevel
	expires time.Time
}

// logEntry is one line of the activity log shown in the side panel.
type logEntry struct {
	at    time.Time
	text  string
	level toastLevel
}

// App is the root Bubble Tea model.
type App struct {
	cfg  *config.Config
	hist *history.Store

	client    *aria2.Client
	connected bool
	spawned   bool
	version   string
	rpcURL    string

	width, height int
	view          View

	// downloads state
	stat      aria2.GlobalStat
	active    []aria2.Status
	waiting   []aria2.Status
	stopped   []aria2.Status
	rows      []aria2.Status
	tab       int // 0 all, 1 active, 2 queued, 3 finished
	sortMode  int // 0 default, 1 name, 2 size, 3 speed, 4 progress
	cursor    int
	scroll    int
	filter    string
	searching bool
	searchBuf string
	speedHist []float64            // global download speed samples
	upHist    []float64            // global upload speed samples
	gidHist   map[string][]float64 // per-task download speed samples
	sidebar   bool                 // side panel visible
	pollBusy  bool
	histDirty bool
	lastRecon time.Time
	startedAt time.Time

	// mouse hit regions, rebuilt on every render
	hits []hitbox

	// sub-views
	add      addModel
	details  detailsModel
	historyV historyModel
	settings settingsModel
	files    fileBrowser
	confirm  *confirmModel
	picker   *dirPicker

	toasts []toast
	log    []logEntry

	// startup URLs handed over from the CLI
	initialAdds []string
}

// New builds the root model.
func New(cfg *config.Config, hist *history.Store, res *daemon.Result, initialAdds []string) *App {
	// Palette first: every style below is derived from the active theme.
	applyTheme(cfg.Theme)

	a := &App{
		cfg:         cfg,
		hist:        hist,
		client:      res.Client,
		connected:   true,
		spawned:     res.Spawned,
		version:     res.Version,
		rpcURL:      res.URL,
		view:        viewDownloads,
		initialAdds: initialAdds,
		gidHist:     map[string][]float64{},
		sidebar:     cfg.Sidebar,
		startedAt:   time.Now(),
	}
	a.add = newAddModel(cfg)
	a.details = newDetailsModel()
	a.historyV = newHistoryModel()
	a.settings = newSettingsModel()
	a.files = newFileBrowser(cfg.DownloadDir)
	return a
}

func (a *App) pollEvery() time.Duration { return time.Duration(a.cfg.PollMS) * time.Millisecond }

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tickCmd(a.pollEvery()),
		pollCmd(a.client),
		listenNotifs(a.client.Notifications()),
	}
	if len(a.initialAdds) > 0 {
		urls := a.initialAdds
		client := a.client
		opts := aria2.Options{aria2.OptDir: a.cfg.DownloadDir}
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := rpcCtx()
			defer cancel()
			n := 0
			var lastErr error
			for _, u := range urls {
				if _, err := client.AddURI(ctx, []string{u}, opts); err != nil {
					lastErr = err
				} else {
					n++
				}
			}
			return addedMsg{count: n, err: lastErr}
		})
	}
	return tea.Batch(cmds...)
}

func (a *App) pushToast(text string, level toastLevel) {
	a.toasts = append(a.toasts, toast{text: text, level: level, expires: time.Now().Add(4 * time.Second)})
	if len(a.toasts) > 3 {
		a.toasts = a.toasts[len(a.toasts)-3:]
	}
	a.log = append(a.log, logEntry{at: time.Now(), text: text, level: level})
	if len(a.log) > 100 {
		a.log = a.log[len(a.log)-100:]
	}
}

func (a *App) pruneToasts() {
	now := time.Now()
	keep := a.toasts[:0]
	for _, t := range a.toasts {
		if t.expires.After(now) {
			keep = append(keep, t)
		}
	}
	a.toasts = keep
}

// selected returns the highlighted download, if any.
func (a *App) selected() *aria2.Status {
	if a.cursor >= 0 && a.cursor < len(a.rows) {
		return &a.rows[a.cursor]
	}
	return nil
}

// buildRows recomputes the visible list from the raw polls.
func (a *App) buildRows() {
	var src []aria2.Status
	switch a.tab {
	case 1:
		src = a.active
	case 2:
		src = a.waiting
	case 3:
		src = a.stopped
	default:
		src = make([]aria2.Status, 0, len(a.active)+len(a.waiting)+len(a.stopped))
		src = append(src, a.active...)
		src = append(src, a.waiting...)
		src = append(src, a.stopped...)
	}
	if a.filter != "" {
		q := strings.ToLower(a.filter)
		filtered := make([]aria2.Status, 0, len(src))
		for _, s := range src {
			if strings.Contains(strings.ToLower(s.Name()), q) ||
				strings.Contains(strings.ToLower(s.PrimaryURI()), q) {
				filtered = append(filtered, s)
			}
		}
		src = filtered
	}
	rows := make([]aria2.Status, len(src))
	copy(rows, src)
	switch a.sortMode {
	case 1:
		sort.SliceStable(rows, func(i, j int) bool {
			return strings.ToLower(rows[i].Name()) < strings.ToLower(rows[j].Name())
		})
	case 2:
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].TotalLength > rows[j].TotalLength })
	case 3:
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].DownloadSpeed > rows[j].DownloadSpeed })
	case 4:
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Progress() > rows[j].Progress() })
	}
	// Try to keep the cursor on the same GID.
	var keep string
	if s := a.selected(); s != nil {
		keep = s.GID
	}
	a.rows = rows
	if keep != "" {
		for i, r := range a.rows {
			if r.GID == keep {
				a.cursor = i
				break
			}
		}
	}
	if a.cursor >= len(a.rows) {
		a.cursor = len(a.rows) - 1
	}
	if a.cursor < 0 {
		a.cursor = 0
	}
}

// reconcileHistory records finished/errored downloads.
func (a *App) reconcileHistory() {
	for _, st := range a.stopped {
		if st.Status != aria2.StatusComplete && st.Status != aria2.StatusError {
			continue
		}
		if a.hist.Has(st.GID, st.Status) {
			continue
		}
		name := st.Name()
		a.hist.Upsert(history.Entry{
			GID:      st.GID,
			Name:     name,
			URL:      st.PrimaryURI(),
			Dir:      st.Dir,
			Size:     st.TotalLength.Int(),
			Status:   st.Status,
			Category: history.Categorize(name, st.IsTorrent()),
			Added:    time.Now(),
			Finished: time.Now(),
		})
		a.histDirty = true
	}
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.add.setSize(msg.Width, msg.Height)
		a.details.setSize(msg.Width, msg.Height)
		if a.picker != nil {
			a.picker.setSize(msg.Width, msg.Height)
		}
		return a, nil

	case tickMsg:
		a.pruneToasts()
		cmds := []tea.Cmd{tickCmd(a.pollEvery())}
		if a.connected && !a.pollBusy {
			a.pollBusy = true
			cmds = append(cmds, pollCmd(a.client))
			if a.view == viewDetails && a.details.gid != "" {
				cmds = append(cmds, detailCmd(a.client, a.details.gid))
			}
		}
		if !a.connected && time.Since(a.lastRecon) > 2*time.Second {
			a.lastRecon = time.Now()
			cfg := a.cfg
			url := a.rpcURL
			cmds = append(cmds, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				c, err := daemon.Redial(ctx, url, cfg.Secret)
				if err != nil {
					return reconMsg{err: err}
				}
				v, _ := c.GetVersion(ctx)
				return reconMsg{client: c, version: v.Version}
			})
		}
		if a.histDirty {
			a.histDirty = false
			h := a.hist
			cmds = append(cmds, func() tea.Msg { return historySavedMsg{err: h.Save()} })
		}
		return a, tea.Batch(cmds...)

	case pollMsg:
		a.pollBusy = false
		if msg.err != nil {
			if a.connected {
				a.connected = false
				a.pushToast("connection lost — reconnecting…", toastErr)
			}
			return a, nil
		}
		a.stat = msg.stat
		a.active, a.waiting, a.stopped = msg.active, msg.waiting, msg.stopped
		a.speedHist = appendSample(a.speedHist, float64(msg.stat.DownloadSpeed))
		a.upHist = appendSample(a.upHist, float64(msg.stat.UploadSpeed))
		// Per-task lifetime speed history: sample while runnable, freeze when
		// stopped, and only discard it after aria2 removes the task entirely.
		keep := map[string]bool{}
		for _, st := range msg.active {
			a.gidHist[st.GID] = appendTaskSample(a.gidHist[st.GID], float64(st.DownloadSpeed))
			keep[st.GID] = true
		}
		for _, st := range msg.waiting {
			a.gidHist[st.GID] = appendTaskSample(a.gidHist[st.GID], 0)
			keep[st.GID] = true
		}
		for _, st := range msg.stopped {
			keep[st.GID] = true
		}
		for gid := range a.gidHist {
			if !keep[gid] {
				delete(a.gidHist, gid)
			}
		}
		a.buildRows()
		a.reconcileHistory()
		return a, nil

	case reconMsg:
		if msg.err != nil || msg.client == nil {
			return a, nil
		}
		a.client = msg.client
		a.connected = true
		a.version = msg.version
		a.pushToast("reconnected to aria2", toastOK)
		return a, tea.Batch(pollCmd(a.client), listenNotifs(a.client.Notifications()))

	case notifMsg:
		if !msg.ok {
			if a.connected {
				a.connected = false
				a.pushToast("connection lost — reconnecting…", toastErr)
			}
			return a, nil
		}
		name := msg.n.GID
		for _, s := range a.rows {
			if s.GID == msg.n.GID {
				name = truncate(s.Name(), 40)
				break
			}
		}
		switch msg.n.Method {
		case aria2.EventComplete, aria2.EventBTComplete:
			a.pushToast("✓ completed: "+name, toastOK)
		case aria2.EventError:
			a.pushToast("✗ failed: "+name, toastErr)
		case aria2.EventStart:
			a.pushToast("⇣ started: "+name, toastInfo)
		}
		cmds := []tea.Cmd{listenNotifs(a.client.Notifications())}
		if a.connected && !a.pollBusy {
			a.pollBusy = true
			cmds = append(cmds, pollCmd(a.client))
		}
		return a, tea.Batch(cmds...)

	case actionMsg:
		if msg.err != nil {
			a.pushToast("error: "+msg.err.Error(), toastErr)
		} else if msg.toast != "" {
			a.pushToast(msg.toast, msg.level)
		}
		if a.connected && !a.pollBusy {
			a.pollBusy = true
			return a, pollCmd(a.client)
		}
		return a, nil

	case addedMsg:
		if msg.err != nil {
			a.pushToast("add failed: "+msg.err.Error(), toastErr)
		}
		if msg.count > 0 {
			a.pushToast(fmt.Sprintf("added %d download(s)", msg.count), toastOK)
			a.view = viewDownloads
		}
		if a.connected && !a.pollBusy {
			a.pollBusy = true
			return a, pollCmd(a.client)
		}
		return a, nil

	case detailMsg:
		a.details.absorb(msg)
		return a, nil

	case globalOptsMsg:
		a.settings.absorb(msg)
		return a, nil

	case probeMsg:
		a.add.probing = false
		res := msg.res
		a.add.probe = &res
		return a, nil

	case historySavedMsg:
		if msg.err != nil {
			a.pushToast("history save failed: "+msg.err.Error(), toastErr)
		}
		return a, nil

	case tea.MouseMsg:
		return a.handleMouse(msg)

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

// --- mouse ------------------------------------------------------------------

// hitbox is a clickable screen region mapped to an action id.
type hitbox struct {
	x0, y0, x1, y1 int // inclusive bounds
	id             string
}

// hitspec is a clickable region relative to a rendered fragment (e.g. a modal).
type hitspec struct {
	x, y, w, h int
	id         string
}

func (a *App) addHit(x, y, w, h int, id string) {
	if w < 1 || h < 1 {
		return
	}
	a.hits = append(a.hits, hitbox{x0: x, y0: y, x1: x + w - 1, y1: y + h - 1, id: id})
}

func (a *App) hitAt(x, y int) string {
	for i := len(a.hits) - 1; i >= 0; i-- {
		hb := a.hits[i]
		if x >= hb.x0 && x <= hb.x1 && y >= hb.y0 && y <= hb.y1 {
			return hb.id
		}
	}
	return ""
}

// keyFromString synthesizes a KeyMsg so mouse clicks reuse key handlers.
func keyFromString(s string) tea.KeyMsg {
	switch s {
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "ctrl+o":
		return tea.KeyMsg{Type: tea.KeyCtrlO}
	case "ctrl+t":
		return tea.KeyMsg{Type: tea.KeyCtrlT}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func (a *App) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return a, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return a.handleKey(tea.KeyMsg{Type: tea.KeyUp})
	case tea.MouseButtonWheelDown:
		return a.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	case tea.MouseButtonLeft:
		id := a.hitAt(msg.X, msg.Y)
		if id == "" {
			return a, nil
		}
		return a.dispatchClick(id)
	}
	return a, nil
}

func (a *App) dispatchClick(id string) (tea.Model, tea.Cmd) {
	kind, arg, _ := strings.Cut(id, ":")
	switch kind {
	case "modal":
		if arg == "close" {
			a.confirm = nil
			a.picker = nil
		}
		return a, nil
	case "confirm":
		if a.confirm != nil {
			c := a.confirm
			a.confirm = nil
			if arg == "yes" {
				return a, c.action()
			}
		}
		return a, nil
	case "prow":
		if a.picker != nil {
			return a.pickerRowClick(atoiSafe(arg))
		}
		return a, nil
	case "pbtn":
		if a.picker != nil {
			return a.pickerButtonClick(arg)
		}
		return a, nil
	case "tab":
		n := atoiSafe(arg)
		a.view = viewDownloads
		a.tab = n
		a.cursor, a.scroll = 0, 0
		a.buildRows()
		return a, nil
	case "nav":
		switch arg {
		case "add":
			return a, a.gotoAdd()
		case "history":
			a.gotoHistory()
		case "files":
			a.gotoFiles()
		case "settings":
			return a, a.gotoSettings()
		}
		return a, nil
	case "addfld":
		if a.view == viewAdd {
			n := atoiSafe(arg)
			m := &a.add
			wasFocused := m.focus == n
			m.setFocusIndex(n)
			if f := m.field(); f != nil {
				switch f.kind {
				case kindCheck:
					if f.value == "true" {
						f.value = ""
					} else {
						f.value = "true"
					}
				case kindSelect:
					if wasFocused { // second click cycles the value
						return a.handleKey(tea.KeyMsg{Type: tea.KeyRight})
					}
				}
			}
			return a, textinput.Blink
		}
		return a, nil
	case "row":
		n := atoiSafe(arg)
		if n >= 0 && n < len(a.rows) {
			if a.cursor == n {
				// second click on the selection opens details
				return a.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
			}
			a.cursor = n
		}
		return a, nil
	case "fbrow":
		n := atoiSafe(arg)
		if a.view == viewFiles && n >= 0 && n < len(a.files.entries) {
			if a.files.cursor == n {
				return a.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
			}
			a.files.cursor = n
		}
		return a, nil
	case "hrow":
		n := atoiSafe(arg)
		if a.view == viewHistory && n >= 0 && n < len(a.historyV.rows) {
			if a.historyV.cursor == n {
				return a.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
			}
			a.historyV.cursor = n
		}
		return a, nil
	case "srow":
		n := atoiSafe(arg)
		if a.view == viewSettings && n >= 0 && n < len(a.settings.defs()) {
			if a.settings.cursor == n && !a.settings.editing {
				return a.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
			}
			a.settings.cursor = n
		}
		return a, nil
	case "stab":
		n := atoiSafe(arg)
		if a.view == viewSettings && n >= 0 && n < len(settingTabs) {
			a.settings.tab = n
			a.settings.cursor, a.settings.scroll = 0, 0
			a.settings.editing, a.settings.filtering = false, false
		}
		return a, nil
	case "dtab":
		if a.view == viewDetails {
			a.details.tab = atoiSafe(arg)
			a.details.cursor, a.details.scroll = 0, 0
		}
		return a, nil
	case "key":
		return a.handleKey(keyFromString(arg))
	}
	return a, nil
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// appendSample pushes v onto a ring of at most 240 samples.
func appendSample(s []float64, v float64) []float64 {
	s = append(s, v)
	if len(s) > 240 {
		s = s[len(s)-240:]
	}
	return s
}

// appendTaskSample keeps a longer bounded history for per-download lifetime
// charts. At the default poll rate this represents roughly 14 minutes.
func appendTaskSample(s []float64, v float64) []float64 {
	s = append(s, v)
	if len(s) > 1200 {
		s = s[len(s)-1200:]
	}
	return s
}

// --- navigation helpers (shared by keys and mouse) ---------------------------

func (a *App) gotoAdd() tea.Cmd {
	a.add.reset(a.cfg)
	a.view = viewAdd
	return a.add.focusCmd()
}

func (a *App) gotoHistory() {
	a.historyV.reload(a.hist)
	a.view = viewHistory
}

func (a *App) gotoFiles() {
	a.files.ensure(a.cfg.DownloadDir)
	a.view = viewFiles
}

func (a *App) gotoSettings() tea.Cmd {
	a.view = viewSettings
	a.settings.loading = true
	client := a.client
	return func() tea.Msg {
		ctx, cancel := rpcCtx()
		defer cancel()
		opts, err := client.GetGlobalOption(ctx)
		return globalOptsMsg{opts: opts, err: err}
	}
}

// handleKey routes keystrokes to overlays or the active view.
func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Modal overlays swallow everything.
	if a.confirm != nil {
		return a.updateConfirm(msg)
	}
	if a.picker != nil {
		return a.updatePicker(msg)
	}

	// ctrl+c always quits (daemon keeps running).
	if msg.String() == "ctrl+c" {
		return a, a.quit(false)
	}

	// Inline search input on downloads view.
	if a.searching {
		switch msg.String() {
		case "enter":
			a.searching = false
			a.filter = a.searchBuf
			a.buildRows()
		case "esc":
			a.searching = false
			a.searchBuf = ""
			a.filter = ""
			a.buildRows()
		case "backspace":
			if len(a.searchBuf) > 0 {
				a.searchBuf = a.searchBuf[:len(a.searchBuf)-1]
				a.filter = a.searchBuf
				a.buildRows()
			}
		default:
			if msg.Type == tea.KeyRunes {
				a.searchBuf += string(msg.Runes)
				a.filter = a.searchBuf
				a.buildRows()
			}
		}
		return a, nil
	}

	// View-specific input first when the view owns text fields.
	switch a.view {
	case viewAdd:
		return a.updateAdd(msg)
	case viewSettings:
		return a.updateSettings(msg)
	case viewHistory:
		return a.updateHistory(msg)
	case viewDetails:
		return a.updateDetails(msg)
	case viewFiles:
		return a.updateFiles(msg)
	case viewHelp:
		switch msg.String() {
		case "q", "esc", "?":
			a.view = viewDownloads
		case "Q":
			a.confirmShutdown()
		}
		return a, nil
	}

	// Downloads view + global keys.
	return a.updateDownloads(msg)
}

// tabsHeight is the rendered height of the tab bar — big Surge-style
// buttons on wide terminals, a compact strip on narrow ones.
func (a *App) tabsHeight() int {
	if a.width >= 110 {
		return 3
	}
	return 1
}

// bodyTop is the first row of the body area (header + tab bar above it).
func (a *App) bodyTop() int { return 1 + a.tabsHeight() }

// View implements tea.Model.
func (a *App) View() string {
	if a.width == 0 {
		return "loading…"
	}
	a.hits = a.hits[:0]

	header := a.renderHeader()
	tabs := a.renderTabs()

	toastLine := a.renderToastLine()
	extra := 0
	if toastLine != "" {
		extra = 1
	}

	bodyH := a.height - a.bodyTop() - 1 - extra
	if bodyH < 3 {
		bodyH = 3
	}

	var body string
	switch a.view {
	case viewAdd:
		body = a.viewAdd(bodyH)
	case viewDetails:
		body = a.viewDetails(bodyH)
	case viewHistory:
		body = a.viewHistory(bodyH)
	case viewSettings:
		body = a.viewSettings(bodyH)
	case viewHelp:
		body = a.viewHelp(bodyH)
	case viewFiles:
		body = a.viewFiles(bodyH)
	default:
		body = a.viewDownloads(bodyH)
	}
	body = lipgloss.NewStyle().Height(bodyH).MaxHeight(bodyH).Render(body)

	footer := a.renderFooter(a.bodyTop() + bodyH + extra)

	parts := []string{header, tabs, body}
	if toastLine != "" {
		parts = append(parts, toastLine)
	}
	parts = append(parts, footer)
	screen := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Modal overlays are centered on top of everything.
	if a.confirm != nil {
		modal, specs := a.confirm.render()
		return a.overlay(modal, specs)
	}
	if a.picker != nil {
		modal, specs := a.picker.render()
		return a.overlay(modal, specs)
	}
	return screen
}

// overlay centers a modal above a dimmed backdrop and rebinds all mouse
// hitboxes to the modal (clicks outside close it).
func (a *App) overlay(modal string, specs []hitspec) string {
	a.hits = a.hits[:0]
	a.addHit(0, 0, a.width, a.height, "modal:close")
	mw, mh := lipgloss.Width(modal), lipgloss.Height(modal)
	// lipgloss.Place puts the smaller half of an odd gap first (floor).
	x0 := (a.width - mw) / 2
	y0 := (a.height - mh) / 2
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	// The modal body itself swallows clicks (so outside-click closes, inside doesn't).
	a.addHit(x0, y0, mw, mh, "modal:body")
	for _, s := range specs {
		a.addHit(x0+s.x, y0+s.y, s.w, s.h, s.id)
	}
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars("░"), lipgloss.WithWhitespaceForeground(lipgloss.Color("#101120")))
}

func (a *App) renderHeader() string {
	logo := styleLogo.Render("⬡ tidefetch")

	down := styleDownArr.Render("▼ ") + styleText.Render(humanSpeed(a.stat.DownloadSpeed.Int()))
	up := styleUpArr.Render("▲ ") + styleText.Render(humanSpeed(a.stat.UploadSpeed.Int()))
	spark := styleSpark.Render(sparkline(a.speedHist, 24))

	counts := styleDim.Render(fmt.Sprintf("active %d · queued %d · stopped %d",
		a.stat.NumActive.Int(), a.stat.NumWaiting.Int(), a.stat.NumStopped.Int()))

	var conn string
	if a.connected {
		conn = styleGood.Render("● ") + styleDim.Render("aria2 "+a.version)
	} else {
		conn = styleBad.Render("○ reconnecting…")
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", down, "  ", up, "  ", spark, "  ", counts)
	gap := a.width - lipgloss.Width(left) - lipgloss.Width(conn) - 1
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + conn
}

var tabNames = []string{"All", "Active", "Queued", "Finished"}

// tabCounts returns the number of downloads behind each tab.
func (a *App) tabCounts() [4]int {
	return [4]int{
		len(a.active) + len(a.waiting) + len(a.stopped),
		len(a.active),
		len(a.waiting),
		len(a.stopped),
	}
}

func (a *App) renderTabs() string {
	if a.tabsHeight() == 1 {
		return a.renderTabsCompact()
	}
	counts := a.tabCounts()

	type seg struct {
		label  string
		id     string
		active bool
	}
	segs := make([]seg, 0, 9)
	for i, name := range tabNames {
		segs = append(segs, seg{
			label:  fmt.Sprintf("%d  %s (%d)", i+1, name, counts[i]),
			id:     fmt.Sprintf("tab:%d", i),
			active: a.view == viewDownloads && a.tab == i,
		})
	}
	segs = append(segs,
		seg{label: "", id: ""}, // spacer
		seg{label: "＋ Add", id: "nav:add", active: a.view == viewAdd},
		seg{label: "▤ Files", id: "nav:files", active: a.view == viewFiles},
		seg{label: "⟲ History", id: "nav:history", active: a.view == viewHistory},
		seg{label: "⚙ Settings", id: "nav:settings", active: a.view == viewSettings},
	)

	blocks := make([]string, 0, len(segs))
	x := 1 // leading space
	blocks = append(blocks, " ")
	for _, s := range segs {
		if s.id == "" {
			blocks = append(blocks, "  ")
			x += 2
			continue
		}
		st := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 2)
		if s.active {
			st = st.BorderForeground(cAccent).Foreground(cBright).Bold(true)
		} else {
			st = st.BorderForeground(cFaint).Foreground(cDim)
		}
		chip := st.Render(s.label)
		w := lipgloss.Width(chip)
		a.addHit(x, 1, w, 3, s.id)
		x += w
		blocks = append(blocks, chip)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
}

// renderTabsCompact is the single-line tab strip for narrow terminals.
func (a *App) renderTabsCompact() string {
	counts := a.tabCounts()
	var b strings.Builder
	x := 0
	for i, name := range tabNames {
		label := fmt.Sprintf("%d %s (%d)", i+1, name, counts[i])
		var seg string
		if a.view == viewDownloads && a.tab == i {
			seg = styleTabActive.Render(label)
		} else {
			seg = styleTab.Render(label)
		}
		a.addHit(x, 1, lipgloss.Width(seg), 1, fmt.Sprintf("tab:%d", i))
		x += lipgloss.Width(seg)
		b.WriteString(seg)
	}
	sep := styleFaint.Render("│")
	b.WriteString(sep)
	x += lipgloss.Width(sep)
	extra := func(v View, label, id string) {
		var seg string
		if a.view == v {
			seg = styleTabActive.Render(label)
		} else {
			seg = styleTab.Render(label)
		}
		a.addHit(x, 1, lipgloss.Width(seg), 1, id)
		x += lipgloss.Width(seg)
		b.WriteString(seg)
	}
	extra(viewAdd, "＋ Add", "nav:add")
	extra(viewFiles, "▤ Files", "nav:files")
	extra(viewHistory, "⟲ History", "nav:history")
	extra(viewSettings, "⚙ Settings", "nav:settings")

	right := ""
	if a.filter != "" || a.searching {
		right = styleAccent.Render(" /" + a.filter)
		if a.searching {
			right += styleAccent.Render("▌")
		}
	}
	if a.sortMode != 0 {
		names := []string{"", "name", "size", "speed", "progress"}
		right += styleDim.Render("  sort:" + names[a.sortMode])
	}
	line := b.String()
	gap := a.width - lipgloss.Width(line) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return line + strings.Repeat(" ", gap) + right
}

// renderToastLine shows the most recent toast on its own line (right aligned).
func (a *App) renderToastLine() string {
	if len(a.toasts) == 0 {
		return ""
	}
	t := a.toasts[len(a.toasts)-1]
	var s lipgloss.Style
	switch t.level {
	case toastOK:
		s = styleToastGood
	case toastErr:
		s = styleToastBad
	default:
		s = styleToastInfo
	}
	msg := s.Render(truncate(t.text, a.width-4))
	gap := a.width - lipgloss.Width(msg) - 1
	if gap < 0 {
		gap = 0
	}
	return strings.Repeat(" ", gap) + msg
}

// footerButton is one clickable chip in the bottom bar.
type footerButton struct {
	key   string // synthesized key, also the visible shortcut
	label string
}

func (a *App) footerButtons() []footerButton {
	switch a.view {
	case viewDownloads:
		return []footerButton{
			{"a", "add"}, {"space", "pause"}, {"enter", "details"}, {"x", "remove"},
			{"r", "retry"}, {"/", "search"}, {"t", "panel"}, {"S", "sort"},
			{"h", "history"}, {"s", "settings"}, {"?", "help"}, {"q", "quit"},
		}
	case viewAdd:
		return []footerButton{
			{"tab", "next field"}, {"ctrl+k", "check link"}, {"ctrl+a", "advanced"},
			{"ctrl+o", "browse dir"}, {"ctrl+t", "torrent file"},
			{"ctrl+s", "start"}, {"esc", "cancel"},
		}
	case viewDetails:
		return []footerButton{
			{"tab", "next panel"}, {"enter", "edit option"}, {"space", "toggle file"},
			{"+", "limit +"}, {"-", "limit −"}, {"p", "pause"},
			{"x", "remove"}, {"D", "delete files"}, {"o", "open"}, {"y", "copy"}, {"esc", "back"},
		}
	case viewHistory:
		return []footerButton{
			{"enter", "re-download"}, {"/", "search"}, {"c", "category"}, {"x", "delete"},
			{"o", "open"}, {"C", "clear all"}, {"esc", "back"},
		}
	case viewSettings:
		return []footerButton{
			{"left", "prev category"}, {"right", "next category"},
			{"enter", "edit / save"}, {"/", "filter all"}, {"r", "reload"}, {"esc", "back"},
		}
	case viewFiles:
		return []footerButton{
			{"enter", "open"}, {"backspace", "up"}, {"o", "reveal"}, {"x", "delete"},
			{"n", "new folder"}, {"~", "home"}, {"/", "root"},
			{"d", "downloads"}, {".", "hidden"}, {"esc", "back"},
		}
	default: // help
		return []footerButton{{"esc", "back"}, {"q", "quit"}}
	}
}

// renderFooter paints the clickable button bar; y is its screen row.
func (a *App) renderFooter(y int) string {
	var b strings.Builder
	x := 0
	shown := func(k string) string {
		switch k {
		case "space":
			return "␣"
		case "enter":
			return "↵"
		default:
			return k
		}
	}
	for _, btn := range a.footerButtons() {
		seg := styleBtnKey.Render(shown(btn.key)) + styleBtn.Render(btn.label)
		w := lipgloss.Width(seg)
		if x+w+1 > a.width {
			break
		}
		a.addHit(x, y, w, 1, "key:"+btn.key)
		b.WriteString(seg)
		b.WriteString(" ")
		x += w + 1
	}
	// right side: connection state hint
	var right string
	if !a.connected {
		right = styleBad.Render("◌ reconnecting ")
	} else if a.view == viewDownloads {
		right = styleFaint.Render("click rows · wheel scrolls · click tabs ")
	}
	gap := a.width - x - lipgloss.Width(right)
	if gap < 0 {
		right = ""
		gap = maxInt(0, a.width-x)
	}
	return b.String() + strings.Repeat(" ", gap) + right
}

// --- shared small helpers used by view files --------------------------------

// confirmAction arms the confirmation modal.
func (a *App) confirmAction(title, body string, action func() tea.Cmd) {
	a.confirm = &confirmModel{title: title, body: body, action: action}
}

// confirmShutdown arms the "quit + stop daemon" modal.
func (a *App) confirmShutdown() {
	a.confirmAction("Shut down daemon?",
		"This stops aria2c and all transfers.\nThe session is saved and resumes next start.",
		func() tea.Cmd { return a.quit(true) })
}

// removeFilesOf deletes downloaded files (and .aria2 control file) from disk.
func removeFilesOf(st aria2.Status) error {
	var firstErr error
	for _, f := range st.Files {
		if f.Path == "" || strings.HasPrefix(f.Path, "[METADATA]") {
			continue
		}
		if err := os.Remove(f.Path); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
		if err := os.Remove(f.Path + ".aria2"); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	// Torrent multi-file downloads live in a directory named after the torrent.
	if st.BitTorrent != nil && st.BitTorrent.Info.Name != "" {
		root := filepath.Join(st.Dir, st.BitTorrent.Info.Name)
		if fi, err := os.Stat(root); err == nil && fi.IsDir() {
			if err := os.RemoveAll(root); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
