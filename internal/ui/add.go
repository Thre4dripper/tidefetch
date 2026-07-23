package ui

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/turbostart/aria2c-tui/internal/config"
	"github.com/turbostart/aria2c-tui/pkg/aria2"
)

const (
	fldURLs = iota
	fldDir
	fldOut
	fldSplit
	fldConn
	fldPause
	fldCount
)

// addModel is the "new download" form.
type addModel struct {
	urls        textarea.Model
	dir         textinput.Model
	out         textinput.Model
	split       textinput.Model
	conn        textinput.Model
	pauseStart  bool
	torrentPath string
	focus       int
	width       int
	height      int
}

func newAddModel(cfg *config.Config) addModel {
	ta := textarea.New()
	ta.Placeholder = "one download per line — http(s)://, ftp://, sftp://, magnet:…\nmirrors of the same file: separate with spaces on one line"
	ta.ShowLineNumbers = false
	ta.SetHeight(5)
	ta.CharLimit = 0

	mk := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.Prompt = ""
		ti.CharLimit = 0
		return ti
	}
	m := addModel{
		urls:  ta,
		dir:   mk("download directory"),
		out:   mk("optional — rename saved file (single URL only)"),
		split: mk("16"),
		conn:  mk("16"),
	}
	m.reset(cfg)
	return m
}

func (m *addModel) reset(cfg *config.Config) {
	m.urls.SetValue("")
	m.dir.SetValue(cfg.DownloadDir)
	m.out.SetValue("")
	m.split.SetValue(cfg.DefaultSplit)
	m.conn.SetValue(cfg.DefaultMaxConn)
	m.pauseStart = false
	m.torrentPath = ""
	m.focus = fldURLs
	m.syncFocus()
}

func (m *addModel) setSize(w, h int) {
	m.width, m.height = w, h
	inner := w - 10
	if inner < 20 {
		inner = 20
	}
	if inner > 100 {
		inner = 100
	}
	m.urls.SetWidth(inner)
	m.dir.Width = inner - 2
	m.out.Width = inner - 2
	m.split.Width = 8
	m.conn.Width = 8
}

func (m *addModel) syncFocus() {
	m.urls.Blur()
	m.dir.Blur()
	m.out.Blur()
	m.split.Blur()
	m.conn.Blur()
	switch m.focus {
	case fldURLs:
		m.urls.Focus()
	case fldDir:
		m.dir.Focus()
	case fldOut:
		m.out.Focus()
	case fldSplit:
		m.split.Focus()
	case fldConn:
		m.conn.Focus()
	}
}

func (m *addModel) focusCmd() tea.Cmd { return textarea.Blink }

// updateAdd handles input while the Add form is visible.
func (a *App) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m := &a.add

	switch msg.String() {
	case "esc":
		a.view = viewDownloads
		return a, nil
	case "tab", "shift+tab", "down", "up":
		// Up/down only leave the textarea from its edges; simplest: tab cycles.
		if msg.String() == "down" && m.focus == fldURLs {
			break // let textarea handle line movement
		}
		if msg.String() == "up" && m.focus == fldURLs {
			break
		}
		if msg.String() == "tab" || msg.String() == "down" {
			m.focus = (m.focus + 1) % fldCount
		} else {
			m.focus = (m.focus - 1 + fldCount) % fldCount
		}
		m.syncFocus()
		return a, textinput.Blink
	case "ctrl+o":
		start := m.dir.Value()
		a.picker = newDirPicker("Choose download directory", start, false, nil, func(path string) tea.Cmd {
			a.add.dir.SetValue(path)
			return nil
		})
		a.picker.setSize(a.width, a.height)
		return a, nil
	case "ctrl+t":
		home := a.cfg.DownloadDir
		a.picker = newDirPicker("Pick a .torrent / .metalink file", home, true,
			[]string{".torrent", ".metalink", ".meta4"}, func(path string) tea.Cmd {
				a.add.torrentPath = path
				return nil
			})
		a.picker.setSize(a.width, a.height)
		return a, nil
	case "ctrl+s":
		return a, a.submitAdd()
	case " ", "enter":
		if m.focus == fldPause {
			m.pauseStart = !m.pauseStart
			return a, nil
		}
		if msg.String() == "enter" && m.focus != fldURLs {
			return a, a.submitAdd()
		}
	}

	var cmd tea.Cmd
	switch m.focus {
	case fldURLs:
		m.urls, cmd = m.urls.Update(msg)
	case fldDir:
		m.dir, cmd = m.dir.Update(msg)
	case fldOut:
		m.out, cmd = m.out.Update(msg)
	case fldSplit:
		m.split, cmd = m.split.Update(msg)
	case fldConn:
		m.conn, cmd = m.conn.Update(msg)
	}
	return a, cmd
}

// submitAdd queues everything entered in the form.
func (a *App) submitAdd() tea.Cmd {
	m := &a.add
	var lines []string
	for _, l := range strings.Split(m.urls.Value(), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	torrent := m.torrentPath
	if len(lines) == 0 && torrent == "" {
		a.pushToast("nothing to add — paste a URL or pick a torrent", toastErr)
		return nil
	}

	opts := aria2.Options{}
	if v := strings.TrimSpace(m.dir.Value()); v != "" {
		opts[aria2.OptDir] = v
	}
	if v := strings.TrimSpace(m.split.Value()); v != "" {
		opts[aria2.OptSplit] = v
	}
	if v := strings.TrimSpace(m.conn.Value()); v != "" {
		opts[aria2.OptMaxConnectionPerServer] = v
	}
	if m.pauseStart {
		opts[aria2.OptPause] = "true"
	}
	out := strings.TrimSpace(m.out.Value())

	client := a.client
	return func() tea.Msg {
		ctx, cancel := rpcCtx()
		defer cancel()
		count := 0
		var lastErr error

		for _, line := range lines {
			mirrors := strings.Fields(line)
			// Local .torrent / .metalink paths are allowed straight in the textarea too.
			if len(mirrors) == 1 && isLocalMeta(mirrors[0]) {
				if err := addLocalMeta(ctx, client, mirrors[0], opts); err != nil {
					lastErr = err
				} else {
					count++
				}
				continue
			}
			o := cloneOpts(opts)
			if out != "" && len(lines) == 1 {
				o[aria2.OptOut] = out
			}
			if _, err := client.AddURI(ctx, mirrors, o); err != nil {
				lastErr = err
			} else {
				count++
			}
		}
		if torrent != "" {
			if err := addLocalMeta(ctx, client, torrent, cloneOpts(opts)); err != nil {
				lastErr = err
			} else {
				count++
			}
		}
		return addedMsg{count: count, err: lastErr}
	}
}

func isLocalMeta(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".torrent", ".metalink", ".meta4":
		return !strings.Contains(p, "://")
	}
	return false
}

func addLocalMeta(ctx context.Context, c *aria2.Client, path string, opts aria2.Options) error {
	if strings.ToLower(filepath.Ext(path)) == ".torrent" {
		_, err := c.AddTorrentFile(ctx, path, opts)
		return err
	}
	_, err := c.AddMetalinkFile(ctx, path, opts)
	return err
}

func cloneOpts(o aria2.Options) aria2.Options {
	c := aria2.Options{}
	for k, v := range o {
		c[k] = v
	}
	return c
}

// viewAdd renders the form with clickable fields and buttons.
func (a *App) viewAdd(h int) string {
	m := &a.add
	panelW := a.width - 4
	if panelW > 108 {
		panelW = 108
	}
	innerW := panelW - 2

	label := func(idx int, text string) string {
		if m.focus == idx {
			return styleFieldFocus.Render("▍" + text)
		}
		return styleInputLabel.Render(" " + text)
	}
	// labelBtn right-aligns a button chip on a label line.
	labelBtn := func(lbl, btn string) string {
		gap := innerW - lipgloss.Width(lbl) - lipgloss.Width(btn)
		if gap < 1 {
			gap = 1
		}
		return lbl + strings.Repeat(" ", gap) + btn
	}

	type spec struct {
		line, x, w, h int
		id            string
	}
	var specs []spec
	var lines []string
	add := func(s string) int {
		lines = append(lines, s)
		return len(lines) - 1
	}

	// --- title + torrent button
	torrentBtn := styleBadge.Render(" ⧉ torrent file ")
	ln := add(labelBtn(styleTitle.Render("Add downloads"), torrentBtn))
	specs = append(specs, spec{ln, innerW - lipgloss.Width(torrentBtn), lipgloss.Width(torrentBtn), 1, "key:ctrl+t"})
	add("")

	// --- URLs textarea
	ln = add(label(fldURLs, "URLs / magnets / local .torrent paths"))
	specs = append(specs, spec{ln, 0, innerW, 1, "addfld:0"})
	taTop := add(strings.Split(m.urls.View(), "\n")[0])
	for _, l := range strings.Split(m.urls.View(), "\n")[1:] {
		add(l)
	}
	specs = append(specs, spec{taTop, 0, innerW, m.urls.Height(), "addfld:0"})
	add("")

	// --- directory + browse button
	browseBtn := styleBadge.Render(" ⧉ browse ")
	ln = add(labelBtn(label(fldDir, "Save to"), browseBtn))
	specs = append(specs,
		spec{ln, innerW - lipgloss.Width(browseBtn), lipgloss.Width(browseBtn), 1, "key:ctrl+o"},
		spec{ln, 0, innerW - lipgloss.Width(browseBtn) - 1, 1, "addfld:1"})
	ln = add(" " + m.dir.View())
	specs = append(specs, spec{ln, 0, innerW, 1, "addfld:1"})
	if free, total, err := diskUsage(strings.TrimSpace(m.dir.Value())); err == nil && total > 0 {
		add(" " + styleGood.Render("●") + styleDim.Render(" "+humanBytes(free)+" free on this volume"))
	} else {
		add(" " + styleWarn.Render("○") + styleDim.Render(" folder will be created on start"))
	}
	add("")

	// --- filename override
	ln = add(label(fldOut, "Filename override"))
	specs = append(specs, spec{ln, 0, innerW, 1, "addfld:2"})
	ln = add(" " + m.out.View())
	specs = append(specs, spec{ln, 0, innerW, 1, "addfld:2"})
	add("")

	// --- split / conn
	const colW = 18
	ln = add(padRight(label(fldSplit, "Split"), colW) + label(fldConn, "Conn/server"))
	specs = append(specs,
		spec{ln, 0, colW - 1, 2, "addfld:3"},
		spec{ln, colW, innerW - colW, 2, "addfld:4"})
	add(padRight(" "+m.split.View(), colW) + " " + m.conn.View())
	add("")

	// --- pause checkbox
	check := "☐"
	if m.pauseStart {
		check = "☑"
	}
	pauseLabel := check + " add paused"
	if m.focus == fldPause {
		ln = add(styleFieldFocus.Render("▍" + pauseLabel))
	} else {
		ln = add(styleInputLabel.Render(" " + pauseLabel))
	}
	specs = append(specs, spec{ln, 0, lipgloss.Width(pauseLabel) + 2, 1, "addfld:5"})

	if m.torrentPath != "" {
		add("")
		add(styleAccent2.Render(" ⧉ "+truncate(m.torrentPath, innerW-24)) + styleDim.Render("  (queued on submit)"))
	}
	add("")

	// --- action buttons
	startBtn := styleToastGood.Render(" ▶ start download ")
	cancelBtn := styleBadge.Render(" esc cancel ")
	ln = add(startBtn + "  " + cancelBtn)
	specs = append(specs,
		spec{ln, 0, lipgloss.Width(startBtn), 1, "key:ctrl+s"},
		spec{ln, lipgloss.Width(startBtn) + 2, lipgloss.Width(cancelBtn), 1, "key:esc"})

	content := strings.Join(lines, "\n")
	panel := stylePanelFocus.Width(panelW).Render(content)

	// Register hitboxes at absolute screen coordinates.
	x0 := (a.width - lipgloss.Width(panel)) / 2 // lipgloss.Place floors odd gaps
	if x0 < 0 {
		x0 = 0
	}
	const padX, padY = 2, 1 // panel border + horizontal padding
	for _, s := range specs {
		a.addHit(x0+padX+s.x, bodyTop+padY+s.line, s.w, s.h, s.id)
	}
	return lipgloss.Place(a.width, h, lipgloss.Center, lipgloss.Top, panel)
}
