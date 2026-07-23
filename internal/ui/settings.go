package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/turbostart/aria2c-tui/pkg/aria2"
)

// settingsModel edits daemon-wide aria2 options live over RPC.
type settingsModel struct {
	loading bool
	opts    aria2.Options
	cursor  int
	editing bool
	buf     string   // text-mode buffer
	choices []string // active chooser values (nil = free text)
	choice  int      // active chooser index
}

// The options we surface. Options with choices edit as a selector,
// the rest as free text.
var settingDefs = []struct {
	key     string
	label   string
	hint    string
	choices []string
}{
	{aria2.OptMaxConcurrentDownloads, "Max concurrent downloads", "simultaneous downloads",
		[]string{"1", "2", "3", "5", "8", "10", "16"}},
	{aria2.OptMaxOverallDownloadLimit, "Global download limit", "0 = unlimited",
		[]string{"0", "256K", "512K", "1M", "2M", "5M", "10M", "20M", "50M"}},
	{aria2.OptMaxOverallUploadLimit, "Global upload limit", "0 = unlimited",
		[]string{"0", "128K", "256K", "512K", "1M", "2M", "5M"}},
	{aria2.OptSplit, "Split (connections per download)", "parallel segments",
		[]string{"1", "2", "4", "8", "16", "32", "64"}},
	{aria2.OptMaxConnectionPerServer, "Max connections per server", "per-host limit",
		[]string{"1", "2", "4", "8", "16"}},
	{aria2.OptMinSplitSize, "Min split size", "don't split below this",
		[]string{"1M", "5M", "10M", "20M", "64M"}},
	{aria2.OptSeedRatio, "BT seed ratio", "0 = seed forever",
		[]string{"0.0", "0.5", "1.0", "2.0", "5.0"}},
	{aria2.OptSeedTime, "BT seed time (minutes)", "0 = no time seeding",
		[]string{"0", "10", "30", "60", "120"}},
	{aria2.OptBTMaxPeers, "BT max peers", "0 = unlimited",
		[]string{"0", "20", "55", "100", "200"}},
	{aria2.OptUserAgent, "User agent", "free text", nil},
	{aria2.OptAllProxy, "Proxy", "http://user:pass@host:port", nil},
	{"dir", "Default download dir (daemon)", "absolute path", nil},
}

func newSettingsModel() settingsModel { return settingsModel{loading: true} }

func (m *settingsModel) absorb(msg globalOptsMsg) {
	m.loading = false
	if msg.err == nil {
		m.opts = msg.opts
	}
}

// updateSettings handles keys on the settings screen.
func (a *App) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m := &a.settings

	if m.editing {
		// --- selector mode -------------------------------------------------
		if m.choices != nil {
			switch msg.String() {
			case "left", "h", "-":
				m.choice = (m.choice - 1 + len(m.choices)) % len(m.choices)
			case "right", "l", "+", "tab", " ":
				m.choice = (m.choice + 1) % len(m.choices)
			case "enter":
				m.editing = false
				return a, a.saveSetting(m.choices[m.choice])
			case "esc":
				m.editing = false
			}
			return a, nil
		}
		// --- free-text mode --------------------------------------------------
		switch msg.String() {
		case "enter":
			m.editing = false
			return a, a.saveSetting(strings.TrimSpace(m.buf))
		case "esc":
			m.editing = false
		case "backspace":
			if len(m.buf) > 0 {
				m.buf = m.buf[:len(m.buf)-1]
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.buf += string(msg.Runes)
			}
		}
		return a, nil
	}

	switch msg.String() {
	case "esc", "q", "s":
		a.view = viewDownloads
	case "Q":
		a.confirmShutdown()
	case "j", "down":
		if m.cursor < len(settingDefs)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter", "e", "l", "right":
		def := settingDefs[m.cursor]
		m.editing = true
		m.buf = m.opts[def.key]
		m.choices = nil
		if def.choices != nil {
			// Build the chooser: current value first if it's not a preset.
			m.choices = def.choices
			m.choice = 0
			cur := m.opts[def.key]
			found := false
			for i, c := range m.choices {
				if normEq(c, cur) {
					m.choice, found = i, true
					break
				}
			}
			if !found && cur != "" {
				m.choices = append([]string{cur}, def.choices...)
			}
		}
	case "r":
		client := a.client
		m.loading = true
		return a, func() tea.Msg {
			ctx, cancel := rpcCtx()
			defer cancel()
			opts, err := client.GetGlobalOption(ctx)
			return globalOptsMsg{opts: opts, err: err}
		}
	}
	return a, nil
}

// saveSetting persists the value of the setting under the cursor.
func (a *App) saveSetting(val string) tea.Cmd {
	def := settingDefs[a.settings.cursor]
	client := a.client
	return tea.Sequence(
		doRPC(def.label+" → "+val, func(ctx context.Context) error {
			return client.ChangeGlobalOption(ctx, aria2.Options{def.key: val})
		}),
		func() tea.Msg {
			ctx, cancel := rpcCtx()
			defer cancel()
			opts, err := client.GetGlobalOption(ctx)
			return globalOptsMsg{opts: opts, err: err}
		},
	)
}

// normEq compares option values loosely (512K == 512000, 1.0 == 1).
func normEq(a, b string) bool {
	if strings.EqualFold(a, b) {
		return true
	}
	if pa, pb := parseLimit(a), parseLimit(b); pa > 0 && pa == pb {
		return true
	}
	return strings.TrimRight(a, ".0") == strings.TrimRight(b, ".0") && a != "" && b != ""
}

// viewSettings renders the option list.
func (a *App) viewSettings(h int) string {
	m := &a.settings
	if m.loading {
		return lipgloss.Place(a.width, h, lipgloss.Center, lipgloss.Center, styleDim.Render("loading options…"))
	}

	var b strings.Builder
	fmt.Fprint(&b, styleTitle.Render("Settings"), styleDim.Render("  (live daemon options — saved to session)"), "\n\n")

	for i, def := range settingDefs {
		a.addHit(0, bodyTop+2+i, a.width, 1, fmt.Sprintf("srow:%d", i))
		val := m.opts[def.key]
		disp := val
		switch def.key {
		case aria2.OptMaxOverallDownloadLimit, aria2.OptMaxOverallUploadLimit:
			disp = fmtLimit(val)
		}
		if disp == "" {
			disp = styleFaint.Render("(unset)")
		}
		label := padRight(def.label, 34)
		if i == m.cursor {
			if m.editing && m.choices != nil {
				// selector: ‹ value ›
				sel := styleAccent.Render("‹ ") + styleTitle.Render(m.choices[m.choice]) + styleAccent.Render(" ›")
				extra := ""
				switch def.key {
				case aria2.OptMaxOverallDownloadLimit, aria2.OptMaxOverallUploadLimit:
					extra = styleDim.Render("  = " + fmtLimit(m.choices[m.choice]))
				}
				b.WriteString(styleRowSel.Width(a.width - 2).Render(
					styleSelBar.Render("┃") + styleFieldFocus.Render(label) + sel + extra +
						styleDim.Render("   ←→ choose · enter save · esc cancel")))
			} else if m.editing {
				b.WriteString(styleRowSel.Width(a.width - 2).Render(
					styleSelBar.Render("┃") + styleFieldFocus.Render(label) +
						styleText.Render(m.buf) + styleAccent.Render("▌") +
						styleDim.Render("  ("+def.hint+")")))
			} else {
				mode := " ‹edit›"
				if def.choices != nil {
					mode = " ‹select›"
				}
				b.WriteString(styleRowSel.Width(a.width - 2).Render(
					styleSelBar.Render("┃") + styleFieldFocus.Render(label) + styleAccent2.Render(disp) +
						styleFaint.Render(mode+" · "+def.hint)))
			}
		} else {
			fmt.Fprint(&b, " ", styleInputLabel.Render(label), styleText.Render(disp))
		}
		b.WriteString("\n")
	}

	fmt.Fprint(&b, "\n", styleDim.Render(" RPC endpoint: "), styleText.Render(a.rpcURL))
	if a.spawned {
		b.WriteString(styleDim.Render("  (daemon spawned by aria2tui)"))
	}
	return b.String()
}

// viewHelp renders the key reference.
func (a *App) viewHelp(h int) string {
	col := func(pairs [][2]string) string {
		var b strings.Builder
		for _, p := range pairs {
			fmt.Fprint(&b, styleKey.Render(padLeft(p[0], 9)), "  ", styleDesc.Render(p[1]), "\n")
		}
		return b.String()
	}

	left := styleTitle.Render("Downloads") + "\n" + col([][2]string{
		{"j/k ↑↓", "move"},
		{"1-4", "filter tabs"},
		{"space/p", "pause · resume"},
		{"P / R", "pause all · resume all"},
		{"enter/i", "details"},
		{"a", "add downloads"},
		{"x", "remove"},
		{"D", "remove + delete files"},
		{"r", "retry failed"},
		{"J/K", "move in queue"},
		{"S", "cycle sort"},
		{"/", "search"},
		{"c", "clear finished"},
		{"w", "save session"},
	}) + "\n" + styleTitle.Render("Speed limits") + "\n" + col([][2]string{
		{"[ / ]", "global ↓ limit −/+"},
		{"{ / }", "global ↑ limit −/+"},
		{"+ / -", "per-download (details)"},
	})

	right := styleTitle.Render("Everywhere") + "\n" + col([][2]string{
		{"o", "open folder"},
		{"y", "copy URL / magnet"},
		{"t", "toggle side panel"},
		{"f", "file browser"},
		{"h", "history"},
		{"s", "settings"},
		{"?", "help"},
		{"esc", "back / clear"},
		{"q", "quit (daemon keeps running)"},
		{"Q", "quit + stop daemon"},
	}) + "\n" + styleTitle.Render("Mouse") + "\n" + col([][2]string{
		{"click", "tabs · rows · buttons · fields"},
		{"click ×2", "row → details · setting → edit"},
		{"wheel", "scroll lists"},
	}) + "\n" + styleTitle.Render("Files view") + "\n" + col([][2]string{
		{"enter", "open dir / launch file"},
		{"o / x", "reveal · delete"},
		{"d", "jump to download dir"},
	}) + "\n" + styleTitle.Render("Add form") + "\n" + col([][2]string{
		{"tab", "next field"},
		{"ctrl+o", "browse directory"},
		{"ctrl+t", "pick .torrent/.metalink"},
		{"ctrl+s", "start"},
	})

	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(44).Render(left),
		right)
	panel := stylePanel.Render(cols)
	return lipgloss.Place(a.width, h, lipgloss.Center, lipgloss.Top, panel)
}
