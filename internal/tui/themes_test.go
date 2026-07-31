package tui

import (
	"regexp"
	"testing"
)

var hexColor = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func TestThemesAreWellFormed(t *testing.T) {
	if len(themes) < 10 {
		t.Fatalf("expected at least 10 themes, got %d", len(themes))
	}

	names := map[string]bool{}
	labels := map[string]bool{}
	for _, th := range themes {
		if th.Name == "" || th.Label == "" {
			t.Errorf("theme %+v missing name or label", th)
		}
		if names[th.Name] {
			t.Errorf("duplicate theme name %q", th.Name)
		}
		if labels[th.Label] {
			t.Errorf("duplicate theme label %q", th.Label)
		}
		names[th.Name] = true
		labels[th.Label] = true

		swatches := map[string]string{
			"Ink": string(th.Ink), "Accent": string(th.Accent), "Accent2": string(th.Accent2),
			"Green": string(th.Green), "Yellow": string(th.Yellow), "Red": string(th.Red),
			"Cyan": string(th.Cyan), "Text": string(th.Text), "Bright": string(th.Bright),
			"Dim": string(th.Dim), "Faint": string(th.Faint), "Surface": string(th.Surface),
			"Surface2": string(th.Surface2), "SelBG": string(th.SelBG),
		}
		for field, value := range swatches {
			if !hexColor.MatchString(value) {
				t.Errorf("theme %q field %s = %q, want #RRGGBB", th.Name, field, value)
			}
		}
	}

	if !names[defaultTheme] {
		t.Errorf("defaultTheme %q is not in the theme list", defaultTheme)
	}
}

func TestLookupThemeFallsBackToDefault(t *testing.T) {
	if got := lookupTheme("does-not-exist").Name; got != defaultTheme {
		t.Errorf("lookupTheme(unknown) = %q, want %q", got, defaultTheme)
	}
	if got := lookupTheme("nord").Name; got != "nord" {
		t.Errorf("lookupTheme(nord) = %q", got)
	}
}

func TestThemeLabelRoundTrip(t *testing.T) {
	for _, name := range themeNames() {
		if got := themeByLabel(themeLabel(name)); got != name {
			t.Errorf("round trip %q -> %q -> %q", name, themeLabel(name), got)
		}
	}
	if got := themeByLabel("Nothing Like This"); got != defaultTheme {
		t.Errorf("themeByLabel(unknown) = %q, want %q", got, defaultTheme)
	}
}

func TestApplyThemeRestylesUI(t *testing.T) {
	t.Cleanup(func() { applyTheme(defaultTheme) })

	applyTheme("gruvbox")
	if string(cAccent) != "#FE8019" {
		t.Errorf("gruvbox accent = %q", cAccent)
	}
	gruvboxAccent := styleAccent.GetForeground()

	applyTheme("nord")
	if string(cAccent) != "#88C0D0" {
		t.Errorf("nord accent = %q", cAccent)
	}
	if styleAccent.GetForeground() == gruvboxAccent {
		t.Error("styles were not rebuilt when the theme changed")
	}

	// Derived styles must track the palette, not just the raw colours.
	if styleGraphDown.GetForeground() != styleAccent.GetForeground() {
		t.Error("graph style did not follow the active accent")
	}
	if activeTheme.Name != "nord" {
		t.Errorf("activeTheme = %q, want nord", activeTheme.Name)
	}
}

func TestApplyThemeUnknownKeepsUsable(t *testing.T) {
	t.Cleanup(func() { applyTheme(defaultTheme) })

	applyTheme("not-a-theme")
	if activeTheme.Name != defaultTheme {
		t.Errorf("activeTheme = %q, want %q", activeTheme.Name, defaultTheme)
	}
	if string(cAccent) == "" || string(cText) == "" {
		t.Error("palette left unpopulated after unknown theme")
	}
}
