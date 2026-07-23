package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// humanBytes renders 1536 → "1.5 KB".
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	val := float64(n) / float64(div)
	suffix := []string{"KB", "MB", "GB", "TB", "PB"}[exp]
	if val >= 100 {
		return fmt.Sprintf("%.0f %s", val, suffix)
	}
	return fmt.Sprintf("%.1f %s", val, suffix)
}

// humanSpeed renders bytes/sec.
func humanSpeed(n int64) string {
	if n <= 0 {
		return "0 B/s"
	}
	return humanBytes(n) + "/s"
}

// humanETA renders remaining time compactly.
func humanETA(remaining int64, speed int64) string {
	if speed <= 0 || remaining <= 0 {
		return "--"
	}
	d := time.Duration(remaining/speed) * time.Second
	switch {
	case d >= 100*time.Hour:
		return "∞"
	case d >= time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	case d >= time.Minute:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}

// gauge renders a smooth progress bar of the given cell width using eighth blocks.
func gauge(frac float64, width int) string {
	if width < 1 {
		return ""
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	cells := frac * float64(width)
	full := int(cells)
	rem := cells - float64(full)
	partials := []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}
	var b strings.Builder
	b.WriteString(strings.Repeat("█", full))
	if full < width {
		idx := int(rem * 8)
		if idx > 0 {
			b.WriteRune(partials[idx])
		} else {
			b.WriteRune(' ')
		}
		if pad := width - full - 1; pad > 0 {
			b.WriteString(strings.Repeat("─", pad))
		}
	}
	s := b.String()
	// Split into on/off parts for coloring.
	on := full
	if full < width && int(rem*8) > 0 {
		on++
	}
	return styleGaugeOn.Render(string([]rune(s)[:on])) + styleGaugeOff.Render(string([]rune(s)[on:]))
}

var sparkChars = []rune("▁▂▃▄▅▆▇█")

// sparkline renders a mini chart of the sample window scaled to its max.
func sparkline(samples []float64, width int) string {
	if width < 1 {
		return ""
	}
	if len(samples) > width {
		samples = samples[len(samples)-width:]
	}
	var max float64
	for _, v := range samples {
		if v > max {
			max = v
		}
	}
	var b strings.Builder
	for i := 0; i < width-len(samples); i++ {
		b.WriteRune(' ')
	}
	for _, v := range samples {
		if max <= 0 || v <= 0 {
			b.WriteRune('▁')
			continue
		}
		idx := int(v / max * float64(len(sparkChars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		b.WriteRune(sparkChars[idx])
	}
	return b.String()
}

// truncate cuts s to max display cells adding an ellipsis.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > max {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// padRight pads s with spaces to exactly width cells (truncating if needed).
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w > width {
		return truncate(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}

// padLeft right-aligns s in width cells.
func padLeft(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// bitfieldBits expands aria2's hex bitfield into per-piece booleans.
func bitfieldBits(bitfield string, numPieces int) []bool {
	if bitfield == "" || numPieces <= 0 {
		return nil
	}
	bits := make([]bool, 0, len(bitfield)*4)
	for _, c := range strings.ToLower(bitfield) {
		var v int
		switch {
		case c >= '0' && c <= '9':
			v = int(c - '0')
		case c >= 'a' && c <= 'f':
			v = int(c-'a') + 10
		default:
			continue
		}
		for i := 3; i >= 0; i-- {
			bits = append(bits, v&(1<<i) != 0)
		}
	}
	if len(bits) > numPieces {
		bits = bits[:numPieces]
	}
	return bits
}

// piecesBar renders a torrent bitfield (hex) into a density bar of the given width.
func piecesBar(bitfield string, numPieces int, width int) string {
	bits := bitfieldBits(bitfield, numPieces)
	if len(bits) == 0 || width <= 0 {
		return ""
	}
	var b strings.Builder
	per := float64(len(bits)) / float64(width)
	if per <= 0 {
		per = 1
	}
	shades := []rune(" ░▒▓█")
	for i := 0; i < width; i++ {
		lo := int(float64(i) * per)
		hi := int(float64(i+1) * per)
		if hi > len(bits) {
			hi = len(bits)
		}
		if lo >= hi {
			b.WriteRune(' ')
			continue
		}
		have := 0
		for _, set := range bits[lo:hi] {
			if set {
				have++
			}
		}
		frac := float64(have) / float64(hi-lo)
		idx := int(frac * float64(len(shades)-1))
		b.WriteRune(shades[idx])
	}
	return styleAccent.Render(b.String())
}

// chunkMap renders a Surge-style grid of piece cells: done, partial, missing.
func chunkMap(bitfield string, numPieces, cols, rows int) []string {
	bits := bitfieldBits(bitfield, numPieces)
	if len(bits) == 0 || cols <= 0 || rows <= 0 {
		return nil
	}
	cells := cols * rows
	per := float64(len(bits)) / float64(cells)
	if per <= 0 {
		per = 1
	}
	out := make([]string, 0, rows)
	for r := 0; r < rows; r++ {
		var line strings.Builder
		for c := 0; c < cols; c++ {
			idx := r*cols + c
			lo := int(float64(idx) * per)
			hi := int(float64(idx+1) * per)
			if hi > len(bits) {
				hi = len(bits)
			}
			if lo >= hi {
				line.WriteString(styleFaint.Render("·"))
				continue
			}
			have := 0
			for _, set := range bits[lo:hi] {
				if set {
					have++
				}
			}
			switch {
			case have == hi-lo:
				line.WriteString(styleGood.Render("▪"))
			case have > 0:
				line.WriteString(styleWarn.Render("▪"))
			default:
				line.WriteString(styleFaint.Render("·"))
			}
		}
		out = append(out, line.String())
	}
	return out
}

// stepLimit adjusts a "max-*-limit" value (bytes, 0 = unlimited) by ±step.
func stepLimit(current string, up bool) string {
	cur := parseLimit(current)
	const step = 256 * 1024 // 256 KiB
	if up {
		cur += step
	} else {
		cur -= step
		if cur < 0 {
			cur = 0
		}
	}
	if cur == 0 {
		return "0"
	}
	return fmt.Sprintf("%dK", cur/1024)
}

func parseLimit(v string) int64 {
	v = strings.TrimSpace(strings.ToUpper(v))
	if v == "" || v == "0" {
		return 0
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(v, "K"):
		mult, v = 1024, strings.TrimSuffix(v, "K")
	case strings.HasSuffix(v, "M"):
		mult, v = 1024*1024, strings.TrimSuffix(v, "M")
	case strings.HasSuffix(v, "G"):
		mult, v = 1024*1024*1024, strings.TrimSuffix(v, "G")
	}
	var n int64
	fmt.Sscanf(v, "%d", &n)
	return n * mult
}

// fmtLimit shows a limit value for humans.
func fmtLimit(v string) string {
	n := parseLimit(v)
	if n == 0 {
		return "∞"
	}
	return humanSpeed(n)
}
