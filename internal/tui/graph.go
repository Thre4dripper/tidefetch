package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// downsampleHistory bucket-averages an entire series into at most n points.
// Unlike simply taking the tail, this preserves the shape of a task's full
// lifetime after it completes.
func downsampleHistory(samples []float64, n int) []float64 {
	if n < 1 || len(samples) == 0 {
		return nil
	}
	if len(samples) <= n {
		out := make([]float64, len(samples))
		copy(out, samples)
		return out
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		lo := i * len(samples) / n
		hi := (i + 1) * len(samples) / n
		if hi <= lo {
			hi = lo + 1
		}
		for _, value := range samples[lo:hi] {
			out[i] += value
		}
		out[i] /= float64(hi - lo)
	}
	return out
}

// brailleGraph renders samples as a btop-style braille chart of w×h cells.
// Each cell packs 2 columns × 4 rows of dots. The most recent sample is at
// the right edge. maxVal <= 0 auto-scales to the window maximum.
func brailleGraph(samples []float64, w, h int, maxVal float64) []string {
	if w < 1 || h < 1 {
		return nil
	}
	cols := w * 2
	if len(samples) > cols {
		samples = samples[len(samples)-cols:]
	}
	if maxVal <= 0 {
		for _, v := range samples {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	rows := h * 4
	// heights[c] = number of dots lit in column c (right aligned).
	heights := make([]int, cols)
	off := cols - len(samples)
	for i, v := range samples {
		if maxVal <= 0 || v <= 0 {
			heights[off+i] = 0
			continue
		}
		d := int(v/maxVal*float64(rows) + 0.5)
		if d > rows {
			d = rows
		}
		if d == 0 && v > 0 {
			d = 1
		}
		heights[off+i] = d
	}

	// Braille dot bit for (x 0..1, y 0..3), y=0 is the TOP dot of the cell.
	dotBits := [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

	out := make([]string, h)
	for cy := 0; cy < h; cy++ {
		var line strings.Builder
		for cx := 0; cx < w; cx++ {
			bits := 0
			for dx := 0; dx < 2; dx++ {
				colH := heights[cx*2+dx]
				for dy := 0; dy < 4; dy++ {
					// absolute dot row from top: cy*4+dy; from bottom:
					fromBottom := rows - (cy*4 + dy)
					if fromBottom <= colH {
						bits |= dotBits[dy][dx]
					}
				}
			}
			line.WriteRune(rune(0x2800 + bits))
		}
		out[cy] = line.String()
	}
	return out
}

// titledBox draws a btop-style rounded box with the title embedded in the
// top border: ╭─ Title ─────╮. Content lines are clipped/padded to fit.
// w is the total outer width; the content area is w-4 wide (border + pad).
func titledBox(title string, lines []string, w int, borderStyle, titleStyle lipgloss.Style) string {
	if w < 8 {
		w = 8
	}
	inner := w - 4
	var b strings.Builder

	// top border with title
	t := " " + title + " "
	tw := lipgloss.Width(t)
	if tw > w-4 {
		t = truncate(t, w-4)
		tw = lipgloss.Width(t)
	}
	b.WriteString(borderStyle.Render("╭─"))
	b.WriteString(titleStyle.Render(t))
	b.WriteString(borderStyle.Render(strings.Repeat("─", maxInt(0, w-3-tw)) + "╮"))
	b.WriteString("\n")

	for _, ln := range lines {
		b.WriteString(borderStyle.Render("│ "))
		b.WriteString(padRight(truncate(ln, inner), inner))
		b.WriteString(borderStyle.Render(" │"))
		b.WriteString("\n")
	}

	b.WriteString(borderStyle.Render("╰" + strings.Repeat("─", w-2) + "╯"))
	return b.String()
}

// usageBar renders a used/total meter like ███████░░░░ with color by pressure.
func usageBar(used, total int64, width int) string {
	if width < 1 || total <= 0 {
		return ""
	}
	frac := float64(used) / float64(total)
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	on := int(frac*float64(width) + 0.5)
	style := styleGood
	if frac > 0.75 {
		style = styleWarn
	}
	if frac > 0.9 {
		style = styleBad
	}
	return style.Render(strings.Repeat("█", on)) + styleFaint.Render(strings.Repeat("░", width-on))
}
