package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/turbostart/aria2c-tui/pkg/aria2"
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

var detailTabs = []string{"Info", "Files", "Peers", "Servers"}

// updateDetails handles keys in the details view.
func (a *App) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := &a.details
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
		d.cursor++
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
		a.addHit(x, bodyTop+2, lipgloss.Width(seg), 1, fmt.Sprintf("dtab:%d", i))
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
	default:
		body = d.renderInfo(innerW, bodyH, a.gidHist[d.gid])
	}

	head := styleTitle.Render(truncate(st.Name(), innerW))
	panel := stylePanel.Width(a.width - 2).Render(head + "\n" + tabs.String() + "\n" + body)
	return panel
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
		"  " + gauge(frac, minInt(w-12, 60)) + fmt.Sprintf(" %5.1f%%", frac*100),
	}
	if st.Status == aria2.StatusActive && len(hist) > 1 {
		gw := minInt(w-6, 72)
		lines = append(lines, "")
		for _, gl := range brailleGraph(hist, gw, 4, 0) {
			lines = append(lines, "  "+styleGraphTask.Render(gl))
		}
		lines = append(lines, "  "+styleDownArr.Render("▼ ")+styleText.Render(humanSpeed(st.DownloadSpeed.Int()))+
			styleDim.Render("  peak "+humanSpeed(int64(peakOf(hist)))))
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
