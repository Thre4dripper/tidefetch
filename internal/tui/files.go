package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fileBrowser is a read/write view of the download directory — browse what
// aria2 downloaded, open files, reveal them in the OS file manager, delete.
type fileBrowser struct {
	root       string
	cwd        string
	entries    []fileEntry
	cursor     int
	scroll     int
	showHidden bool
	loaded     bool
	mkdirMode  bool
	mkdirBuf   string
	notice     string
}

type fileEntry struct {
	name  string
	isDir bool
	size  int64
	mod   time.Time
}

func newFileBrowser(root string) fileBrowser {
	return fileBrowser{root: root, cwd: root}
}

// ensure loads the listing on first open and refreshes the root.
func (f *fileBrowser) ensure(root string) {
	f.root = root
	if !f.loaded {
		f.cwd = root
	}
	f.load()
}

func (f *fileBrowser) load() {
	f.loaded = true
	f.entries = f.entries[:0]
	resolved, moved := nearestExistingDir(f.cwd)
	if moved {
		f.cwd = resolved
		f.notice = "path unavailable · showing nearest accessible folder"
	} else {
		f.notice = ""
	}
	items, err := os.ReadDir(f.cwd)
	if err != nil {
		f.notice = err.Error()
		return
	}
	for _, e := range items {
		name := e.Name()
		if !f.showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		fe := fileEntry{name: name, isDir: e.IsDir()}
		if info, err := e.Info(); err == nil {
			fe.size = info.Size()
			fe.mod = info.ModTime()
		}
		f.entries = append(f.entries, fe)
	}
	sort.Slice(f.entries, func(i, j int) bool {
		if f.entries[i].isDir != f.entries[j].isDir {
			return f.entries[i].isDir
		}
		return strings.ToLower(f.entries[i].name) < strings.ToLower(f.entries[j].name)
	})
	if f.cursor >= len(f.entries) {
		f.cursor = maxInt(0, len(f.entries)-1)
	}
	f.scroll = minInt(f.scroll, f.cursor)
}

func (f *fileBrowser) selected() *fileEntry {
	if f.cursor >= 0 && f.cursor < len(f.entries) {
		return &f.entries[f.cursor]
	}
	return nil
}

// updateFiles handles keys in the file browser view.
func (a *App) updateFiles(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := &a.files
	if f.mkdirMode {
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(f.mkdirBuf)
			f.mkdirMode, f.mkdirBuf = false, ""
			if name != "" && !strings.ContainsAny(name, `/\`) {
				path := filepath.Join(f.cwd, name)
				if err := os.Mkdir(path, 0o755); err != nil {
					a.pushToast("mkdir failed: "+err.Error(), toastErr)
				} else {
					f.cwd = path
					f.load()
					a.pushToast("created "+name, toastOK)
				}
			}
		case "esc":
			f.mkdirMode, f.mkdirBuf = false, ""
		case "backspace":
			if len(f.mkdirBuf) > 0 {
				f.mkdirBuf = f.mkdirBuf[:len(f.mkdirBuf)-1]
			}
		default:
			if msg.Type == tea.KeyRunes {
				f.mkdirBuf += string(msg.Runes)
			}
		}
		return a, nil
	}
	switch msg.String() {
	case "esc", "q", "f":
		a.view = viewDownloads
	case "Q":
		a.confirmShutdown()
	case "j", "down":
		if f.cursor < len(f.entries)-1 {
			f.cursor++
		}
	case "k", "up":
		if f.cursor > 0 {
			f.cursor--
		}
	case "g", "home":
		f.cursor = 0
	case "G", "end":
		f.cursor = maxInt(0, len(f.entries)-1)
	case ".":
		f.showHidden = !f.showHidden
		f.load()
	case "d":
		f.cwd = f.root
		f.cursor, f.scroll = 0, 0
		f.load()
	case "~":
		f.cwd, _ = os.UserHomeDir()
		f.cursor, f.scroll = 0, 0
		f.load()
	case "/":
		f.cwd = string(filepath.Separator)
		f.cursor, f.scroll = 0, 0
		f.load()
	case "n":
		f.mkdirMode, f.mkdirBuf = true, ""
	case "backspace", "h", "left":
		parent := filepath.Dir(f.cwd)
		if parent != f.cwd {
			prev := filepath.Base(f.cwd)
			f.cwd = parent
			f.cursor, f.scroll = 0, 0
			f.load()
			for i, e := range f.entries {
				if e.name == prev {
					f.cursor = i
					break
				}
			}
		}
	case "enter", "l", "right":
		if e := f.selected(); e != nil {
			full := filepath.Join(f.cwd, e.name)
			if e.isDir {
				f.cwd = full
				f.cursor, f.scroll = 0, 0
				f.load()
			} else if err := openPath(full); err != nil {
				a.pushToast("open failed: "+err.Error(), toastErr)
			} else {
				a.pushToast("opening "+e.name, toastInfo)
			}
		}
	case "o":
		if e := f.selected(); e != nil {
			if err := revealPath(filepath.Join(f.cwd, e.name)); err != nil {
				a.pushToast("reveal failed: "+err.Error(), toastErr)
			}
		} else {
			_ = openPath(f.cwd)
		}
	case "r":
		f.load()
		a.pushToast("refreshed", toastInfo)
	case "x", "delete":
		if e := f.selected(); e != nil {
			full := filepath.Join(f.cwd, e.name)
			name := e.name
			isDir := e.isDir
			warn := "The file is erased from disk."
			if isDir {
				warn = "The folder and EVERYTHING inside it are erased."
			}
			a.confirmAction("Delete from disk?", name+"\n"+warn, func() tea.Cmd {
				var err error
				if isDir {
					err = os.RemoveAll(full)
				} else {
					err = os.Remove(full)
				}
				a.files.load()
				if err != nil {
					a.pushToast("delete failed: "+err.Error(), toastErr)
				} else {
					a.pushToast("deleted "+name, toastOK)
				}
				return nil
			})
		}
	}
	return a, nil
}

// viewFiles renders the browser inside a titled box.
func (a *App) viewFiles(h int) string {
	f := &a.files
	if !f.loaded {
		f.load()
	}
	w := a.width - 2
	inner := w - 4
	listH := h - 5 // box borders + header + status/input line
	if listH < 3 {
		listH = 3
	}

	if f.cursor < f.scroll {
		f.scroll = f.cursor
	}
	if f.cursor >= f.scroll+listH {
		f.scroll = f.cursor - listH + 1
	}

	var lines []string
	free := ""
	if fr, total, err := diskUsage(f.cwd); err == nil && total > 0 {
		free = humanBytes(fr) + " free"
	}
	lines = append(lines, styleDim.Render(fmt.Sprintf("%-*s %10s  %s",
		inner-30, "name", "size", "modified"))+"")
	if f.notice != "" {
		lines = append(lines, styleWarn.Render("  "+truncate(f.notice, inner-4)))
		listH--
		if listH < 1 {
			listH = 1
		}
	}

	if len(f.entries) == 0 {
		lines = append(lines, styleFaint.Render("  (empty folder)"))
	}
	end := minInt(f.scroll+listH, len(f.entries))
	rowYOffset := 2
	if f.notice != "" {
		rowYOffset++
	}
	for i := f.scroll; i < end; i++ {
		e := f.entries[i]
		// hitbox: +2 x (box border+pad), +1 y (box top border) + 1 (header line)
		a.addHit(2, a.bodyTop()+rowYOffset+(i-f.scroll), inner, 1, fmt.Sprintf("fbrow:%d", i))
		icon, style := "▸ ", styleText
		size := humanBytes(e.size)
		if e.isDir {
			icon, style = "▪ ", styleAccent2
			size = "—"
		}
		nameW := inner - 30
		row := fmt.Sprintf("%s %10s  %s",
			style.Render(padRight(icon+truncate(e.name, nameW-3), nameW)),
			styleDim.Render(size),
			styleFaint.Render(e.mod.Format("2006-01-02 15:04")))
		if i == f.cursor {
			row = styleRowSel.Width(inner).Render(styleSelBar.Render("┃") + row)
		} else {
			row = " " + row
		}
		lines = append(lines, row)
	}
	if f.mkdirMode {
		lines = append(lines, styleInputLabel.Render("  new folder: ")+styleText.Render(f.mkdirBuf)+styleAccent.Render("▌"))
	} else {
		lines = append(lines, styleFaint.Render("  ~ home  ·  / root  ·  d downloads  ·  n new folder"))
	}

	title := "Files · " + shortPath(f.cwd)
	if free != "" {
		title += " · " + free
	}
	box := titledBox(truncate(title, w-8), lines, w, styleBoxBorderHi, styleBoxTitle)
	return lipgloss.NewStyle().MaxHeight(h).Render(box)
}
