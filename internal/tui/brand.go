package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// tideRunes is one full period of a traveling swell. Sliding a window across
// it animates the same wave the SVG logo uses on the web.
var tideRunes = []rune("▁▂▃▄▅▆▇▆▅▄▃▂")

// tidePhase advances roughly every 130ms so the wave drifts, never strobes.
func tidePhase(since time.Duration) int {
	return int(since/(130*time.Millisecond)) % len(tideRunes)
}

// tideGlyphs renders n cells of the animated tide at the given phase.
func tideGlyphs(phase, n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteRune(tideRunes[(phase+i)%len(tideRunes)])
	}
	return b.String()
}

// brandLockup renders the tidefetch wordmark: animated tide glyphs, then
// "tide" in cyan and "Fetch" in bright — matching the web/site lockup.
func brandLockup(since time.Duration) string {
	wave := styleTideWave.Render(tideGlyphs(tidePhase(since), 3))
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		wave,
		" ",
		styleTideName.Render("tide"),
		styleFetchName.Render("Fetch"),
	)
}

// brandStatic renders the lockup without animation, for static contexts.
func brandStatic() string {
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		styleTideWave.Render(tideGlyphs(0, 3)),
		" ",
		styleTideName.Render("tide"),
		styleFetchName.Render("Fetch"),
	)
}
