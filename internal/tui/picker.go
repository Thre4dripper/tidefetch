package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// dirPicker is a themed filesystem browser used for "dir lookup" —
// choosing a download directory or picking a .torrent/.metalink file.
type dirPicker struct {
	title      string
	cwd        string
	entries    []os.DirEntry
	cursor     int
	scroll     int
	showHidden bool
	fileMode   bool     // true: pick a file; false: pick a directory
	exts       []string // when fileMode, only show these extensions
	mkdirBuf   string
	mkdirMode  bool
	notice     string
	width      int
	height     int
	onPick     func(path string) tea.Cmd
}

func newDirPicker(title, start string, fileMode bool, exts []string, onPick func(string) tea.Cmd) *dirPicker {
	var moved bool
	start, moved = nearestExistingDir(start)
	p := &dirPicker{title: title, cwd: start, fileMode: fileMode, exts: exts, onPick: onPick}
	if moved {
		p.notice = "path unavailable · showing nearest accessible folder"
	}
	p.load()
	return p
}

// nearestExistingDir walks toward the filesystem root until it finds a
// readable directory, then falls back to the user's home directory.
func nearestExistingDir(start string) (string, bool) {
	home, _ := os.UserHomeDir()
	if start == "" {
		return home, true
	}
	original := filepath.Clean(start)
	p := original
	for {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			if _, err := os.ReadDir(p); err == nil {
				return p, p != original
			}
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}
	return home, home != original
}

func (p *dirPicker) setSize(w, h int) { p.width, p.height = w, h }

func (p *dirPicker) load() {
	p.entries = nil
	p.cursor, p.scroll = 0, 0
	items, err := os.ReadDir(p.cwd)
	if err != nil {
		resolved, _ := nearestExistingDir(filepath.Dir(p.cwd))
		p.cwd = resolved
		p.notice = "folder became unavailable · showing nearest accessible folder"
		items, err = os.ReadDir(p.cwd)
		if err != nil {
			p.notice = err.Error()
			return
		}
	}
	for _, e := range items {
		name := e.Name()
		if !p.showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			p.entries = append(p.entries, e)
			continue
		}
		if p.fileMode {
			if len(p.exts) == 0 {
				p.entries = append(p.entries, e)
				continue
			}
			ext := strings.ToLower(filepath.Ext(name))
			for _, want := range p.exts {
				if ext == want {
					p.entries = append(p.entries, e)
					break
				}
			}
		}
	}
	sort.Slice(p.entries, func(i, j int) bool {
		di, dj := p.entries[i].IsDir(), p.entries[j].IsDir()
		if di != dj {
			return di
		}
		return strings.ToLower(p.entries[i].Name()) < strings.ToLower(p.entries[j].Name())
	})
}

// updatePicker handles keys while the picker overlay is open.
func (a *App) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := a.picker

	if p.mkdirMode {
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(p.mkdirBuf)
			p.mkdirMode = false
			p.mkdirBuf = ""
			if name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`) {
				path := filepath.Join(p.cwd, name)
				if err := os.Mkdir(path, 0o755); err != nil {
					a.pushToast("mkdir failed: "+err.Error(), toastErr)
				} else {
					p.cwd = path
					p.notice = ""
					p.load()
				}
			} else if name != "" {
				a.pushToast("invalid folder name", toastErr)
			}
		case "esc":
			p.mkdirMode = false
			p.mkdirBuf = ""
		case "backspace":
			if len(p.mkdirBuf) > 0 {
				p.mkdirBuf = p.mkdirBuf[:len(p.mkdirBuf)-1]
			}
		default:
			if msg.Type == tea.KeyRunes {
				p.mkdirBuf += string(msg.Runes)
			}
		}
		return a, nil
	}

	switch msg.String() {
	case "esc", "ctrl+c":
		a.picker = nil
	case "j", "down":
		if p.cursor < len(p.entries)-1 {
			p.cursor++
		}
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
	case "g", "home":
		p.cursor = 0
	case "G", "end":
		p.cursor = maxInt(0, len(p.entries)-1)
	case ".":
		p.showHidden = !p.showHidden
		p.load()
	case "n":
		p.mkdirMode = true
		p.mkdirBuf = ""
	case "~":
		home, _ := os.UserHomeDir()
		p.cwd = home
		p.notice = ""
		p.load()
	case "/":
		p.cwd = string(filepath.Separator)
		p.notice = ""
		p.load()
	case "d":
		var moved bool
		p.cwd, moved = nearestExistingDir(a.cfg.DownloadDir)
		p.notice = ""
		if moved {
			p.notice = "download folder unavailable · showing nearest accessible folder"
		}
		p.load()
	case "backspace", "h", "left":
		parent := filepath.Dir(p.cwd)
		if parent != p.cwd {
			prev := filepath.Base(p.cwd)
			p.cwd = parent
			p.notice = ""
			p.load()
			for i, e := range p.entries {
				if e.Name() == prev {
					p.cursor = i
					break
				}
			}
		}
	case "enter", "l", "right":
		if p.cursor < len(p.entries) {
			e := p.entries[p.cursor]
			full := filepath.Join(p.cwd, e.Name())
			if e.IsDir() {
				p.cwd = full
				p.notice = ""
				p.load()
			} else if p.fileMode {
				cmd := p.onPick(full)
				a.picker = nil
				return a, cmd
			}
		}
	case "s", "ctrl+s", " ":
		if !p.fileMode {
			cmd := p.onPick(p.cwd)
			a.picker = nil
			return a, cmd
		}
	}
	return a, nil
}

// pickerRowClick selects a row; a second click acts like enter.
func (a *App) pickerRowClick(n int) (tea.Model, tea.Cmd) {
	p := a.picker
	if n < 0 || n >= len(p.entries) {
		return a, nil
	}
	if p.cursor == n {
		return a.updatePicker(tea.KeyMsg{Type: tea.KeyEnter})
	}
	p.cursor = n
	return a, nil
}

// pickerButtonClick handles visible modal buttons directly. Keeping these
// semantic avoids terminal/key translation differences for mouse clicks.
func (a *App) pickerButtonClick(action string) (tea.Model, tea.Cmd) {
	p := a.picker
	if p == nil {
		return a, nil
	}
	switch action {
	case "home":
		p.cwd, _ = os.UserHomeDir()
		p.notice = ""
		p.load()
	case "root":
		p.cwd = string(filepath.Separator)
		p.notice = ""
		p.load()
	case "downloads":
		var moved bool
		p.cwd, moved = nearestExistingDir(a.cfg.DownloadDir)
		p.notice = ""
		if moved {
			p.notice = "download folder unavailable · showing nearest accessible folder"
		}
		p.load()
	case "new":
		if !p.fileMode {
			p.mkdirMode = true
			p.mkdirBuf = ""
		}
	case "use":
		if !p.fileMode {
			cmd := p.onPick(p.cwd)
			a.picker = nil
			return a, cmd
		}
	}
	return a, nil
}

// render draws the picker modal and returns clickable regions.
func (p *dirPicker) render() (string, []hitspec) {
	w := p.width * 2 / 3
	if w < 44 {
		w = minInt(p.width-4, 44)
	}
	if w > 90 {
		w = 90
	}
	listH := p.height - 14
	if listH < 5 {
		listH = 5
	}
	if listH > 22 {
		listH = 22
	}

	inner := w - 8
	// content offset inside styleModal: border(1)+padding(3,1)
	const offX, offY = 4, 2
	var specs []hitspec
	line := 0

	var b strings.Builder
	writeLn := func(s string) {
		b.WriteString(s)
		b.WriteString("\n")
		line++
	}

	writeLn(styleTitle.Render(p.title))
	homeBtn := styleBadge.Render(" ⌂ home ")
	rootBtn := styleBadge.Render(" / root ")
	downBtn := styleBadge.Render(" ⇣ downloads ")
	chipX := offX
	for _, chip := range []struct {
		text string
		key  string
	}{{homeBtn, "home"}, {rootBtn, "root"}, {downBtn, "downloads"}} {
		specs = append(specs, hitspec{x: chipX, y: offY + line, w: lipgloss.Width(chip.text), h: 1, id: "pbtn:" + chip.key})
		chipX += lipgloss.Width(chip.text) + 1
	}
	writeLn(homeBtn + " " + rootBtn + " " + downBtn)
	freeInfo := ""
	if free, total, err := diskUsage(p.cwd); err == nil && total > 0 {
		freeInfo = styleDim.Render("  · " + humanBytes(free) + " free")
	}
	writeLn(styleAccent2.Render(truncate(p.cwd, inner-lipgloss.Width(freeInfo))) + freeInfo)
	if p.notice != "" {
		writeLn(styleWarn.Render(truncate(p.notice, inner)))
	}
	writeLn(styleFaint.Render(strings.Repeat("─", inner)))

	if p.cursor < p.scroll {
		p.scroll = p.cursor
	}
	if p.cursor >= p.scroll+listH {
		p.scroll = p.cursor - listH + 1
	}

	if len(p.entries) == 0 {
		writeLn(styleDim.Render("  (empty)"))
	}
	end := minInt(p.scroll+listH, len(p.entries))
	for i := p.scroll; i < end; i++ {
		e := p.entries[i]
		icon := "▸ "
		style := styleText
		if e.IsDir() {
			icon = "▪ "
			style = styleAccent2
		}
		rowTxt := icon + e.Name()
		if e.IsDir() {
			rowTxt += "/"
		}
		rowTxt = truncate(rowTxt, inner-2)
		specs = append(specs, hitspec{x: offX, y: offY + line, w: inner, h: 1, id: fmt.Sprintf("prow:%d", i)})
		if i == p.cursor {
			writeLn(styleRowSel.Width(inner).Render(styleSelBar.Render("┃") + style.Bold(true).Render(rowTxt)))
		} else {
			writeLn(" " + style.Render(rowTxt))
		}
	}

	writeLn(styleFaint.Render(strings.Repeat("─", inner)))
	if p.mkdirMode {
		fmt.Fprint(&b, styleInputLabel.Render("new folder: "), styleText.Render(p.mkdirBuf), styleAccent.Render("▌"))
	} else if p.fileMode {
		fmt.Fprint(&b, styleKey.Render("↵"), styleDesc.Render(" pick file  "),
			styleKey.Render("bksp"), styleDesc.Render(" up  "),
			styleKey.Render("."), styleDesc.Render(" hidden  "),
			styleKey.Render("esc"), styleDesc.Render(" cancel"))
	} else {
		useBtn := styleToastGood.Render(" ✓ use this folder ")
		newBtn := styleBadge.Render(" + new folder ")
		specs = append(specs, hitspec{x: offX, y: offY + line, w: lipgloss.Width(useBtn), h: 1, id: "pbtn:use"})
		specs = append(specs, hitspec{x: offX + lipgloss.Width(useBtn) + 2, y: offY + line, w: lipgloss.Width(newBtn), h: 1, id: "pbtn:new"})
		writeLn(useBtn + "  " + newBtn)
		fmt.Fprint(&b, styleKey.Render("↵"), styleDesc.Render(" enter dir  "),
			styleKey.Render("bksp"), styleDesc.Render(" up  "),
			styleKey.Render("."), styleDesc.Render(" hidden  "),
			styleKey.Render("esc"), styleDesc.Render(" cancel"))
	}

	return styleModal.Width(w).Render(b.String()), specs
}

// --- confirm modal -----------------------------------------------------------

type confirmModel struct {
	title  string
	body   string
	yes    bool
	action func() tea.Cmd
}

func (a *App) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	c := a.confirm
	switch msg.String() {
	case "esc", "n", "N":
		a.confirm = nil
	case "left", "right", "tab", "h", "l":
		c.yes = !c.yes
	case "y", "Y":
		a.confirm = nil
		return a, c.action()
	case "enter":
		a.confirm = nil
		if c.yes {
			return a, c.action()
		}
	}
	return a, nil
}

func (c *confirmModel) render() (string, []hitspec) {
	yes := "    Yes    "
	no := "    No    "
	if c.yes {
		yes = styleToastBad.Render(yes)
		no = styleBadge.Render(no)
	} else {
		yes = styleBadge.Render(yes)
		no = styleToastInfo.Render(no)
	}
	bodyLines := strings.Count(c.body, "\n") + 1
	btnLine := 2 + bodyLines + 1 // title, blank, body…, blank
	const offX, offY = 4, 2      // styleModal border+padding
	noW := lipgloss.Width(no)
	specs := []hitspec{
		{x: offX, y: offY + btnLine, w: noW, h: 1, id: "confirm:no"},
		{x: offX + noW + 3, y: offY + btnLine, w: lipgloss.Width(yes), h: 1, id: "confirm:yes"},
	}
	body := styleTitle.Render(c.title) + "\n\n" +
		styleText.Render(c.body) + "\n\n" +
		lipgloss.JoinHorizontal(lipgloss.Center, no, "   ", yes) + "\n\n" +
		styleDim.Render("click · y/n · ←→ · enter")
	return styleModal.Render(body), specs
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
