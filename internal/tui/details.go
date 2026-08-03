package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Thre4dripper/tidefetch/pkg/aria2"
)

// detailsModel shows one download in depth: info, files, peers, servers.
type detailsModel struct {
	gid     string
	status  aria2.Status
	peers   []aria2.Peer
	servers []aria2.ServerGroup
	options aria2.Options
	loaded  bool
	tab     int // 0 info, 1 files, 2 peers, 3 servers
	cursor  int
	scroll  int
	width   int
	height  int
	editing bool
	editBuf string
}

func newDetailsModel() detailsModel { return detailsModel{} }

func (d *detailsModel) setSize(w, h int) { d.width, d.height = w, h }

func (d *detailsModel) open(gid string) {
	if d.gid != gid {
		*d = detailsModel{gid: gid, width: d.width, height: d.height}
	}
}

func (d *detailsModel) absorb(msg detailMsg) {
	if msg.err != nil || msg.gid != d.gid {
		return
	}
	d.status = msg.status
	d.peers = msg.peers
	d.servers = msg.servers
	if msg.options != nil {
		d.options = msg.options
	}
	d.loaded = true
}

var detailTabs = []string{"Info", "Files", "Peers", "Servers", "Options"}

func (d *detailsModel) optionKeys() []string {
	keys := make([]string, 0, len(d.options))
	for key := range d.options {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// updateDetails handles keys in the details view.
func (a *App) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := &a.details
	if d.editing {
		keys := d.optionKeys()
		if d.cursor < 0 || d.cursor >= len(keys) {
			d.editing = false
			return a, nil
		}
		key := keys[d.cursor]
		switch msg.String() {
		case "enter":
			d.editing = false
			client := a.client
			gid := d.gid
			value := strings.TrimSpace(d.editBuf)
			return a, tea.Sequence(
				doRPC(key+" → "+value, func(ctx context.Context) error {
					return client.ChangeOption(ctx, gid, aria2.Options{key: value})
				}),
				detailCmd(client, gid),
			)
		case "esc":
			d.editing = false
		case "backspace":
			if len(d.editBuf) > 0 {
				d.editBuf = d.editBuf[:len(d.editBuf)-1]
			}
		default:
			if msg.Type == tea.KeyRunes {
				d.editBuf += string(msg.Runes)
			}
		}
		return a, nil
	}
	switch msg.String() {
	case "esc", "q":
		a.view = viewDownloads
		return a, nil
	case "Q":
		a.confirmShutdown()
		return a, nil
	case "tab", "right":
		d.tab = (d.tab + 1) % len(detailTabs)
		d.cursor, d.scroll = 0, 0
		return a, nil
	case "shift+tab", "left":
		d.tab = (d.tab - 1 + len(detailTabs)) % len(detailTabs)
		d.cursor, d.scroll = 0, 0
		return a, nil
	case "j", "down":
		if d.tab == 4 {
			if d.cursor < len(d.optionKeys())-1 {
				d.cursor++
			}
		} else {
			d.cursor++
		}
	case "k", "up":
		if d.cursor > 0 {
			d.cursor--
		}
	case " ":
		// Toggle file selection (torrents).
		if d.tab == 1 && d.cursor < len(d.status.Files) {
			target := d.status.Files[d.cursor]
			var selected []string
			for _, f := range d.status.Files {
				sel := bool(f.Selected)
				if f.Index == target.Index {
					sel = !sel
				}
				if sel {
					selected = append(selected, fmt.Sprintf("%d", f.Index.Int()))
				}
			}
			client := a.client
			gid := d.gid
			val := strings.Join(selected, ",")
			return a, tea.Sequence(
				doRPC("file selection updated", func(ctx context.Context) error {
					return client.ChangeOption(ctx, gid, aria2.Options{aria2.OptSelectFile: val})
				}),
				detailCmd(a.client, gid),
			)
		}
	case "enter", "e":
		if d.tab == 4 {
			keys := d.optionKeys()
			if d.cursor >= 0 && d.cursor < len(keys) {
				d.editing = true
				d.editBuf = d.options[keys[d.cursor]]
			}
		}
	case "+", "=", "-", "_":
		up := msg.String() == "+" || msg.String() == "="
		client := a.client
		gid := d.gid
		cur := d.options[aria2.OptMaxDownloadLimit]
		nv := stepLimit(cur, up)
		return a, tea.Sequence(
			doRPC("↓ limit: "+fmtLimit(nv), func(ctx context.Context) error {
				return client.ChangeOption(ctx, gid, aria2.Options{aria2.OptMaxDownloadLimit: nv})
			}),
			detailCmd(a.client, gid),
		)
	case "p":
		st := d.status
		client := a.client
		switch st.Status {
		case aria2.StatusActive, aria2.StatusWaiting:
			return a, doRPC("paused", func(ctx context.Context) error { return client.Pause(ctx, st.GID) })
		case aria2.StatusPaused:
			return a, doRPC("resumed", func(ctx context.Context) error { return client.Unpause(ctx, st.GID) })
		}
	case "o":
		dir := d.status.Dir
		if dir != "" {
			_ = openPath(dir)
		}
	case "y":
		if uri := d.status.PrimaryURI(); uri != "" {
			_ = copyClipboard(uri)
			a.pushToast("URL copied", toastOK)
		}
	case "x":
		st := d.status
		client := a.client
		if !a.cfg.ConfirmRemove {
			a.view = viewDownloads
			return a, doRPC("removed "+truncate(st.Name(), 40), func(ctx context.Context) error {
				return removeAny(ctx, client, st)
			})
		}
		a.confirmAction("Remove download?", truncate(st.Name(), 60), func() tea.Cmd {
			a.view = viewDownloads
			return doRPC("removed "+truncate(st.Name(), 40), func(ctx context.Context) error {
				return removeAny(ctx, client, st)
			})
		})
	case "D":
		st := d.status
		client := a.client
		a.confirmAction("Remove AND delete files?",
			truncate(st.Name(), 60)+"\nFiles are erased from disk. This cannot be undone.",
			func() tea.Cmd {
				a.view = viewDownloads
				return doRPC("removed + deleted "+truncate(st.Name(), 40), func(ctx context.Context) error {
					if err := removeAny(ctx, client, st); err != nil {
						return err
					}
					return removeFilesOf(st)
				})
			})
	}
	return a, nil
}

// viewDetails renders the details screen.
func (a *App) viewDetails(h int) string {
	d := &a.details
	if !d.loaded {
		return lipgloss.Place(a.width, h, lipgloss.Center, lipgloss.Center, styleDim.Render("loading…"))
	}
	st := d.status

	// tab strip (clickable)
	var tabs strings.Builder
	x := 2 // panel border + padding
	for i, t := range detailTabs {
		var seg string
		if i == d.tab {
			seg = styleTabActive.Render(t)
		} else {
			seg = styleTab.Render(t)
		}
		a.addHit(x, a.bodyTop()+2, lipgloss.Width(seg), 1, fmt.Sprintf("dtab:%d", i))
		x += lipgloss.Width(seg)
		tabs.WriteString(seg)
	}

	innerW := a.width - 6
	bodyH := h - 4
	if bodyH < 3 {
		bodyH = 3
	}

	var body string
	switch d.tab {
	case 1:
		body = d.renderFiles(innerW, bodyH)
	case 2:
		body = d.renderPeers(innerW, bodyH)
	case 3:
		body = d.renderServers(innerW, bodyH)
	case 4:
		body = d.renderOptions(innerW, bodyH)
	default:
		body = d.renderInfo(innerW, bodyH, a.gidHist[d.gid])
	}

	head := styleTitle.Render(truncate(st.Name(), innerW))
	panel := stylePanel.Width(a.width - 2).Render(head + "\n" + tabs.String() + "\n" + body)
	return panel
}

func (d *detailsModel) renderOptions(w, h int) string {
	keys := d.optionKeys()
	if len(keys) == 0 {
		return styleDim.Render("  no task options")
	}
	if d.cursor >= len(keys) {
		d.cursor = len(keys) - 1
	}
	listH := h - 2
	if listH < 1 {
		listH = 1
	}
	if d.cursor < d.scroll {
		d.scroll = d.cursor
	}
	if d.cursor >= d.scroll+listH {
		d.scroll = d.cursor - listH + 1
	}
	var lines []string
	lines = append(lines, styleFaint.Render("  ↑↓ navigate · enter edit · esc cancel"))
	end := minInt(len(keys), d.scroll+listH)
	keyW := minInt(40, maxInt(24, w/2))
	for i := d.scroll; i < end; i++ {
		key := keys[i]
		label := padRight(truncate(key, keyW-1), keyW)
		value := d.options[key]
		var row string
		if i == d.cursor && d.editing {
			row = styleSelBar.Render("┃") + styleFieldFocus.Render(label) +
				styleText.Render(d.editBuf) + styleAccent.Render("▌")
		} else if i == d.cursor {
			row = styleRowSel.Width(w - 2).Render(styleSelBar.Render("┃") +
				styleFieldFocus.Render(label) + styleAccent2.Render(truncate(value, w-keyW-6)))
		} else {
			row = " " + styleInputLabel.Render(label) + styleText.Render(truncate(value, w-keyW-6))
		}
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

func kv(k, v string) string {
	return styleDim.Render(padRight(k, 14)) + styleText.Render(v)
}

func (d *detailsModel) renderInfo(w, h int, hist []float64) string {
	st := d.status
	_, _, label := statusStyle(st.Status, bool(st.Seeder))
	frac := st.Progress()

	lines := []string{
		"",
		"  " + gauge(frac, minInt(w-12, 60), gaugeStyle(st.Status)) + fmt.Sprintf(" %5.1f%%", frac*100),
	}
	if len(hist) > 1 {
		gw := minInt(w-6, 72)
		lines = append(lines, "")
		series := downsampleHistory(hist, gw*2)
		tint := graphStyle(st.Status, bool(st.Seeder))
		for _, gl := range brailleGraph(series, gw, 4, 0) {
			lines = append(lines, "  "+tint.Render(gl))
		}
		graphLabel := "▼ " + humanSpeed(st.DownloadSpeed.Int())
		if st.Status != aria2.StatusActive {
			graphLabel = "final"
		}
		lines = append(lines, "  "+tint.Bold(true).Render(graphLabel)+
			styleDim.Render("  · peak "+humanSpeed(int64(peakOf(hist)))))
	}
	lines = append(lines,
		"",
		"  "+kv("status", label),
		"  "+kv("size", fmt.Sprintf("%s / %s", humanBytes(st.CompletedLength.Int()), humanBytes(st.TotalLength.Int()))),
		"  "+kv("speed", fmt.Sprintf("↓ %s   ↑ %s", humanSpeed(st.DownloadSpeed.Int()), humanSpeed(st.UploadSpeed.Int()))),
		"  "+kv("eta", humanETA(st.TotalLength.Int()-st.CompletedLength.Int(), st.DownloadSpeed.Int())),
		"  "+kv("connections", fmt.Sprintf("%d", st.Connections.Int())),
		"  "+kv("dir", truncate(st.Dir, w-18)),
		"  "+kv("gid", st.GID),
	)
	if lim := d.options[aria2.OptMaxDownloadLimit]; lim != "" && lim != "0" {
		lines = append(lines, "  "+kv("↓ limit", fmtLimit(lim)))
	}
	if st.IsTorrent() {
		lines = append(lines,
			"  "+kv("infohash", st.InfoHash),
			"  "+kv("seeders", fmt.Sprintf("%d", st.NumSeeders.Int())),
			"  "+kv("pieces", fmt.Sprintf("%d × %s", st.NumPieces.Int(), humanBytes(st.PieceLength.Int()))),
		)
		if pb := piecesBar(st.Bitfield, int(st.NumPieces.Int()), minInt(w-16, 60)); pb != "" {
			lines = append(lines, "", "  "+styleDim.Render("pieces        ")+pb)
		}
	}
	if st.ErrorMessage != "" {
		lines = append(lines, "", "  "+styleBad.Render("error: "+truncate(st.ErrorMessage, w-10)))
	}
	if uri := st.PrimaryURI(); uri != "" {
		lines = append(lines, "", "  "+kv("url", truncate(uri, w-18)))
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}

func (d *detailsModel) renderFiles(w, h int) string {
	files := d.status.Files
	if len(files) == 0 {
		return styleDim.Render("  no file info")
	}
	if d.cursor >= len(files) {
		d.cursor = len(files) - 1
	}
	if d.cursor < d.scroll {
		d.scroll = d.cursor
	}
	if d.cursor >= d.scroll+h {
		d.scroll = d.cursor - h + 1
	}
	var b strings.Builder
	end := minInt(d.scroll+h, len(files))
	for i := d.scroll; i < end; i++ {
		f := files[i]
		mark := styleDim.Render("☐")
		if f.Selected {
			mark = styleGood.Render("☑")
		}
		var frac float64
		if f.Length > 0 {
			frac = float64(f.CompletedLength) / float64(f.Length)
		}
		pct := fmt.Sprintf("%4.0f%%", frac*100)
		size := padLeft(humanBytes(f.Length.Int()), 9)
		name := truncate(strings.TrimPrefix(f.Path, d.status.Dir+"/"), w-24)
		line := fmt.Sprintf(" %s %s %s  %s", mark, pct, size, name)
		if i == d.cursor {
			line = styleRowSel.Width(w).Render(styleSelBar.Render("┃") + line[1:])
		}
		fmt.Fprintln(&b, line)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (d *detailsModel) renderPeers(w, h int) string {
	if len(d.peers) == 0 {
		return styleDim.Render("  no peers connected")
	}
	var b strings.Builder
	fmt.Fprintln(&b, styleDim.Render(fmt.Sprintf("  %-21s %10s %10s  %s", "address", "↓ speed", "↑ speed", "flags")))
	end := minInt(h-1, len(d.peers))
	for i := 0; i < end; i++ {
		p := d.peers[i]
		flags := ""
		if p.Seeder {
			flags += "seed "
		}
		if p.AmChoking {
			flags += "choking "
		}
		if p.PeerChoking {
			flags += "choked"
		}
		b.WriteString(fmt.Sprintf("  %-21s %10s %10s  %s\n",
			truncate(p.IP+":"+fmt.Sprint(p.Port.Int()), 21),
			humanSpeed(p.DownloadSpeed.Int()), humanSpeed(p.UploadSpeed.Int()),
			styleDim.Render(flags)))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (d *detailsModel) renderServers(w, h int) string {
	if len(d.servers) == 0 {
		return styleDim.Render("  no active server connections")
	}
	var b strings.Builder
	n := 0
	for _, grp := range d.servers {
		for _, s := range grp.Servers {
			if n >= h {
				break
			}
			b.WriteString(fmt.Sprintf("  %s %s\n",
				padLeft(humanSpeed(s.DownloadSpeed.Int()), 10),
				styleText.Render(truncate(s.CurrentURI, w-14))))
			n++
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
