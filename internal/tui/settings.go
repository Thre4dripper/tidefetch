package tui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/crypto/bcrypt"

	"github.com/Thre4dripper/tidefetch/pkg/aria2"
)

const (
	localDownloadDir = "@download-dir"
	localPollMS      = "@poll-ms"
	localSidebar     = "@sidebar"
	localCompact     = "@compact-rows"
	localConfirm     = "@confirm-remove"
	localHistory     = "@history-limit"
	localWebHost     = "@web-host"
	localWebPort     = "@web-port"
	localWebAuth     = "@web-auth"
	localWebPassword = "@web-password"
)

type settingDef struct {
	key     string
	label   string
	hint    string
	choices []string
	local   bool
}

func settingReadOnly(def settingDef) bool { return def.key == localWebAuth }
func settingSecret(def settingDef) bool   { return def.key == localWebPassword }

type settingTab struct {
	name  string
	short string
	defs  []settingDef
	raw   bool
}

var settingTabs = []settingTab{
	{name: "Transfer", short: "Xfer", defs: []settingDef{
		{aria2.OptMaxOverallDownloadLimit, "Global download limit", "0 = unlimited", []string{"0", "1M", "2M", "5M", "10M", "20M", "50M"}, false},
		{aria2.OptMaxOverallUploadLimit, "Global upload limit", "0 = unlimited", []string{"0", "256K", "512K", "1M", "2M", "5M"}, false},
		{aria2.OptMaxConcurrentDownloads, "Concurrent downloads", "simultaneous transfers", []string{"1", "2", "3", "5", "8", "10", "16"}, false},
		{aria2.OptSplit, "Connections per download", "parallel segments", []string{"1", "2", "4", "8", "16", "32", "64"}, false},
		{aria2.OptMaxConnectionPerServer, "Connections per server", "per-host limit", []string{"1", "2", "4", "8", "16"}, false},
		{aria2.OptMinSplitSize, "Minimum split size", "smaller = more segments", []string{"1M", "5M", "10M", "20M", "1G"}, false},
	}},
	{name: "Behaviour", short: "Behave", defs: []settingDef{
		{aria2.OptContinue, "Resume partial downloads", "continue incomplete files", []string{"true", "false"}, false},
		{"file-allocation", "File allocation", "disk allocation strategy", []string{"none", "prealloc", "trunc", "falloc"}, false},
		{"max-tries", "Maximum retries", "0 = retry forever", []string{"0", "3", "5", "10", "20"}, false},
		{"retry-wait", "Retry wait (seconds)", "delay between attempts", []string{"0", "5", "10", "30", "60"}, false},
		{"auto-file-renaming", "Auto-rename conflicts", "avoid overwriting existing files", []string{"true", "false"}, false},
		{"allow-overwrite", "Allow overwrite", "replace existing files", []string{"true", "false"}, false},
		{"check-integrity", "Verify integrity", "check hashes when available", []string{"true", "false"}, false},
	}},
	{name: "BitTorrent", short: "BT", defs: []settingDef{
		{aria2.OptSeedRatio, "Seed ratio", "0.0 = seed forever", []string{"0.0", "0.5", "1.0", "2.0", "5.0"}, false},
		{aria2.OptSeedTime, "Seed time (minutes)", "0 = no time limit", []string{"0", "10", "30", "60", "120"}, false},
		{aria2.OptBTMaxPeers, "Maximum peers", "0 = unlimited", []string{"0", "20", "55", "100", "200"}, false},
		{"bt-request-peer-speed-limit", "Peer speed target", "preferred peer throughput", []string{"50K", "256K", "1M", "5M"}, false},
		{"listen-port", "Listen port", "single port or range", nil, false},
		{"dht-listen-port", "DHT listen port", "single port or range", nil, false},
	}},
	{name: "Network", short: "Net", defs: []settingDef{
		{aria2.OptAllProxy, "Proxy", "http://user:pass@host:port", nil, false},
		{aria2.OptUserAgent, "User agent", "HTTP user-agent string", nil, false},
		{"connect-timeout", "Connect timeout (seconds)", "connection establishment", []string{"10", "30", "60", "120"}, false},
		{"timeout", "I/O timeout (seconds)", "socket inactivity timeout", []string{"10", "30", "60", "120"}, false},
	}},
	{name: "Interface", short: "TUI", defs: []settingDef{
		{localDownloadDir, "Default download directory", "TUI + daemon default", nil, true},
		{localPollMS, "Refresh interval", "milliseconds; 300 is the minimum", []string{"300", "500", "700", "1000", "2000"}, true},
		{localSidebar, "Side panel by default", "toggle any time with t", []string{"true", "false"}, true},
		{localCompact, "Compact download cards", "hide the context line", []string{"true", "false"}, true},
		{localConfirm, "Confirm before removing", "delete-files always confirms", []string{"true", "false"}, true},
		{localHistory, "History limit", "applies fully after restart", []string{"500", "1000", "2000", "5000", "10000"}, true},
	}},
	{name: "Advanced", short: "All", raw: true},
	{name: "Security", short: "Sec", defs: []settingDef{
		{key: localWebHost, label: "Web listen address", hint: "applies after web server restart", local: true},
		{key: localWebPort, label: "Web listen port", hint: "1–65535; applies after restart", choices: []string{"8210", "8080", "8000", "9090"}, local: true},
		{key: localWebAuth, label: "Web UI authentication", hint: "managed by tidefetch serve", local: true},
		{key: localWebPassword, label: "Set/change web password", hint: "min 6 · restart web server to apply", local: true},
	}},
}

// settingsModel edits daemon-wide aria2 options and TUI-local preferences.
type settingsModel struct {
	loading      bool
	opts         aria2.Options
	tab          int
	cursor       int
	scroll       int
	editing      bool
	buf          string
	choices      []string
	choice       int
	filtering    bool
	filter       string
	revealSecret bool
}

func newSettingsModel() settingsModel { return settingsModel{loading: true} }

func (m *settingsModel) absorb(msg globalOptsMsg) {
	m.loading = false
	if msg.err == nil {
		m.opts = msg.opts
	}
}

func (m *settingsModel) defs() []settingDef {
	if m.tab < 0 || m.tab >= len(settingTabs) {
		return nil
	}
	tab := settingTabs[m.tab]
	if !tab.raw {
		return tab.defs
	}
	q := strings.ToLower(strings.TrimSpace(m.filter))
	keys := make([]string, 0, len(m.opts))
	for key := range m.opts {
		if q == "" || strings.Contains(strings.ToLower(key), q) ||
			strings.Contains(strings.ToLower(m.opts[key]), q) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	defs := make([]settingDef, 0, len(keys))
	for _, key := range keys {
		def := settingDef{key: key, label: key, hint: "raw aria2 option"}
		if val := m.opts[key]; val == "true" || val == "false" {
			def.choices = []string{"true", "false"}
		}
		defs = append(defs, def)
	}
	return defs
}

func (m *settingsModel) clamp() {
	defs := m.defs()
	if len(defs) == 0 {
		m.cursor, m.scroll = 0, 0
		return
	}
	if m.cursor >= len(defs) {
		m.cursor = len(defs) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *settingsModel) changeTab(delta int) {
	m.tab = (m.tab + delta + len(settingTabs)) % len(settingTabs)
	m.cursor, m.scroll = 0, 0
	m.editing, m.filtering = false, false
	m.revealSecret = false
	if !settingTabs[m.tab].raw {
		m.filter = ""
	}
}

func (a *App) settingValue(def settingDef) string {
	if !def.local {
		return a.settings.opts[def.key]
	}
	switch def.key {
	case localDownloadDir:
		return a.cfg.DownloadDir
	case localPollMS:
		return strconv.Itoa(a.cfg.PollMS)
	case localSidebar:
		return strconv.FormatBool(a.cfg.Sidebar)
	case localCompact:
		return strconv.FormatBool(a.cfg.CompactRows)
	case localConfirm:
		return strconv.FormatBool(a.cfg.ConfirmRemove)
	case localHistory:
		return strconv.Itoa(a.cfg.HistoryLimit)
	case localWebHost:
		return a.cfg.WebHost
	case localWebPort:
		return strconv.Itoa(a.cfg.WebPort)
	case localWebAuth:
		if a.cfg.WebPasswordHash != "" {
			return "password protected"
		}
		return "not configured"
	case localWebPassword:
		if a.cfg.WebPasswordHash != "" {
			return "••••••••"
		}
		return "(not set)"
	}
	return ""
}

// updateSettings handles keys on the settings screen.
func (a *App) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m := &a.settings

	if m.filtering {
		switch msg.String() {
		case "enter":
			m.filtering = false
		case "esc":
			m.filtering = false
			m.filter = ""
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.filter += string(msg.Runes)
			}
		}
		m.cursor, m.scroll = 0, 0
		return a, nil
	}

	defs := m.defs()
	if m.editing {
		if len(defs) == 0 {
			m.editing = false
			return a, nil
		}
		def := defs[m.cursor]
		if settingSecret(def) && msg.String() == "f2" {
			m.revealSecret = !m.revealSecret
			return a, nil
		}
		if m.choices != nil {
			switch msg.String() {
			case "left", "h", "-":
				m.choice = (m.choice - 1 + len(m.choices)) % len(m.choices)
			case "right", "l", "+", "tab", " ":
				m.choice = (m.choice + 1) % len(m.choices)
			case "enter":
				m.editing = false
				return a, a.saveSetting(def, m.choices[m.choice])
			case "esc":
				m.editing = false
			}
			return a, nil
		}
		switch msg.String() {
		case "enter":
			m.editing = false
			return a, a.saveSetting(def, strings.TrimSpace(m.buf))
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
	case "tab", "right", "l":
		m.changeTab(1)
	case "shift+tab", "left", "h":
		m.changeTab(-1)
	case "j", "down":
		if m.cursor < len(defs)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = maxInt(0, len(defs)-1)
	case "/":
		if settingTabs[m.tab].raw {
			m.filtering = true
		}
	case "enter", "e":
		if len(defs) == 0 {
			return a, nil
		}
		def := defs[m.cursor]
		if settingReadOnly(def) {
			a.pushToast(def.label+": "+a.settingValue(def), toastInfo)
			return a, nil
		}
		m.editing = true
		m.revealSecret = false
		m.buf = a.settingValue(def)
		if settingSecret(def) {
			m.buf = ""
		}
		m.choices = nil
		if def.choices != nil {
			m.choices = append([]string(nil), def.choices...)
			m.choice = 0
			cur := a.settingValue(def)
			found := false
			for i, choice := range m.choices {
				if normEq(choice, cur) {
					m.choice, found = i, true
					break
				}
			}
			if !found && cur != "" {
				m.choices = append([]string{cur}, m.choices...)
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

func (a *App) saveSetting(def settingDef, val string) tea.Cmd {
	if def.local {
		return a.saveLocalSetting(def, val)
	}
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

func (a *App) saveLocalSetting(def settingDef, val string) tea.Cmd {
	cfg := a.cfg
	client := a.client
	setDaemonDir := false
	toastText := def.label + " → " + val
	switch def.key {
	case localDownloadDir:
		if val == "" {
			return func() tea.Msg { return actionMsg{err: fmt.Errorf("download directory cannot be empty")} }
		}
		cfg.DownloadDir = val
		a.files = newFileBrowser(val)
		setDaemonDir = true
	case localPollMS:
		n, err := strconv.Atoi(val)
		if err != nil || n < 300 {
			return func() tea.Msg { return actionMsg{err: fmt.Errorf("refresh interval must be at least 300 ms")} }
		}
		cfg.PollMS = n
	case localSidebar:
		on, err := strconv.ParseBool(val)
		if err != nil {
			return func() tea.Msg { return actionMsg{err: err} }
		}
		a.sidebar = on
		cfg.Sidebar = on
	case localCompact:
		on, err := strconv.ParseBool(val)
		if err != nil {
			return func() tea.Msg { return actionMsg{err: err} }
		}
		cfg.CompactRows = on
	case localConfirm:
		on, err := strconv.ParseBool(val)
		if err != nil {
			return func() tea.Msg { return actionMsg{err: err} }
		}
		cfg.ConfirmRemove = on
	case localHistory:
		n, err := strconv.Atoi(val)
		if err != nil || n < 1 {
			return func() tea.Msg { return actionMsg{err: fmt.Errorf("history limit must be positive")} }
		}
		cfg.HistoryLimit = n
	case localWebHost:
		if val == "" {
			return func() tea.Msg { return actionMsg{err: fmt.Errorf("web listen address cannot be empty")} }
		}
		cfg.WebHost = val
	case localWebPort:
		n, err := strconv.Atoi(val)
		if err != nil || n < 1 || n > 65535 {
			return func() tea.Msg { return actionMsg{err: fmt.Errorf("web port must be between 1 and 65535")} }
		}
		cfg.WebPort = n
	case localWebPassword:
		if len([]rune(val)) < 6 {
			return func() tea.Msg { return actionMsg{err: fmt.Errorf("web password must be at least 6 characters")} }
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(val), bcrypt.DefaultCost)
		if err != nil {
			return func() tea.Msg { return actionMsg{err: err} }
		}
		cfg.WebPasswordHash = string(hash)
		toastText = "web password updated · restart tidefetch serve to apply"
	}
	return func() tea.Msg {
		if setDaemonDir {
			ctx, cancel := rpcCtx()
			err := client.ChangeGlobalOption(ctx, aria2.Options{aria2.OptDir: val})
			cancel()
			if err != nil {
				return actionMsg{err: err}
			}
		}
		if err := cfg.Save(); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{toast: toastText, level: toastOK}
	}
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

func (a *App) settingsTabLabel(tab settingTab) string {
	if a.width < 104 {
		return tab.short
	}
	return tab.name
}

// renderSettingsTabs uses real bordered chips on normal terminals, falling
// back to a compact segmented strip only when the viewport is very narrow.
// It returns the rendered block and its height for downstream hitbox math.
func (a *App) renderSettingsTabs(m *settingsModel) (string, int) {
	if a.width < 68 {
		var b strings.Builder
		x := 1
		for i, tab := range settingTabs {
			label := " " + tab.short + " "
			style := lipgloss.NewStyle().Foreground(cDim).Background(cSurface).Bold(false)
			if i == m.tab {
				style = style.Foreground(cBright).Background(cAccent).Bold(true)
			}
			seg := style.Render(label)
			a.addHit(x, a.bodyTop()+1, lipgloss.Width(seg), 1, fmt.Sprintf("stab:%d", i))
			x += lipgloss.Width(seg)
			b.WriteString(seg)
		}
		return " " + b.String(), 1
	}

	chips := make([]string, 0, len(settingTabs)*2+1)
	chips = append(chips, " ")
	x := 1
	for i, tab := range settingTabs {
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cFaint).
			Foreground(cDim).
			Padding(0, 1)
		if i == m.tab {
			style = style.
				BorderForeground(cAccent).
				Foreground(cBright).
				Background(cSurface2).
				Bold(true)
		}
		chip := style.Render(a.settingsTabLabel(tab))
		w := lipgloss.Width(chip)
		a.addHit(x, a.bodyTop()+1, w, 3, fmt.Sprintf("stab:%d", i))
		chips = append(chips, chip)
		x += w
		if i < len(settingTabs)-1 {
			chips = append(chips, " ")
			x++
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, chips...), 3
}

// viewSettings renders one category at a time with a clearly separated tab strip.
func (a *App) viewSettings(h int) string {
	m := &a.settings
	if m.loading {
		return lipgloss.Place(a.width, h, lipgloss.Center, lipgloss.Center, styleDim.Render("loading options…"))
	}

	var b strings.Builder
	fmt.Fprint(&b, styleTitle.Render("Settings"), styleDim.Render("  aria2 + terminal preferences"), "\n")

	tabBlock, tabH := a.renderSettingsTabs(m)
	b.WriteString(tabBlock)
	b.WriteString("\n")

	if settingTabs[m.tab].raw {
		filter := m.filter
		if m.filtering {
			filter += "▌"
		}
		fmt.Fprintln(&b, styleDim.Render(" / filter: ")+styleText.Render(filter)+
			styleFaint.Render("   · all live aria2 global options"))
	} else {
		fmt.Fprintln(&b, styleFaint.Render(" ←→ categories · ↑↓ navigate · enter edit · r refresh"))
	}

	defs := m.defs()
	m.clamp()
	listH := h - tabH - 5
	if listH < 2 {
		listH = 2
	}
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+listH {
		m.scroll = m.cursor - listH + 1
	}
	end := minInt(len(defs), m.scroll+listH)
	if len(defs) == 0 {
		fmt.Fprintln(&b, styleDim.Render("  no matching options"))
	}
	for i := m.scroll; i < end; i++ {
		def := defs[i]
		a.addHit(0, a.bodyTop()+tabH+2+(i-m.scroll), a.width, 1, fmt.Sprintf("srow:%d", i))
		val := a.settingValue(def)
		disp := val
		switch def.key {
		case aria2.OptMaxOverallDownloadLimit, aria2.OptMaxOverallUploadLimit,
			"bt-request-peer-speed-limit":
			disp = fmtLimit(val)
		case aria2.OptMinSplitSize:
			if bytes := parseLimit(val); bytes > 0 {
				disp = humanBytes(bytes)
			}
		}
		if disp == "" {
			disp = "(unset)"
		}
		labelW := 32
		if settingTabs[m.tab].raw {
			labelW = minInt(42, maxInt(24, a.width/2))
		}
		label := padRight(truncate(def.label, labelW-1), labelW)
		if i == m.cursor {
			if m.editing && m.choices != nil {
				sel := styleAccent.Render("‹ ") + styleTitle.Render(m.choices[m.choice]) + styleAccent.Render(" ›")
				b.WriteString(styleRowSel.Width(a.width - 2).Render(
					styleSelBar.Render("┃") + styleFieldFocus.Render(label) + sel +
						styleDim.Render("   ←→ choose · enter save")))
			} else if m.editing {
				editValue := m.buf
				if settingSecret(def) && !m.revealSecret {
					editValue = strings.Repeat("•", len([]rune(m.buf)))
				}
				hint := def.hint
				if settingSecret(def) {
					hint += " · F2 reveal"
				}
				b.WriteString(styleRowSel.Width(a.width - 2).Render(
					styleSelBar.Render("┃") + styleFieldFocus.Render(label) +
						styleText.Render(editValue) + styleAccent.Render("▌") +
						styleDim.Render("  "+hint)))
			} else {
				mode := "edit"
				if def.choices != nil {
					mode = "select"
				}
				if settingReadOnly(def) {
					mode = "status"
				}
				b.WriteString(styleRowSel.Width(a.width - 2).Render(
					styleSelBar.Render("┃") + styleFieldFocus.Render(label) + styleAccent2.Render(disp) +
						styleFaint.Render("  ‹"+mode+"› · "+def.hint)))
			}
		} else {
			fmt.Fprint(&b, " ", styleInputLabel.Render(label), styleText.Render(disp))
		}
		b.WriteString("\n")
	}

	fmt.Fprint(&b, "\n", styleDim.Render(" RPC: "), styleText.Render(a.rpcURL))
	if settingTabs[m.tab].raw {
		fmt.Fprint(&b, styleDim.Render(fmt.Sprintf("  · %d options", len(defs))))
	}
	if a.spawned {
		b.WriteString(styleDim.Render("  · spawned by tidefetch"))
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
		{"n", "create folder"},
	}) + "\n" + styleTitle.Render("Add form") + "\n" + col([][2]string{
		{"tab", "next field"},
		{"←/→", "change selector values"},
		{"ctrl+k", "check link (size · resumable?)"},
		{"ctrl+a", "advanced aria2 options"},
		{"ctrl+o", "browse directory"},
		{"ctrl+t", "pick .torrent/.metalink"},
		{"ctrl+s", "start"},
	}) + "\n" + styleTitle.Render("Settings") + "\n" + col([][2]string{
		{"←/→", "switch category"},
		{"enter", "edit selected option"},
		{"/", "filter Advanced options"},
	})

	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(44).Render(left),
		right)
	panel := stylePanel.Render(cols)
	return lipgloss.Place(a.width, h, lipgloss.Center, lipgloss.Top, panel)
}
