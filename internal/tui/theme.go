package tui

import "github.com/charmbracelet/lipgloss"

// Active palette. These are populated by applyTheme and re-read on every
// render, so switching themes at runtime restyles the entire UI.
var (
	cAccent   lipgloss.Color
	cAccent2  lipgloss.Color
	cGreen    lipgloss.Color
	cYellow   lipgloss.Color
	cRed      lipgloss.Color
	cCyan     lipgloss.Color
	cText     lipgloss.Color
	cBright   lipgloss.Color
	cDim      lipgloss.Color
	cFaint    lipgloss.Color
	cSurface  lipgloss.Color
	cSurface2 lipgloss.Color
	cSelBG    lipgloss.Color
	cInk      lipgloss.Color
)

// Styles derived from the active palette. Rebuilt by applyTheme.
var (
	// Brand lockup: animated tide glyphs + "tide" + "Fetch".
	styleTideWave  lipgloss.Style
	styleTideName  lipgloss.Style
	styleFetchName lipgloss.Style
	styleLogo      lipgloss.Style

	styleTitle    lipgloss.Style
	styleText     lipgloss.Style
	styleDim      lipgloss.Style
	styleFaint    lipgloss.Style
	styleAccent   lipgloss.Style
	styleAccent2  lipgloss.Style
	styleGood     lipgloss.Style
	styleWarn     lipgloss.Style
	styleBad      lipgloss.Style
	styleCyan     lipgloss.Style
	styleDownArr  lipgloss.Style
	styleUpArr    lipgloss.Style
	styleSpark    lipgloss.Style
	styleSelBar   lipgloss.Style
	styleRowSel   lipgloss.Style
	styleGaugeOn  lipgloss.Style
	styleGaugeOff lipgloss.Style

	styleTabActive lipgloss.Style
	styleTab       lipgloss.Style

	styleHeaderBox  lipgloss.Style
	stylePanel      lipgloss.Style
	stylePanelFocus lipgloss.Style
	styleModal      lipgloss.Style

	styleKey  lipgloss.Style
	styleDesc lipgloss.Style

	styleToastInfo lipgloss.Style
	styleToastGood lipgloss.Style
	styleToastBad  lipgloss.Style

	styleBadge lipgloss.Style

	styleInputLabel lipgloss.Style
	styleFieldFocus lipgloss.Style

	// btop-style titled blocks
	styleBoxBorder   lipgloss.Style
	styleBoxBorderHi lipgloss.Style
	styleBoxTitle    lipgloss.Style
	styleGraphDown   lipgloss.Style
	styleGraphUp     lipgloss.Style
	styleGraphTask   lipgloss.Style

	// clickable footer buttons
	styleBtn    lipgloss.Style
	styleBtnKey lipgloss.Style
)

// activeTheme is the palette currently applied.
var activeTheme Theme

func init() { applyTheme(defaultTheme) }

// applyTheme switches the palette and rebuilds every derived style.
// Unknown names fall back to the default theme.
func applyTheme(name string) {
	t := lookupTheme(name)
	activeTheme = t

	cAccent, cAccent2 = t.Accent, t.Accent2
	cGreen, cYellow, cRed, cCyan = t.Green, t.Yellow, t.Red, t.Cyan
	cText, cBright, cDim, cFaint = t.Text, t.Bright, t.Dim, t.Faint
	cSurface, cSurface2, cSelBG = t.Surface, t.Surface2, t.SelBG
	cInk = t.Ink

	styleTideWave = lipgloss.NewStyle().Foreground(cAccent)
	styleTideName = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	styleFetchName = lipgloss.NewStyle().Foreground(cBright)
	styleLogo = lipgloss.NewStyle().Foreground(cBright).Bold(true)

	styleTitle = lipgloss.NewStyle().Foreground(cBright).Bold(true)
	styleText = lipgloss.NewStyle().Foreground(cText)
	styleDim = lipgloss.NewStyle().Foreground(cDim)
	styleFaint = lipgloss.NewStyle().Foreground(cFaint)
	styleAccent = lipgloss.NewStyle().Foreground(cAccent)
	styleAccent2 = lipgloss.NewStyle().Foreground(cAccent2)
	styleGood = lipgloss.NewStyle().Foreground(cGreen)
	styleWarn = lipgloss.NewStyle().Foreground(cYellow)
	styleBad = lipgloss.NewStyle().Foreground(cRed)
	styleCyan = lipgloss.NewStyle().Foreground(cCyan)
	styleDownArr = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	styleUpArr = lipgloss.NewStyle().Foreground(cAccent2).Bold(true)
	styleSpark = lipgloss.NewStyle().Foreground(cAccent)
	styleSelBar = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	styleRowSel = lipgloss.NewStyle().Background(cSelBG)
	styleGaugeOn = lipgloss.NewStyle().Foreground(cAccent)
	styleGaugeOff = lipgloss.NewStyle().Foreground(cFaint)

	styleTabActive = lipgloss.NewStyle().Foreground(cBright).Background(cSurface2).Bold(true).Padding(0, 2)
	styleTab = lipgloss.NewStyle().Foreground(cDim).Padding(0, 2)

	styleHeaderBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(cFaint).
		Padding(0, 1)

	stylePanel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(cFaint).
		Padding(0, 1)

	stylePanelFocus = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(cAccent).
		Padding(0, 1)

	styleModal = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).BorderForeground(cAccent).
		Background(cSurface).Padding(1, 3)

	styleKey = lipgloss.NewStyle().Foreground(cAccent2).Bold(true)
	styleDesc = lipgloss.NewStyle().Foreground(cDim)

	// Toasts sit on saturated backgrounds, so they use the theme's ink colour
	// for contrast rather than the normal foreground.
	styleToastInfo = lipgloss.NewStyle().Foreground(cInk).Background(cAccent).Padding(0, 1).Bold(true)
	styleToastGood = lipgloss.NewStyle().Foreground(cInk).Background(cGreen).Padding(0, 1).Bold(true)
	styleToastBad = lipgloss.NewStyle().Foreground(cBright).Background(cRed).Padding(0, 1).Bold(true)

	styleBadge = lipgloss.NewStyle().Foreground(cBright).Background(cSurface2).Padding(0, 1)

	styleInputLabel = lipgloss.NewStyle().Foreground(cDim).Bold(true)
	styleFieldFocus = lipgloss.NewStyle().Foreground(cAccent).Bold(true)

	styleBoxBorder = lipgloss.NewStyle().Foreground(cFaint)
	styleBoxBorderHi = lipgloss.NewStyle().Foreground(cAccent)
	styleBoxTitle = lipgloss.NewStyle().Foreground(cBright).Bold(true)
	styleGraphDown = lipgloss.NewStyle().Foreground(cAccent)
	styleGraphUp = lipgloss.NewStyle().Foreground(cAccent2)
	styleGraphTask = lipgloss.NewStyle().Foreground(cGreen)

	styleBtn = lipgloss.NewStyle().Foreground(cText).Background(cSurface2).Padding(0, 2, 0, 1)
	styleBtnKey = lipgloss.NewStyle().Foreground(cAccent2).Background(cSurface2).Bold(true).Padding(0, 0, 0, 2)
}

// statusStyle picks color + icon for an aria2 download status.
func statusStyle(status string, seeding bool) (lipgloss.Style, string, string) {
	switch status {
	case "active":
		if seeding {
			return styleCyan, "⇡", "SEED"
		}
		return styleGood, "⇣", "ACTIVE"
	case "waiting":
		return styleAccent2, "◌", "QUEUED"
	case "paused":
		return styleWarn, "⏸", "PAUSED"
	case "complete":
		return styleGood, "✓", "DONE"
	case "error":
		return styleBad, "✗", "ERROR"
	case "removed":
		return styleDim, "–", "REMOVED"
	}
	return styleDim, "•", status
}
