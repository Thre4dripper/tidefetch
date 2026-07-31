package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:             "0 B",
		999:           "999 B",
		1024:          "1.0 KB",
		1536:          "1.5 KB",
		1048576:       "1.0 MB",
		210 * 1 << 20: "210 MB",
		3 * 1 << 30:   "3.0 GB",
		1 << 40:       "1.0 TB",
	}
	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestHumanETA(t *testing.T) {
	if got := humanETA(1000, 0); got != "--" {
		t.Errorf("zero speed: %q", got)
	}
	if got := humanETA(100, 100); got != "1s" {
		t.Errorf("1s: %q", got)
	}
	if got := humanETA(90*100, 100); got != "1m30s" {
		t.Errorf("90s: %q", got)
	}
	if got := humanETA(3660*100, 100); got != "1h01m" {
		t.Errorf("1h1m: %q", got)
	}
}

func TestGaugeWidth(t *testing.T) {
	for _, frac := range []float64{-0.5, 0, 0.33, 0.5, 0.999, 1, 2} {
		for _, w := range []int{1, 10, 40} {
			g := gauge(frac, w)
			if lw := lipgloss.Width(g); lw != w {
				t.Errorf("gauge(%v,%d) width = %d", frac, w, lw)
			}
		}
	}
}

func TestSparkline(t *testing.T) {
	s := sparkline([]float64{0, 1, 2, 3, 4, 5, 6, 7}, 8)
	if got := []rune(s); len(got) != 8 {
		t.Errorf("len = %d", len(got))
	}
	// The scale carries headroom so a steady transfer never renders as a
	// solid filled bar, which reads as a progress indicator. The peak should
	// therefore sit high but stop short of the full block.
	runes := []rune(s)
	peak := runes[len(runes)-1]
	if peak == '█' {
		t.Errorf("peak should leave headroom, got full block: %q", s)
	}
	if peak != '▆' && peak != '▇' {
		t.Errorf("peak should be near the top, got %q in %q", peak, s)
	}
	// Values must increase monotonically with the samples.
	for i := 2; i < len(runes); i++ {
		if runes[i] < runes[i-1] {
			t.Errorf("not monotonic at %d: %q", i, s)
		}
	}
	// window smaller than samples
	if got := sparkline([]float64{1, 2, 3, 4}, 2); len([]rune(got)) != 2 {
		t.Errorf("window: %q", got)
	}
	// empty
	if got := sparkline(nil, 5); len([]rune(got)) != 5 {
		t.Errorf("empty: %q", got)
	}
}

func TestTruncatePad(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("no-op: %q", got)
	}
	if got := truncate("hello world", 6); lipgloss.Width(got) > 6 {
		t.Errorf("too wide: %q", got)
	}
	if got := padRight("ab", 5); got != "ab   " {
		t.Errorf("padRight: %q", got)
	}
	if got := padLeft("ab", 5); got != "   ab" {
		t.Errorf("padLeft: %q", got)
	}
}

func TestLimits(t *testing.T) {
	if v := parseLimit("500K"); v != 512000 {
		t.Errorf("500K = %d", v)
	}
	if v := parseLimit("2M"); v != 2097152 {
		t.Errorf("2M = %d", v)
	}
	if v := parseLimit("0"); v != 0 {
		t.Errorf("0 = %d", v)
	}
	if got := stepLimit("0", true); got != "256K" {
		t.Errorf("step up from 0: %q", got)
	}
	if got := stepLimit("256K", false); got != "0" {
		t.Errorf("step down to 0: %q", got)
	}
	if got := fmtLimit("0"); got != "∞" {
		t.Errorf("unlimited: %q", got)
	}
}

func TestPiecesBar(t *testing.T) {
	// 8 pieces, all set: 0xFF
	bar := piecesBar("ff", 8, 4)
	if lipgloss.Width(bar) != 4 {
		t.Errorf("width: %q", bar)
	}
	if piecesBar("", 0, 10) != "" {
		t.Error("empty bitfield should render empty")
	}
}
