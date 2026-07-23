package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestBrailleGraph(t *testing.T) {
	samples := []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	g := brailleGraph(samples, 10, 4, 0)
	if len(g) != 4 {
		t.Fatalf("rows = %d", len(g))
	}
	for _, line := range g {
		if n := len([]rune(line)); n != 10 {
			t.Errorf("line width = %d", n)
		}
	}
	// bottom row should have more dots lit than the top row
	if g[0] == g[3] {
		t.Errorf("graph should vary vertically")
	}
	// empty input still renders the full grid
	e := brailleGraph(nil, 5, 2, 0)
	if len(e) != 2 || len([]rune(e[0])) != 5 {
		t.Errorf("empty graph dims wrong: %v", e)
	}
	if brailleGraph(samples, 0, 0, 0) != nil {
		t.Error("zero size should be nil")
	}
}

func TestTitledBox(t *testing.T) {
	box := titledBox("Test", []string{"hello", "world"}, 20, styleBoxBorder, styleBoxTitle)
	lines := strings.Split(box, "\n")
	if len(lines) != 4 {
		t.Fatalf("lines = %d", len(lines))
	}
	for i, ln := range lines {
		if w := lipgloss.Width(ln); w != 20 {
			t.Errorf("line %d width = %d", i, w)
		}
	}
	if !strings.Contains(box, "Test") {
		t.Error("title missing")
	}
}

func TestUsageBar(t *testing.T) {
	if got := lipgloss.Width(usageBar(50, 100, 10)); got != 10 {
		t.Errorf("width = %d", got)
	}
	if usageBar(1, 0, 10) != "" {
		t.Error("zero total should be empty")
	}
}

func TestHostOf(t *testing.T) {
	cases := map[string]string{
		"https://dl.example.com/path/file.iso?x=1": "dl.example.com",
		"ftp://user:pass@mirror.net/file":          "mirror.net",
		"magnet:?xt=urn:btih:abc":                  "magnet",
		"":                                         "local",
	}
	for in, want := range cases {
		if got := hostOf(in); got != want {
			t.Errorf("hostOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestShortPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := shortPath(home + "/Downloads"); got != "~/Downloads" {
		t.Errorf("shortPath home = %q", got)
	}
	if got := shortPath("/tmp/x"); got != "/tmp/x" {
		t.Errorf("shortPath other = %q", got)
	}
}

func TestHitboxes(t *testing.T) {
	a := &App{}
	a.addHit(0, 1, 10, 1, "tab:0")
	a.addHit(0, 2, 80, 3, "row:0")
	a.addHit(0, 5, 80, 3, "row:1")
	if got := a.hitAt(5, 1); got != "tab:0" {
		t.Errorf("tab hit = %q", got)
	}
	if got := a.hitAt(40, 4); got != "row:0" {
		t.Errorf("row0 hit = %q", got)
	}
	if got := a.hitAt(0, 7); got != "row:1" {
		t.Errorf("row1 hit = %q", got)
	}
	if got := a.hitAt(90, 1); got != "" {
		t.Errorf("miss = %q", got)
	}
}

func TestKeyFromString(t *testing.T) {
	if keyFromString("a").String() != "a" {
		t.Error("rune key")
	}
	if keyFromString("space").String() != " " {
		t.Error("space key")
	}
	if keyFromString("enter").String() != "enter" {
		t.Error("enter key")
	}
	if keyFromString("ctrl+s").String() != "ctrl+s" {
		t.Error("ctrl+s key")
	}
}

func TestDiskUsage(t *testing.T) {
	free, total, err := diskUsage("/")
	if err != nil {
		t.Skipf("diskUsage unsupported: %v", err)
	}
	if total <= 0 || free < 0 || free > total {
		t.Errorf("free=%d total=%d", free, total)
	}
}
