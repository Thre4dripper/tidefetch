package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/turbostart/aria2c-tui/internal/history"
)

// historyModel lists past downloads with category filter + search.
type historyModel struct {
	all       []history.Entry
	rows      []history.Entry
	cursor    int
	scroll    int
	catIdx    int // 0 = all, 1.. = history.Categories
	filter    string
	searching bool
}

func newHistoryModel() historyModel { return historyModel{} }

func (m *historyModel) reload(store *history.Store) {
	m.all = store.All()
	m.apply()
}

func (m *historyModel) apply() {
	m.rows = m.rows[:0]
	var cat string
	if m.catIdx > 0 && m.catIdx <= len(history.Categories) {
		cat = history.Categories[m.catIdx-1]
	}
	q := strings.ToLower(m.filter)
	for _, e := range m.all {
		if cat != "" && e.Category != cat {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(e.Name), q) &&
			!strings.Contains(strings.ToLower(e.URL), q) {
			continue
		}
		m.rows = append(m.rows, e)
	}
	if m.cursor >= len(m.rows) {
		m.cursor = maxInt(0, len(m.rows)-1)
	}
}

// updateHistory handles keys on the history screen.
func (a *App) updateHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m := &a.historyV

	if m.searching {
		switch msg.String() {
		case "enter", "esc":
			m.searching = false
			if msg.String() == "esc" {
				m.filter = ""
			}
			m.apply()
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.apply()
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.filter += string(msg.Runes)
				m.apply()
			}
		}
		return a, nil
	}

	switch msg.String() {
	case "esc", "q", "h":
		a.view = viewDownloads
	case "Q":
		a.confirmShutdown()
	case "j", "down":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = maxInt(0, len(m.rows)-1)
	case "/":
		m.searching = true
	case "c":
		m.catIdx = (m.catIdx + 1) % (len(history.Categories) + 1)
		m.apply()
	case "x", "d":
		if m.cursor < len(m.rows) {
			gid := m.rows[m.cursor].GID
			a.hist.Delete(gid)
			a.histDirty = true
			m.reload(a.hist)
		}
	case "C":
		a.confirmAction("Clear ALL history?", "Every record will be erased.", func() tea.Cmd {
			a.hist.Clear()
			a.histDirty = true
			a.historyV.reload(a.hist)
			return nil
		})
	case "o":
		if m.cursor < len(m.rows) {
			if err := openPath(m.rows[m.cursor].Dir); err != nil {
				a.pushToast("open failed: "+err.Error(), toastErr)
			}
		}
	case "y":
		if m.cursor < len(m.rows) && m.rows[m.cursor].URL != "" {
			_ = copyClipboard(m.rows[m.cursor].URL)
			a.pushToast("URL copied", toastOK)
		}
	case "enter", "r":
		if m.cursor < len(m.rows) {
			e := m.rows[m.cursor]
			if e.URL == "" {
				a.pushToast("no URL recorded for this entry", toastErr)
				return a, nil
			}
			a.add.reset(a.cfg)
			a.add.urls.SetValue(e.URL)
			if e.Dir != "" {
				a.add.dir.SetValue(e.Dir)
			}
			a.view = viewAdd
			return a, a.add.focusCmd()
		}
	}
	return a, nil
}

// viewHistory renders the table.
func (a *App) viewHistory(h int) string {
	m := &a.historyV

	cat := "All"
	if m.catIdx > 0 {
		cat = history.Categories[m.catIdx-1]
	}
	head := styleTitle.Render("History") + "  " +
		styleBadge.Render(cat) + " " +
		styleDim.Render(fmt.Sprintf("%d item(s)", len(m.rows)))
	if m.searching || m.filter != "" {
		head += "  " + styleAccent.Render("/"+m.filter)
		if m.searching {
			head += styleAccent.Render("▌")
		}
	}

	if len(m.rows) == 0 {
		body := lipgloss.Place(a.width, h-2, lipgloss.Center, lipgloss.Center,
			styleDim.Render("history is empty"))
		return head + "\n" + body
	}

	listH := h - 3
	if listH < 1 {
		listH = 1
	}
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+listH {
		m.scroll = m.cursor - listH + 1
	}

	w := a.width
	var b strings.Builder
	fmt.Fprintln(&b, head)
	fmt.Fprintln(&b, styleDim.Render(fmt.Sprintf("  %-10s %-9s %9s  %-12s %s",
		"date", "status", "size", "category", "name")))

	end := minInt(m.scroll+listH, len(m.rows))
	for i := m.scroll; i < end; i++ {
		e := m.rows[i]
		a.addHit(0, bodyTop+2+(i-m.scroll), w, 1, fmt.Sprintf("hrow:%d", i))
		stStyle := styleGood
		stTxt := "done"
		if e.Status == "error" {
			stStyle, stTxt = styleBad, "failed"
		}
		nameW := w - 50
		if nameW < 10 {
			nameW = 10
		}
		line := fmt.Sprintf("  %-10s %s %9s  %-12s %s",
			e.Finished.Format("2006-01-02"),
			stStyle.Render(padRight(stTxt, 9)),
			humanBytes(e.Size),
			styleAccent2.Render(padRight(e.Category, 12)),
			styleText.Render(truncate(e.Name, nameW)))
		if i == m.cursor {
			line = styleRowSel.Width(w).Render(styleSelBar.Render("┃") + line[1:])
		}
		fmt.Fprintln(&b, line)
	}
	return strings.TrimRight(b.String(), "\n")
}
