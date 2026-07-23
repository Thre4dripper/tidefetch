package ui

import "github.com/charmbracelet/lipgloss"

// Surge-inspired dark palette.
var (
	cAccent   = lipgloss.Color("#7C6CF0") // purple
	cAccent2  = lipgloss.Color("#5A9CF8") // blue
	cGreen    = lipgloss.Color("#3DDC97")
	cYellow   = lipgloss.Color("#F5C744")
	cRed      = lipgloss.Color("#F0637C")
	cCyan     = lipgloss.Color("#43C6D8")
	cText     = lipgloss.Color("#C8CCE8")
	cBright   = lipgloss.Color("#EDEFFB")
	cDim      = lipgloss.Color("#62678C")
	cFaint    = lipgloss.Color("#3D4163")
	cSurface  = lipgloss.Color("#191B28")
	cSurface2 = lipgloss.Color("#232638")
	cSelBG    = lipgloss.Color("#2A2D45")
)

var (
	styleLogo = lipgloss.NewStyle().Foreground(cBright).Background(cAccent).Bold(true).Padding(0, 1)

	styleTitle    = lipgloss.NewStyle().Foreground(cBright).Bold(true)
	styleText     = lipgloss.NewStyle().Foreground(cText)
	styleDim      = lipgloss.NewStyle().Foreground(cDim)
	styleFaint    = lipgloss.NewStyle().Foreground(cFaint)
	styleAccent   = lipgloss.NewStyle().Foreground(cAccent)
	styleAccent2  = lipgloss.NewStyle().Foreground(cAccent2)
	styleGood     = lipgloss.NewStyle().Foreground(cGreen)
	styleWarn     = lipgloss.NewStyle().Foreground(cYellow)
	styleBad      = lipgloss.NewStyle().Foreground(cRed)
	styleCyan     = lipgloss.NewStyle().Foreground(cCyan)
	styleDownArr  = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	styleUpArr    = lipgloss.NewStyle().Foreground(cAccent2).Bold(true)
	styleSpark    = lipgloss.NewStyle().Foreground(cAccent)
	styleSelBar   = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	styleRowSel   = lipgloss.NewStyle().Background(cSelBG)
	styleGaugeOn  = lipgloss.NewStyle().Foreground(cAccent)
	styleGaugeOff = lipgloss.NewStyle().Foreground(cFaint)

	styleTabActive = lipgloss.NewStyle().Foreground(cBright).Background(cSurface2).Bold(true).Padding(0, 2)
	styleTab       = lipgloss.NewStyle().Foreground(cDim).Padding(0, 2)

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

	styleKey  = lipgloss.NewStyle().Foreground(cAccent2).Bold(true)
	styleDesc = lipgloss.NewStyle().Foreground(cDim)

	styleToastInfo = lipgloss.NewStyle().Foreground(cBright).Background(cAccent).Padding(0, 1).Bold(true)
	styleToastGood = lipgloss.NewStyle().Foreground(lipgloss.Color("#0B2E20")).Background(cGreen).Padding(0, 1).Bold(true)
	styleToastBad  = lipgloss.NewStyle().Foreground(cBright).Background(cRed).Padding(0, 1).Bold(true)

	styleBadge = lipgloss.NewStyle().Foreground(cBright).Background(cSurface2).Padding(0, 1)

	styleInputLabel = lipgloss.NewStyle().Foreground(cDim).Bold(true)
	styleFieldFocus = lipgloss.NewStyle().Foreground(cAccent).Bold(true)

	// btop-style titled blocks
	styleBoxBorder   = lipgloss.NewStyle().Foreground(cFaint)
	styleBoxBorderHi = lipgloss.NewStyle().Foreground(cAccent)
	styleBoxTitle    = lipgloss.NewStyle().Foreground(cBright).Bold(true)
	styleGraphDown   = lipgloss.NewStyle().Foreground(cAccent)
	styleGraphUp     = lipgloss.NewStyle().Foreground(cAccent2)
	styleGraphTask   = lipgloss.NewStyle().Foreground(cGreen)

	// clickable footer buttons
	styleBtn    = lipgloss.NewStyle().Foreground(cText).Background(cSurface2).Padding(0, 1)
	styleBtnKey = lipgloss.NewStyle().Foreground(cAccent2).Background(cSurface2).Bold(true).Padding(0, 0, 0, 1)
)

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
