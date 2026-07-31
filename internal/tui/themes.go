package tui

import "github.com/charmbracelet/lipgloss"

// Theme is a complete terminal colour palette. Every style in the UI is
// derived from these values, so switching a theme restyles the whole app.
type Theme struct {
	Name  string
	Label string

	// Ink is a near-black used as foreground on bright accent backgrounds.
	Ink lipgloss.Color

	Accent   lipgloss.Color // primary: selection, focus, download graphs
	Accent2  lipgloss.Color // secondary: uploads, keybind hints
	Green    lipgloss.Color // success / completed
	Yellow   lipgloss.Color // warnings / paused
	Red      lipgloss.Color // errors
	Cyan     lipgloss.Color // seeding
	Text     lipgloss.Color
	Bright   lipgloss.Color
	Dim      lipgloss.Color
	Faint    lipgloss.Color
	Surface  lipgloss.Color
	Surface2 lipgloss.Color
	SelBG    lipgloss.Color
}

// themes are ordered as they appear in Settings. Surge is the default.
var themes = []Theme{
	{
		Name: "surge", Label: "Surge",
		Ink:    "#0C0A1A",
		Accent: "#7C6CF0", Accent2: "#5A9CF8",
		Green: "#3DDC97", Yellow: "#F5C744", Red: "#F0637C", Cyan: "#43C6D8",
		Text: "#C8CCE8", Bright: "#EDEFFB", Dim: "#7B80A8", Faint: "#3D4163",
		Surface: "#191B28", Surface2: "#232638", SelBG: "#2A2D45",
	},
	{
		Name: "tide", Label: "Tide",
		Ink:    "#0C0E11",
		Accent: "#5ED8E7", Accent2: "#8C82FF",
		Green: "#B8FF3D", Yellow: "#F0C445", Red: "#FF795F", Cyan: "#5ED8E7",
		Text: "#CDD5CF", Bright: "#EEF3EF", Dim: "#8B938E", Faint: "#4B5450",
		Surface: "#14181C", Surface2: "#1E2429", SelBG: "#1C2227",
	},
	{
		Name: "tokyonight", Label: "Tokyo Night",
		Ink:    "#16161E",
		Accent: "#7AA2F7", Accent2: "#BB9AF7",
		Green: "#9ECE6A", Yellow: "#E0AF68", Red: "#F7768E", Cyan: "#7DCFFF",
		Text: "#C0CAF5", Bright: "#D5DBF5", Dim: "#787C99", Faint: "#3B4261",
		Surface: "#1A1B26", Surface2: "#24283B", SelBG: "#292E42",
	},
	{
		Name: "catppuccin", Label: "Catppuccin Mocha",
		Ink:    "#11111B",
		Accent: "#CBA6F7", Accent2: "#89B4FA",
		Green: "#A6E3A1", Yellow: "#F9E2AF", Red: "#F38BA8", Cyan: "#89DCEB",
		Text: "#CDD6F4", Bright: "#E6E9F4", Dim: "#9399B2", Faint: "#45475A",
		Surface: "#1E1E2E", Surface2: "#313244", SelBG: "#313244",
	},
	{
		Name: "gruvbox", Label: "Gruvbox Dark",
		Ink:    "#1D2021",
		Accent: "#FE8019", Accent2: "#83A598",
		Green: "#B8BB26", Yellow: "#FABD2F", Red: "#FB4934", Cyan: "#8EC07C",
		Text: "#EBDBB2", Bright: "#FBF1C7", Dim: "#A89984", Faint: "#504945",
		Surface: "#282828", Surface2: "#3C3836", SelBG: "#3C3836",
	},
	{
		Name: "nord", Label: "Nord",
		Ink:    "#2E3440",
		Accent: "#88C0D0", Accent2: "#B48EAD",
		Green: "#A3BE8C", Yellow: "#EBCB8B", Red: "#BF616A", Cyan: "#8FBCBB",
		Text: "#D8DEE9", Bright: "#ECEFF4", Dim: "#8891A5", Faint: "#4C566A",
		Surface: "#2E3440", Surface2: "#3B4252", SelBG: "#434C5E",
	},
	{
		Name: "dracula", Label: "Dracula",
		Ink:    "#21222C",
		Accent: "#BD93F9", Accent2: "#FF79C6",
		Green: "#50FA7B", Yellow: "#F1FA8C", Red: "#FF5555", Cyan: "#8BE9FD",
		Text: "#F8F8F2", Bright: "#FFFFFF", Dim: "#8A93BC", Faint: "#44475A",
		Surface: "#282A36", Surface2: "#343746", SelBG: "#44475A",
	},
	{
		Name: "rosepine", Label: "Rosé Pine",
		Ink:    "#191724",
		Accent: "#C4A7E7", Accent2: "#EBBCBA",
		Green: "#9CCFD8", Yellow: "#F6C177", Red: "#EB6F92", Cyan: "#9CCFD8",
		Text: "#E0DEF4", Bright: "#F2F0FA", Dim: "#908CAA", Faint: "#6E6A86",
		Surface: "#1F1D2E", Surface2: "#26233A", SelBG: "#26233A",
	},
	{
		Name: "everforest", Label: "Everforest",
		Ink:    "#232A2E",
		Accent: "#83C092", Accent2: "#7FBBB3",
		Green: "#A7C080", Yellow: "#DBBC7F", Red: "#E67E80", Cyan: "#7FBBB3",
		Text: "#D3C6AA", Bright: "#E8DFC8", Dim: "#9DA9A0", Faint: "#4F585E",
		Surface: "#2D353B", Surface2: "#343F44", SelBG: "#3D484D",
	},
	{
		Name: "kanagawa", Label: "Kanagawa",
		Ink:    "#16161D",
		Accent: "#7E9CD8", Accent2: "#957FB8",
		Green: "#98BB6C", Yellow: "#E6C384", Red: "#E46876", Cyan: "#7AA89F",
		Text: "#DCD7BA", Bright: "#F2ECE0", Dim: "#9A9A8F", Faint: "#54546D",
		Surface: "#1F1F28", Surface2: "#2A2A37", SelBG: "#363646",
	},
	{
		Name: "solarized", Label: "Solarized Dark",
		Ink:    "#002B36",
		Accent: "#268BD2", Accent2: "#6C71C4",
		Green: "#859900", Yellow: "#B58900", Red: "#DC322F", Cyan: "#2AA198",
		Text: "#93A1A1", Bright: "#EEE8D5", Dim: "#7A8C8C", Faint: "#586E75",
		Surface: "#002B36", Surface2: "#073642", SelBG: "#073642",
	},
	{
		Name: "ayu", Label: "Ayu Dark",
		Ink:    "#0B0E14",
		Accent: "#59C2FF", Accent2: "#D2A6FF",
		Green: "#AAD94C", Yellow: "#E6B450", Red: "#F07178", Cyan: "#95E6CB",
		Text: "#BFBDB6", Bright: "#E6E1CF", Dim: "#8A8F99", Faint: "#3D424D",
		Surface: "#0F131A", Surface2: "#1A1F29", SelBG: "#1A1F29",
	},
	{
		Name: "monokai", Label: "Monokai Pro",
		Ink:    "#221F22",
		Accent: "#AB9DF2", Accent2: "#78DCE8",
		Green: "#A9DC76", Yellow: "#FFD866", Red: "#FF6188", Cyan: "#78DCE8",
		Text: "#FCFCFA", Bright: "#FFFFFF", Dim: "#939293", Faint: "#5B595C",
		Surface: "#2D2A2E", Surface2: "#403E41", SelBG: "#403E41",
	},
}

// defaultTheme is used when the config has no theme or names an unknown one.
const defaultTheme = "surge"

// themeNames returns the selectable theme identifiers, in display order.
func themeNames() []string {
	out := make([]string, len(themes))
	for i, t := range themes {
		out[i] = t.Name
	}
	return out
}

// themeLabels returns the human-readable theme names, in display order.
func themeLabels() []string {
	out := make([]string, len(themes))
	for i, t := range themes {
		out[i] = t.Label
	}
	return out
}

// themeByLabel resolves a display label back to its identifier.
func themeByLabel(label string) string {
	for _, t := range themes {
		if t.Label == label {
			return t.Name
		}
	}
	return defaultTheme
}

// themeLabel maps an identifier to its human-readable name.
func themeLabel(name string) string {
	for _, t := range themes {
		if t.Name == name {
			return t.Label
		}
	}
	return themeLabel(defaultTheme)
}

// lookupTheme resolves a theme by name, falling back to the default.
func lookupTheme(name string) Theme {
	for _, t := range themes {
		if t.Name == name {
			return t
		}
	}
	for _, t := range themes {
		if t.Name == defaultTheme {
			return t
		}
	}
	return themes[0]
}
