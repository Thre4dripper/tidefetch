package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/turbostart/aria2c-tui/pkg/aria2"
)

// updateDownloads handles keys on the main list (and app-global keys).
func (a *App) updateDownloads(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pageSize := a.height - 6
	if pageSize < 1 {
		pageSize = 1
	}

	switch msg.String() {

	// --- global navigation ---
	case "q", "ctrl+c":
		return a, a.quit(false)
	case "Q":
		a.confirmShutdown()
		return a, nil
	case "?":
		a.view = viewHelp
		return a, nil
	case "a":
		return a, a.gotoAdd()
	case "h":
		a.gotoHistory()
		return a, nil
	case "f":
		a.gotoFiles()
		return a, nil
	case "s":
		return a, a.gotoSettings()
	case "t":
		a.sidebar = !a.sidebar
		return a, nil
	case "1", "2", "3", "4":
		a.tab = int(msg.String()[0] - '1')
		a.cursor, a.scroll = 0, 0
		a.buildRows()
		return a, nil

	// --- cursor movement ---
	case "j", "down":
		if a.cursor < len(a.rows)-1 {
			a.cursor++
		}
	case "k", "up":
		if a.cursor > 0 {
			a.cursor--
		}
	case "g", "home":
		a.cursor = 0
	case "G", "end":
		a.cursor = len(a.rows) - 1
		if a.cursor < 0 {
			a.cursor = 0
		}
	case "ctrl+d", "pgdown":
		a.cursor += pageSize / 2
		if a.cursor > len(a.rows)-1 {
			a.cursor = len(a.rows) - 1
		}
		if a.cursor < 0 {
			a.cursor = 0
		}
	case "ctrl+u", "pgup":
		a.cursor -= pageSize / 2
		if a.cursor < 0 {
			a.cursor = 0
		}

	case "/":
		a.searching = true
		a.searchBuf = a.filter
		return a, nil
	case "S":
		a.sortMode = (a.sortMode + 1) % 5
		a.buildRows()
		return a, nil
	case "esc":
		if a.filter != "" {
			a.filter, a.searchBuf = "", ""
			a.buildRows()
		}
		return a, nil

	// --- actions on the selection ---
	case " ", "p":
		if st := a.selected(); st != nil {
			client := a.client
			gid := st.GID
			switch st.Status {
			case aria2.StatusActive, aria2.StatusWaiting:
				return a, doRPC("paused "+truncate(st.Name(), 40), func(ctx context.Context) error {
					return client.Pause(ctx, gid)
				})
			case aria2.StatusPaused:
				return a, doRPC("resumed "+truncate(st.Name(), 40), func(ctx context.Context) error {
					return client.Unpause(ctx, gid)
				})
			}
		}
	case "P":
		client := a.client
		return a, doRPC("paused all", func(ctx context.Context) error { return client.PauseAll(ctx) })
	case "R":
		client := a.client
		return a, doRPC("resumed all", func(ctx context.Context) error { return client.UnpauseAll(ctx) })

	case "enter", "i", "l":
		if st := a.selected(); st != nil {
			a.details.open(st.GID)
			a.view = viewDetails
			return a, detailCmd(a.client, st.GID)
		}

	case "x", "d", "delete", "backspace":
		if st := a.selected(); st != nil {
			s := *st
			client := a.client
			a.confirmAction("Remove download?", truncate(s.Name(), 60), func() tea.Cmd {
				return doRPC("removed "+truncate(s.Name(), 40), func(ctx context.Context) error {
					return removeAny(ctx, client, s)
				})
			})
		}
	case "D":
		if st := a.selected(); st != nil {
			s := *st
			client := a.client
			a.confirmAction("Remove AND delete files?",
				truncate(s.Name(), 60)+"\nFiles are erased from disk. This cannot be undone.",
				func() tea.Cmd {
					return doRPC("removed + deleted "+truncate(s.Name(), 40), func(ctx context.Context) error {
						if err := removeAny(ctx, client, s); err != nil {
							return err
						}
						return removeFilesOf(s)
					})
				})
		}

	case "r":
		if st := a.selected(); st != nil && st.Status == aria2.StatusError {
			s := *st
			client := a.client
			dir := s.Dir
			uris := collectURIs(s)
			if len(uris) == 0 {
				a.pushToast("no URI to retry (torrent?)", toastErr)
				return a, nil
			}
			return a, doRPC("retrying "+truncate(s.Name(), 40), func(ctx context.Context) error {
				opts := aria2.Options{}
				if dir != "" {
					opts[aria2.OptDir] = dir
				}
				if _, err := client.AddURI(ctx, uris, opts); err != nil {
					return err
				}
				return client.RemoveDownloadResult(ctx, s.GID)
			})
		}

	case "J":
		if st := a.selected(); st != nil && st.Status == aria2.StatusWaiting {
			client := a.client
			gid := st.GID
			return a, doRPC("moved down", func(ctx context.Context) error {
				_, err := client.ChangePosition(ctx, gid, 1, "POS_CUR")
				return err
			})
		}
	case "K":
		if st := a.selected(); st != nil && st.Status == aria2.StatusWaiting {
			client := a.client
			gid := st.GID
			return a, doRPC("moved up", func(ctx context.Context) error {
				_, err := client.ChangePosition(ctx, gid, -1, "POS_CUR")
				return err
			})
		}

	case "o":
		if st := a.selected(); st != nil {
			dir := st.Dir
			if dir == "" {
				dir = a.cfg.DownloadDir
			}
			if err := openPath(dir); err != nil {
				a.pushToast("open failed: "+err.Error(), toastErr)
			}
		}
	case "y":
		if st := a.selected(); st != nil {
			if uri := st.PrimaryURI(); uri != "" {
				if err := copyClipboard(uri); err != nil {
					a.pushToast("copy failed: "+err.Error(), toastErr)
				} else {
					a.pushToast("URL copied", toastOK)
				}
			} else if st.InfoHash != "" {
				magnet := "magnet:?xt=urn:btih:" + st.InfoHash
				_ = copyClipboard(magnet)
				a.pushToast("magnet link copied", toastOK)
			}
		}

	case "c":
		client := a.client
		a.confirmAction("Clear finished list?", "Purges completed / errored / removed results.", func() tea.Cmd {
			return doRPC("cleared results", func(ctx context.Context) error {
				return client.PurgeDownloadResult(ctx)
			})
		})
	case "w":
		client := a.client
		return a, doRPC("session saved", func(ctx context.Context) error { return client.SaveSession(ctx) })

	// --- global speed limits ---
	case "[", "]":
		up := msg.String() == "]"
		client := a.client
		return a, func() tea.Msg {
			ctx, cancel := rpcCtx()
			defer cancel()
			opts, err := client.GetGlobalOption(ctx)
			if err != nil {
				return actionMsg{err: err}
			}
			nv := stepLimit(opts[aria2.OptMaxOverallDownloadLimit], up)
			if err := client.ChangeGlobalOption(ctx, aria2.Options{aria2.OptMaxOverallDownloadLimit: nv}); err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{toast: "global ↓ limit: " + fmtLimit(nv), level: toastInfo}
		}
	case "{", "}":
		up := msg.String() == "}"
		client := a.client
		return a, func() tea.Msg {
			ctx, cancel := rpcCtx()
			defer cancel()
			opts, err := client.GetGlobalOption(ctx)
			if err != nil {
				return actionMsg{err: err}
			}
			nv := stepLimit(opts[aria2.OptMaxOverallUploadLimit], up)
			if err := client.ChangeGlobalOption(ctx, aria2.Options{aria2.OptMaxOverallUploadLimit: nv}); err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{toast: "global ↑ limit: " + fmtLimit(nv), level: toastInfo}
		}
	}
	return a, nil
}

// quit saves state and optionally shuts the daemon down.
func (a *App) quit(shutdownDaemon bool) tea.Cmd {
	client := a.client
	connected := a.connected
	_ = a.hist.Save()
	return func() tea.Msg {
		if connected {
			ctx, cancel := rpcCtx()
			defer cancel()
			_ = client.SaveSession(ctx)
			if shutdownDaemon {
				_ = client.Shutdown(ctx)
			}
			client.Close()
		}
		return tea.Quit()
	}
}

// removeAny removes a download regardless of its state.
func removeAny(ctx context.Context, c *aria2.Client, st aria2.Status) error {
	switch st.Status {
	case aria2.StatusActive, aria2.StatusWaiting, aria2.StatusPaused:
		if err := c.Remove(ctx, st.GID); err != nil {
			if err2 := c.ForceRemove(ctx, st.GID); err2 != nil {
				return err
			}
		}
		// Also clear it from the stopped list so it doesn't linger.
		_ = c.RemoveDownloadResult(ctx, st.GID)
		return nil
	default:
		return c.RemoveDownloadResult(ctx, st.GID)
	}
}

// collectURIs gathers unique source URIs of a download.
func collectURIs(st aria2.Status) []string {
	seen := map[string]bool{}
	var uris []string
	for _, f := range st.Files {
		for _, u := range f.URIs {
			if u.URI != "" && !seen[u.URI] {
				seen[u.URI] = true
				uris = append(uris, u.URI)
			}
		}
	}
	return uris
}

// sidebarW is the width of the right info panel.
const sidebarW = 42

// rowH is the height of one download card.
const rowH = 3

// viewDownloads renders the main list plus the optional side panel.
func (a *App) viewDownloads(h int) string {
	showSide := a.sidebar && a.width >= 100
	listW := a.width
	if showSide {
		listW = a.width - sidebarW
	}

	var list string
	if len(a.rows) == 0 {
		list = a.emptyState(listW, h)
	} else {
		list = a.renderList(listW, h)
	}

	if !showSide {
		return list
	}
	side := a.renderSidebar(sidebarW, h)
	listBox := lipgloss.NewStyle().Width(listW).MaxWidth(listW).Height(h).MaxHeight(h).Render(list)
	return lipgloss.JoinHorizontal(lipgloss.Top, listBox, side)
}

// emptyState is a small onboarding screen shown when the list is empty.
func (a *App) emptyState(w, h int) string {
	if a.filter != "" {
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			styleDim.Render("no matches for ")+styleAccent.Render("/"+a.filter)+
				styleDim.Render("  —  esc clears the search"))
	}
	row := func(k, d string) string {
		return styleKey.Render(padLeft(k, 8)) + "   " + styleDesc.Render(d)
	}
	tut := lipgloss.JoinVertical(lipgloss.Left,
		styleLogo.Render("⬡ aria2tui")+styleDim.Render("  nothing downloading yet"),
		"",
		row("a", "add downloads — URLs, magnet links, .torrent files"),
		row("ctrl+o", "…browse for a save folder inside the add form"),
		row("1–4", "filter tabs · "+styleKey.Render("t")+styleDesc.Render(" toggles the side panel")),
		row("?", "every keyboard shortcut"),
		"",
		styleFaint.Render("mouse works too: click tabs, rows and the buttons below"),
	)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, tut)
}

// renderList paints the download cards inside a titled box and records hitboxes.
func (a *App) renderList(w, h int) string {
	innerW := w - 4
	listH := h - 2 // box borders
	visible := listH / rowH
	if visible < 1 {
		visible = 1
	}
	if a.cursor < a.scroll {
		a.scroll = a.cursor
	}
	if a.cursor >= a.scroll+visible {
		a.scroll = a.cursor - visible + 1
	}
	if a.scroll < 0 {
		a.scroll = 0
	}

	var lines []string
	end := a.scroll + visible
	if end > len(a.rows) {
		end = len(a.rows)
	}
	for i := a.scroll; i < end; i++ {
		y := a.bodyTop() + 1 + (i-a.scroll)*rowH
		a.addHit(2, y, innerW, rowH, fmt.Sprintf("row:%d", i))
		lines = append(lines, strings.Split(a.renderRow(a.rows[i], i == a.cursor, innerW), "\n")...)
	}

	title := fmt.Sprintf("Downloads · %d", len(a.rows))
	if len(a.rows) > visible {
		title = fmt.Sprintf("Downloads · %d/%d", a.cursor+1, len(a.rows))
	}
	border := styleBoxBorder
	if a.view == viewDownloads {
		border = styleBoxBorderHi
	}
	return titledBox(title, lines, w, border, styleBoxTitle)
}

// renderRow paints one download as a three-line card.
func (a *App) renderRow(st aria2.Status, selected bool, w int) string {
	seeding := st.Status == aria2.StatusActive && bool(st.Seeder)
	stStyle, icon, label := statusStyle(st.Status, seeding)

	marker := "  "
	if selected {
		marker = styleSelBar.Render("┃ ")
	}

	// Line 1 — icon · name · badges
	badge := stStyle.Bold(true).Render(label)
	if st.Status == aria2.StatusError && st.ErrorCode != "" {
		badge += styleDim.Render(" #" + st.ErrorCode)
	}
	torrentTag := ""
	if st.IsTorrent() {
		torrentTag = styleAccent2.Render(" ⧉")
	}
	nameW := w - 4 - lipgloss.Width(badge) - lipgloss.Width(torrentTag) - 2
	if nameW < 8 {
		nameW = 8
	}
	nameStyle := styleTitle
	if !selected {
		nameStyle = styleText
	}
	line1 := marker + stStyle.Render(icon) + " " + nameStyle.Render(padRight(truncate(st.Name(), nameW), nameW)) +
		torrentTag + " " + badge

	// Line 2 — gauge · % · sizes · speed/eta · per-task minigraph
	frac := st.Progress()
	pct := fmt.Sprintf("%5.1f%%", frac*100)
	sizes := fmt.Sprintf("%s / %s", humanBytes(st.CompletedLength.Int()), humanBytes(st.TotalLength.Int()))
	if st.TotalLength == 0 {
		sizes = humanBytes(st.CompletedLength.Int())
	}
	var tail string
	switch st.Status {
	case aria2.StatusActive:
		if seeding {
			tail = fmt.Sprintf("↑ %s seeding", humanSpeed(st.UploadSpeed.Int()))
		} else {
			eta := humanETA(st.TotalLength.Int()-st.CompletedLength.Int(), st.DownloadSpeed.Int())
			tail = fmt.Sprintf("%s  eta %s", humanSpeed(st.DownloadSpeed.Int()), eta)
		}
	case aria2.StatusError:
		tail = "failed"
	case aria2.StatusComplete:
		tail = "completed"
	case aria2.StatusPaused:
		tail = "paused"
	case aria2.StatusWaiting:
		tail = "queued"
	}
	graph := ""
	if st.Status == aria2.StatusActive && !seeding {
		graph = " " + styleGraphTask.Render(sparkline(a.gidHist[st.GID], 14))
	}
	right := styleDim.Render(sizes) + "  " + stStyle.Render(tail) + graph
	gaugeW := w - 4 - 7 - lipgloss.Width(right) - 4
	if gaugeW < 10 {
		gaugeW = 10
	}
	line2 := "  " + gauge(frac, gaugeW) + " " + styleText.Render(pct) + "  " + right

	// Line 3 — context: location · source · connections / error detail
	var info string
	switch st.Status {
	case aria2.StatusActive:
		info = shortPath(st.Dir) + " · " + hostOf(st.PrimaryURI())
		if st.IsTorrent() {
			info += fmt.Sprintf(" · %d peers · %d seeds", st.Connections.Int(), st.NumSeeders.Int())
		} else if st.Connections > 0 {
			info += fmt.Sprintf(" · %d conn", st.Connections.Int())
		}
	case aria2.StatusError:
		info = truncate(st.ErrorMessage, w-8)
	case aria2.StatusWaiting:
		info = shortPath(st.Dir) + " · waiting for a download slot"
	case aria2.StatusPaused:
		info = shortPath(st.Dir) + " · press ␣ to resume"
	default:
		info = shortPath(st.Dir) + " · " + hostOf(st.PrimaryURI())
	}
	infoStyle := styleFaint
	if st.Status == aria2.StatusError {
		infoStyle = styleBad
	}
	line3 := "    " + infoStyle.Render(truncate(info, w-6))

	if selected {
		line1 = styleRowSel.Width(w).Render(line1)
		line2 = styleRowSel.Width(w).Render(line2)
		line3 = styleRowSel.Width(w).Render(line3)
	}
	return line1 + "\n" + line2 + "\n" + line3
}

// renderSidebar paints the btop-style stacked info blocks.
func (a *App) renderSidebar(w, h int) string {
	inner := w - 4
	var blocks []string

	// --- Selected task -------------------------------------------------
	if st := a.selected(); st != nil {
		stStyle, _, label := statusStyle(st.Status, st.Status == aria2.StatusActive && bool(st.Seeder))
		lines := []string{
			styleTitle.Render(truncate(st.Name(), inner)),
			stStyle.Bold(true).Render(label) + styleDim.Render(" · "+humanBytes(st.TotalLength.Int())),
		}
		if st.Status == aria2.StatusActive {
			for _, gl := range brailleGraph(a.gidHist[st.GID], inner, 4, 0) {
				lines = append(lines, styleGraphTask.Render(gl))
			}
			lines = append(lines,
				styleDownArr.Render("▼ ")+styleText.Render(humanSpeed(st.DownloadSpeed.Int()))+
					styleDim.Render("  peak "+humanSpeed(int64(peakOf(a.gidHist[st.GID])))),
			)
		} else {
			lines = append(lines, "")
		}
		lines = append(lines,
			gauge(st.Progress(), inner-7)+fmt.Sprintf(" %5.1f%%", st.Progress()*100),
		)
		ctx := humanETA(st.TotalLength.Int()-st.CompletedLength.Int(), st.DownloadSpeed.Int())
		meta := "eta " + ctx + " · conn " + fmt.Sprint(st.Connections.Int())
		if st.IsTorrent() {
			meta += " · seeds " + fmt.Sprint(st.NumSeeders.Int())
		}
		lines = append(lines,
			styleDim.Render(truncate(meta, inner)),
			styleFaint.Render(truncate(shortPath(st.Dir), inner)),
		)
		blocks = append(blocks, titledBox("Selected", lines, w, styleBoxBorderHi, styleBoxTitle))
	}

	// --- Pieces (Surge-style chunk map) -----------------------------------
	if st := a.selected(); st != nil && st.Bitfield != "" && st.NumPieces > 0 &&
		(st.Status == aria2.StatusActive || st.Status == aria2.StatusPaused) {
		lines := chunkMap(st.Bitfield, int(st.NumPieces.Int()), inner, 3)
		lines = append(lines, styleDim.Render(fmt.Sprintf("%d pieces × %s",
			st.NumPieces.Int(), humanBytes(st.PieceLength.Int()))))
		blocks = append(blocks, titledBox("Pieces", lines, w, styleBoxBorder, styleBoxTitle))
	}

	// --- Traffic ---------------------------------------------------------
	{
		var lines []string
		for _, gl := range brailleGraph(a.speedHist, inner, 4, 0) {
			lines = append(lines, styleGraphDown.Render(gl))
		}
		lines = append(lines,
			styleDownArr.Render("▼ ")+styleText.Render(humanSpeed(a.stat.DownloadSpeed.Int()))+
				styleDim.Render("  peak "+humanSpeed(int64(peakOf(a.speedHist)))))
		for _, gl := range brailleGraph(a.upHist, inner, 2, 0) {
			lines = append(lines, styleGraphUp.Render(gl))
		}
		lines = append(lines,
			styleUpArr.Render("▲ ")+styleText.Render(humanSpeed(a.stat.UploadSpeed.Int())))
		blocks = append(blocks, titledBox("Traffic", lines, w, styleBoxBorder, styleBoxTitle))
	}

	// --- Disk -------------------------------------------------------------
	{
		dir := a.cfg.DownloadDir
		if st := a.selected(); st != nil && st.Dir != "" {
			dir = st.Dir
		}
		var lines []string
		if free, total, err := diskUsage(dir); err == nil && total > 0 {
			lines = []string{
				styleFaint.Render(truncate(shortPath(dir), inner)),
				usageBar(total-free, total, inner),
				styleText.Render(humanBytes(free)) + styleDim.Render(" free of "+humanBytes(total)),
			}
		} else {
			lines = []string{styleFaint.Render(truncate(shortPath(dir), inner)), styleDim.Render("disk info unavailable")}
		}
		blocks = append(blocks, titledBox("Disk", lines, w, styleBoxBorder, styleBoxTitle))
	}

	// --- Session ------------------------------------------------------------
	{
		up := time.Since(a.startedAt).Round(time.Second)
		lines := []string{
			styleText.Render("aria2 "+a.version) + styleDim.Render(" · up "+up.String()),
			styleFaint.Render(truncate(strings.TrimPrefix(a.rpcURL, "ws://"), inner)),
			styleDim.Render(fmt.Sprintf("active %d · queued %d · stopped %d",
				a.stat.NumActive.Int(), a.stat.NumWaiting.Int(), a.stat.NumStopped.Int())),
		}
		blocks = append(blocks, titledBox("Session", lines, w, styleBoxBorder, styleBoxTitle))
	}

	// --- Log ------------------------------------------------------------------
	if len(a.log) > 0 {
		var lines []string
		start := maxInt(0, len(a.log)-4)
		for _, e := range a.log[start:] {
			dot := styleAccent2.Render("●")
			switch e.level {
			case toastOK:
				dot = styleGood.Render("●")
			case toastErr:
				dot = styleBad.Render("●")
			}
			lines = append(lines, dot+styleFaint.Render(" "+e.at.Format("15:04:05"))+" "+
				styleDim.Render(truncate(e.text, inner-11)))
		}
		blocks = append(blocks, titledBox("Log", lines, w, styleBoxBorder, styleBoxTitle))
	}

	joined := strings.Join(blocks, "\n")
	// Clip to available height.
	rows := strings.Split(joined, "\n")
	if len(rows) > h {
		rows = rows[:h]
	}
	return strings.Join(rows, "\n")
}

// peakOf returns the maximum sample.
func peakOf(s []float64) float64 {
	var m float64
	for _, v := range s {
		if v > m {
			m = v
		}
	}
	return m
}

// hostOf extracts the host of a URI for compact display.
func hostOf(uri string) string {
	if uri == "" {
		return "local"
	}
	if strings.HasPrefix(uri, "magnet:") {
		return "magnet"
	}
	if i := strings.Index(uri, "://"); i >= 0 {
		rest := uri[i+3:]
		if j := strings.IndexAny(rest, "/?#"); j >= 0 {
			rest = rest[:j]
		}
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		return rest
	}
	return truncate(uri, 24)
}

// shortPath abbreviates $HOME to ~.
func shortPath(p string) string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
