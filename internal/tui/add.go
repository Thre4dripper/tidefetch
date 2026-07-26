package tui

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/turbostart/tidefetch/internal/config"
	"github.com/turbostart/tidefetch/pkg/aria2"
)

// fieldKind distinguishes how an option row edits.
type fieldKind int

const (
	kindText fieldKind = iota
	kindSelect
	kindCheck
)

// addField is one option row in the Add form.
type addField struct {
	key         string // aria2 option key ("" = handled specially)
	label       string
	kind        fieldKind
	choices     []string // kindSelect; "" renders as "default"
	value       string   // current value; kindCheck: "true"/""
	placeholder string
	advanced    bool
}

// addModel is the "new download" form: a URL textarea plus a data-driven
// list of aria2 options (basic + advanced).
type addModel struct {
	urls         textarea.Model
	input        textinput.Model // shared editor for the focused text field
	fields       []addField
	focus        int // 0 = urls, 1..len(fields) = fields[focus-1]
	showAdvanced bool
	torrentPath  string
	probe        *probeResult
	probing      bool
	width        int
	height       int
}

func defaultAddFields(cfg *config.Config) []addField {
	return []addField{
		{key: "dir", label: "Save to", kind: kindText, value: cfg.DownloadDir},
		{key: aria2.OptOut, label: "Filename override", kind: kindText, placeholder: "(auto)"},
		{key: aria2.OptSplit, label: "Split (segments)", kind: kindSelect,
			choices: []string{"", "1", "2", "4", "8", "16", "32", "64"}, value: cfg.DefaultSplit},
		{key: aria2.OptMaxConnectionPerServer, label: "Connections per server", kind: kindSelect,
			choices: []string{"", "1", "2", "4", "8", "16"}, value: cfg.DefaultMaxConn},
		{key: aria2.OptPause, label: "Start paused", kind: kindCheck},

		{key: aria2.OptContinue, label: "Resume partial file (continue)", kind: kindSelect,
			choices: []string{"", "true", "false"}, advanced: true},
		{key: "file-allocation", label: "File allocation", kind: kindSelect,
			choices: []string{"", "none", "prealloc", "trunc", "falloc"}, advanced: true},
		{key: aria2.OptMaxDownloadLimit, label: "Download limit (this task)", kind: kindSelect,
			choices: []string{"", "256K", "512K", "1M", "2M", "5M", "10M"}, advanced: true},
		{key: "max-tries", label: "Max tries", kind: kindSelect,
			choices: []string{"", "1", "3", "5", "10", "0"}, advanced: true},
		{key: "retry-wait", label: "Retry wait (seconds)", kind: kindSelect,
			choices: []string{"", "0", "5", "10", "30", "60"}, advanced: true},
		{key: "check-integrity", label: "Verify checksum on finish", kind: kindCheck, advanced: true},
		{key: aria2.OptChecksum, label: "Checksum", kind: kindText,
			placeholder: "sha-256=<hex> · md5=<hex>", advanced: true},
		{key: aria2.OptUserAgent, label: "User agent", kind: kindText,
			placeholder: "(daemon default)", advanced: true},
		{key: aria2.OptReferer, label: "Referer", kind: kindText,
			placeholder: "https://…  (* = mirror the URL)", advanced: true},
		{key: aria2.OptHeader, label: "Extra header", kind: kindText,
			placeholder: "Authorization: Bearer …", advanced: true},
		{key: aria2.OptAllProxy, label: "Proxy (this task)", kind: kindText,
			placeholder: "http://user:pass@host:port", advanced: true},
		{key: aria2.OptSeedRatio, label: "BT seed ratio", kind: kindSelect,
			choices: []string{"", "0.0", "0.5", "1.0", "2.0", "5.0"}, advanced: true},
	}
}

func newAddModel(cfg *config.Config) addModel {
	ta := textarea.New()
	ta.Placeholder = "one download per line — http(s)://, ftp://, sftp://, magnet:…\nmirrors of the same file: separate with spaces on one line"
	ta.ShowLineNumbers = false
	ta.SetHeight(4)
	ta.CharLimit = 0

	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 0

	m := addModel{urls: ta, input: ti}
	m.reset(cfg)
	return m
}

func (m *addModel) reset(cfg *config.Config) {
	m.urls.SetValue("")
	m.fields = defaultAddFields(cfg)
	m.focus = 0
	m.showAdvanced = false
	m.torrentPath = ""
	m.probe = nil
	m.probing = false
	m.syncFocus()
}

func (m *addModel) setSize(w, h int) {
	m.width, m.height = w, h
	inner := w - 12
	if inner < 20 {
		inner = 20
	}
	if inner > 96 {
		inner = 96
	}
	m.urls.SetWidth(inner)
	m.input.Width = inner - 30
}

// visibleFields returns indices into m.fields honoring the advanced toggle.
func (m *addModel) visibleFields() []int {
	out := make([]int, 0, len(m.fields))
	for i, f := range m.fields {
		if f.advanced && !m.showAdvanced {
			continue
		}
		out = append(out, i)
	}
	return out
}

// field returns the addField the current focus points at (nil = urls).
func (m *addModel) field() *addField {
	vis := m.visibleFields()
	if m.focus <= 0 || m.focus > len(vis) {
		return nil
	}
	return &m.fields[vis[m.focus-1]]
}

// fieldByKey finds a field by aria2 option key.
func (m *addModel) fieldByKey(key string) *addField {
	for i := range m.fields {
		if m.fields[i].key == key {
			return &m.fields[i]
		}
	}
	return nil
}

// commitInput writes the shared text editor back into the focused field.
func (m *addModel) commitInput() {
	if f := m.field(); f != nil && f.kind == kindText {
		f.value = m.input.Value()
	}
}

// syncFocus (re)wires the textarea / shared input to the focused element.
func (m *addModel) syncFocus() {
	m.urls.Blur()
	m.input.Blur()
	if m.focus == 0 {
		m.urls.Focus()
		return
	}
	if f := m.field(); f != nil && f.kind == kindText {
		m.input.SetValue(f.value)
		m.input.Placeholder = f.placeholder
		m.input.CursorEnd()
		m.input.Focus()
	}
}

func (m *addModel) focusCmd() tea.Cmd { return textarea.Blink }

// moveFocus advances focus by delta over urls + visible fields.
func (m *addModel) moveFocus(delta int) {
	m.commitInput()
	n := len(m.visibleFields()) + 1
	m.focus = (m.focus + delta + n) % n
	m.syncFocus()
}

// setFocusIndex focuses a specific visible slot (0 = urls).
func (m *addModel) setFocusIndex(idx int) {
	m.commitInput()
	n := len(m.visibleFields()) + 1
	if idx < 0 || idx >= n {
		return
	}
	m.focus = idx
	m.syncFocus()
}

// updateAdd handles input while the Add form is visible.
func (a *App) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m := &a.add

	switch msg.String() {
	case "esc":
		m.commitInput()
		a.view = viewDownloads
		return a, nil
	case "tab", "down":
		if msg.String() == "down" && m.focus == 0 {
			break // let the textarea move between lines
		}
		m.moveFocus(1)
		return a, textinput.Blink
	case "shift+tab", "up":
		if msg.String() == "up" && m.focus == 0 {
			break
		}
		m.moveFocus(-1)
		return a, textinput.Blink
	case "ctrl+a":
		m.showAdvanced = !m.showAdvanced
		if m.field() == nil {
			m.focus = 0
			m.syncFocus()
		}
		return a, nil
	case "ctrl+k":
		return a, a.startProbe()
	case "ctrl+o":
		m.commitInput()
		start := ""
		if f := m.fieldByKey("dir"); f != nil {
			start = f.value
		}
		a.picker = newDirPicker("Choose download directory", start, false, nil, func(path string) tea.Cmd {
			if f := a.add.fieldByKey("dir"); f != nil {
				f.value = path
				if a.add.field() == f {
					a.add.input.SetValue(path)
				}
			}
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
	case " ":
		if f := m.field(); f != nil && f.kind == kindCheck {
			if f.value == "true" {
				f.value = ""
			} else {
				f.value = "true"
			}
			return a, nil
		}
	case "left", "right":
		if f := m.field(); f != nil && f.kind == kindSelect {
			delta := 1
			if msg.String() == "left" {
				delta = -1
			}
			idx := 0
			for i, c := range f.choices {
				if c == f.value {
					idx = i
					break
				}
			}
			idx = (idx + delta + len(f.choices)) % len(f.choices)
			f.value = f.choices[idx]
			return a, nil
		}
	case "enter":
		if m.focus != 0 {
			return a, a.submitAdd()
		}
	}

	var cmd tea.Cmd
	if m.focus == 0 {
		m.urls, cmd = m.urls.Update(msg)
	} else if f := m.field(); f != nil && f.kind == kindText {
		m.input, cmd = m.input.Update(msg)
	}
	return a, cmd
}

// startProbe inspects the first URL in the textarea.
func (a *App) startProbe() tea.Cmd {
	m := &a.add
	for _, l := range strings.Split(m.urls.Value(), "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		url := strings.Fields(l)[0]
		if isLocalMeta(url) {
			a.pushToast("local files need no probing", toastInfo)
			return nil
		}
		m.probing = true
		m.probe = nil
		return probeCmd(url)
	}
	a.pushToast("paste a URL first, then check it", toastErr)
	return nil
}

// buildOptions turns the form fields into aria2 input options.
func (m *addModel) buildOptions() aria2.Options {
	m.commitInput()
	opts := aria2.Options{}
	for _, f := range m.fields {
		v := strings.TrimSpace(f.value)
		if v == "" {
			continue
		}
		opts[f.key] = v
	}
	return opts
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

	opts := m.buildOptions()
	out := opts[aria2.OptOut]
	delete(opts, aria2.OptOut) // only applied to single-URL adds below

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
				if err := addLocalMeta(ctx, client, mirrors[0], cloneOpts(opts)); err != nil {
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
	if panelW > 112 {
		panelW = 112
	}
	innerW := panelW - 2
	const labelW = 30

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
	btn := func(label string, hot bool) string {
		if hot {
			return styleToastGood.Render(" " + label + " ")
		}
		return styleBadge.Render(" " + label + " ")
	}
	labelBtnRight := func(lbl, b string) string {
		gap := innerW - lipgloss.Width(lbl) - lipgloss.Width(b)
		if gap < 1 {
			gap = 1
		}
		return lbl + strings.Repeat(" ", gap) + b
	}

	// --- title + torrent button
	torrentBtn := btn("⧉ torrent file", false)
	ln := add(labelBtnRight(styleTitle.Render("Add downloads"), torrentBtn))
	specs = append(specs, spec{ln, innerW - lipgloss.Width(torrentBtn), lipgloss.Width(torrentBtn), 1, "key:ctrl+t"})
	add("")

	// --- URLs textarea
	focusMark := " "
	if m.focus == 0 {
		focusMark = styleFieldFocus.Render("▍")
	}
	ln = add(focusMark + styleInputLabel.Render("URLs / magnets / local .torrent paths"))
	specs = append(specs, spec{ln, 0, innerW, 1, "addfld:0"})
	taLines := strings.Split(m.urls.View(), "\n")
	taTop := len(lines)
	for _, l := range taLines {
		add(l)
	}
	specs = append(specs, spec{taTop, 0, innerW, len(taLines), "addfld:0"})

	// --- probe status line
	if m.probing {
		add(styleCyan.Render(" ⌕ checking link…"))
	} else if m.probe != nil {
		add(" " + m.probe.render(innerW-2))
	} else {
		add("")
	}

	// --- fields (one line each)
	vis := m.visibleFields()
	fieldLine := func(slot int, f *addField) string {
		focused := m.focus == slot+1
		mark := " "
		if focused {
			mark = styleFieldFocus.Render("▍")
		}
		lbl := styleInputLabel.Render(padRight(f.label, labelW))
		if focused {
			lbl = styleFieldFocus.Render(padRight(f.label, labelW))
		}
		var val string
		switch f.kind {
		case kindText:
			if focused {
				val = m.input.View()
			} else if f.value != "" {
				val = styleText.Render(truncate(f.value, innerW-labelW-8))
			} else {
				val = styleFaint.Render(f.placeholder)
			}
		case kindSelect:
			shown := f.value
			if shown == "" {
				shown = "default"
			}
			if f.key == aria2.OptMaxDownloadLimit && f.value != "" {
				shown += "  = " + fmtLimit(f.value)
			}
			if focused {
				val = styleAccent.Render("‹ ") + styleTitle.Render(shown) + styleAccent.Render(" ›") +
					styleFaint.Render("  ←→")
			} else {
				val = styleAccent2.Render(shown)
			}
		case kindCheck:
			box := "☐"
			if f.value == "true" {
				box = styleGood.Render("☑")
			}
			val = box
			if focused {
				val += styleFaint.Render("  space toggles")
			}
		}
		return mark + lbl + val
	}

	extraBtnFor := func(f *addField) (string, string) {
		if f.key == "dir" {
			return btn("⧉ browse", false), "key:ctrl+o"
		}
		return "", ""
	}

	slot := 0
	advertisedAdv := false
	for _, fi := range vis {
		f := &a.add.fields[fi]
		if f.advanced && !advertisedAdv {
			advertisedAdv = true
			add("")
			add(" " + styleDim.Render("─ advanced ") + styleFaint.Render(strings.Repeat("─", maxInt(0, innerW-12))))
		}
		row := fieldLine(slot, f)
		if eb, ebID := extraBtnFor(f); eb != "" {
			row = labelBtnRight(row, eb)
			ln = add(row)
			specs = append(specs,
				spec{ln, innerW - lipgloss.Width(eb), lipgloss.Width(eb), 1, ebID},
				spec{ln, 0, innerW - lipgloss.Width(eb) - 1, 1, addFieldID(slot)})
		} else {
			ln = add(row)
			specs = append(specs, spec{ln, 0, innerW, 1, addFieldID(slot)})
		}
		// free-space hint under the dir field
		if f.key == "dir" {
			if free, total, err := diskUsage(strings.TrimSpace(f.value)); err == nil && total > 0 {
				add(" " + strings.Repeat(" ", labelW) + styleGood.Render("●") +
					styleDim.Render(" "+humanBytes(free)+" free on this volume"))
			}
		}
		slot++
	}

	// --- advanced toggle
	add("")
	advLabel := "▸ advanced options"
	if m.showAdvanced {
		advLabel = "▾ advanced options"
	}
	advBtn := styleBadge.Render(" " + advLabel + " ")
	ln = add(" " + advBtn)
	specs = append(specs, spec{ln, 1, lipgloss.Width(advBtn), 1, "key:ctrl+a"})

	if m.torrentPath != "" {
		add("")
		add(styleAccent2.Render(" ⧉ "+truncate(m.torrentPath, innerW-24)) + styleDim.Render("  (queued on submit)"))
	}
	add("")

	// --- action buttons
	startBtn := btn("▶ start download", true)
	checkBtn := btn("⌕ check link", false)
	cancelBtn := btn("esc cancel", false)
	ln = add(" " + startBtn + "  " + checkBtn + "  " + cancelBtn)
	x := 1
	specs = append(specs, spec{ln, x, lipgloss.Width(startBtn), 1, "key:ctrl+s"})
	x += lipgloss.Width(startBtn) + 2
	specs = append(specs, spec{ln, x, lipgloss.Width(checkBtn), 1, "key:ctrl+k"})
	x += lipgloss.Width(checkBtn) + 2
	specs = append(specs, spec{ln, x, lipgloss.Width(cancelBtn), 1, "key:esc"})

	content := strings.Join(lines, "\n")
	panel := stylePanelFocus.Width(panelW).Render(content)

	// Register hitboxes at absolute screen coordinates.
	x0 := (a.width - lipgloss.Width(panel)) / 2 // lipgloss.Place floors odd gaps
	if x0 < 0 {
		x0 = 0
	}
	const padX, padY = 2, 1 // panel border + horizontal padding
	for _, s := range specs {
		a.addHit(x0+padX+s.x, a.bodyTop()+padY+s.line, s.w, s.h, s.id)
	}
	return lipgloss.Place(a.width, h, lipgloss.Center, lipgloss.Top, panel)
}

// addFieldID builds the click id for a visible field slot (1-based focus).
func addFieldID(slot int) string {
	return "addfld:" + itoa(slot+1)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
