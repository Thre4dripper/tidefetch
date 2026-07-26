package tui

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/turbostart/tidefetch/internal/config"
	"github.com/turbostart/tidefetch/internal/history"
	"github.com/turbostart/tidefetch/pkg/aria2"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestAddBuildOptionsStructuredHeaders(t *testing.T) {
	m := newAddModel(config.Default())
	m.showAdvanced = true
	m.fieldByKey(headerCountKey).value = "3"
	m.fieldByKey(headerNameKey(1)).value = "Authorization"
	m.fieldByKey(headerValueKey(1)).value = "Bearer token"
	m.fieldByKey(headerNameKey(2)).value = "Custom…"
	m.fieldByKey(headerCustomNameKey(2)).value = "X-Trace"
	m.fieldByKey(headerValueKey(2)).value = "abc123"
	m.fieldByKey(headerNameKey(3)).value = "Cookie"
	m.fieldByKey(headerValueKey(3)).value = "" // incomplete rows are ignored
	m.fieldByKey(aria2.OptMaxUploadLimit).value = "1M"

	opts := m.buildOptions()
	if got, want := opts[aria2.OptHeader], "Authorization: Bearer token\nX-Trace: abc123"; got != want {
		t.Fatalf("header = %q, want %q", got, want)
	}
	if got := opts[aria2.OptMaxUploadLimit]; got != "1M" {
		t.Fatalf("upload limit = %q, want 1M", got)
	}
	if _, ok := opts[headerCountKey]; ok {
		t.Fatal("internal header count leaked into aria2 options")
	}
}

func TestAddVisibleHeaderRowsFollowCount(t *testing.T) {
	m := newAddModel(config.Default())
	m.showAdvanced = true
	m.fieldByKey(headerCountKey).value = "2"

	visible := map[string]bool{}
	for _, index := range m.visibleFields() {
		visible[m.fields[index].key] = true
	}
	if !visible[headerNameKey(1)] || !visible[headerValueKey(2)] {
		t.Fatal("configured header rows are not visible")
	}
	if visible[headerNameKey(3)] || visible[headerValueKey(3)] {
		t.Fatal("header rows above the configured count are visible")
	}
	if visible[headerCustomNameKey(1)] || visible[headerCustomNameKey(2)] {
		t.Fatal("custom-name fields are visible for preset headers")
	}
	m.fieldByKey(headerNameKey(2)).value = "Custom…"
	visible = map[string]bool{}
	for _, index := range m.visibleFields() {
		visible[m.fields[index].key] = true
	}
	if !visible[headerCustomNameKey(2)] {
		t.Fatal("custom-name field is hidden for a custom header")
	}
}

func TestAdvancedSettingsFilter(t *testing.T) {
	m := newSettingsModel()
	m.opts = aria2.Options{
		"max-overall-download-limit": "0",
		"max-overall-upload-limit":   "0",
		"user-agent":                 "aria2",
	}
	for i, tab := range settingTabs {
		if tab.raw {
			m.tab = i
			break
		}
	}
	m.filter = "upload"

	defs := m.defs()
	if len(defs) != 1 || defs[0].key != "max-overall-upload-limit" {
		t.Fatalf("filtered defs = %#v", defs)
	}
}

func TestSecuritySettingsExposeWebConfiguration(t *testing.T) {
	var security *settingTab
	for i := range settingTabs {
		if settingTabs[i].name == "Security" {
			security = &settingTabs[i]
			break
		}
	}
	if security == nil {
		t.Fatal("Security settings tab is missing")
	}
	keys := map[string]bool{}
	for _, def := range security.defs {
		keys[def.key] = true
	}
	for _, key := range []string{localWebHost, localWebPort, localWebAuth, localWebPassword} {
		if !keys[key] {
			t.Fatalf("Security settings missing %q", key)
		}
	}
}

func TestNearestExistingDirWalksToAncestor(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "gone", "nested")
	got, moved := nearestExistingDir(missing)
	if !moved {
		t.Fatal("expected stale path to report a fallback")
	}
	if got != root {
		t.Fatalf("fallback = %q, want %q", got, root)
	}
}

func TestTaskHistoryFreezesUntilTaskIsRemoved(t *testing.T) {
	hist, err := history.Open(filepath.Join(t.TempDir(), "history.json"), 10)
	if err != nil {
		t.Fatal(err)
	}
	a := &App{cfg: config.Default(), hist: hist, gidHist: map[string][]float64{}}
	active := aria2.Status{GID: "abc", Status: aria2.StatusActive, DownloadSpeed: aria2.Size(1024)}
	a.Update(pollMsg{active: []aria2.Status{active}})
	if len(a.gidHist["abc"]) != 1 {
		t.Fatalf("active samples = %v", a.gidHist["abc"])
	}
	stopped := active
	stopped.Status = aria2.StatusComplete
	a.Update(pollMsg{stopped: []aria2.Status{stopped}})
	if len(a.gidHist["abc"]) != 1 {
		t.Fatalf("completed history was not frozen: %v", a.gidHist["abc"])
	}
	a.Update(pollMsg{})
	if _, ok := a.gidHist["abc"]; ok {
		t.Fatal("history survived after task disappeared from aria2")
	}
}

func TestCompactTaskCardUsesTwoLines(t *testing.T) {
	a := &App{cfg: config.Default(), gidHist: map[string][]float64{}}
	a.cfg.CompactRows = true
	row := a.renderRow(aria2.Status{GID: "abc", Status: aria2.StatusComplete}, false, 100)
	if got := strings.Count(row, "\n") + 1; got != 2 {
		t.Fatalf("compact row has %d lines, want 2", got)
	}
}

func TestPickerFooterHitboxesMatchRenderedButtons(t *testing.T) {
	p := newDirPicker("Choose folder", t.TempDir(), false, nil, func(string) tea.Cmd { return nil })
	p.setSize(120, 40)
	modal, specs := p.render()
	lines := strings.Split(ansiEscape.ReplaceAllString(modal, ""), "\n")
	a := &App{width: 120, height: 40}
	screen := strings.Split(ansiEscape.ReplaceAllString(a.overlay(modal, specs), ""), "\n")

	want := map[string]string{
		"pbtn:use": "✓ use this folder",
		"pbtn:new": "+ new folder",
	}
	for id, text := range want {
		var spec *hitspec
		for i := range specs {
			if specs[i].id == id {
				spec = &specs[i]
				break
			}
		}
		if spec == nil {
			t.Fatalf("missing hitbox %q", id)
		}
		if spec.y < 0 || spec.y >= len(lines) || !strings.Contains(lines[spec.y], text) {
			t.Fatalf("%s hitbox row %d does not contain %q; line=%q", id, spec.y, text, lineAt(lines, spec.y))
		}
		found := false
		for y, line := range screen {
			if byteIndex := strings.Index(line, text); byteIndex >= 0 {
				found = true
				x := lipgloss.Width(line[:byteIndex])
				if got := a.hitAt(x, y); got != id {
					t.Fatalf("visible %q at (%d,%d) resolves to %q, want %q", text, x, y, got, id)
				}
			}
		}
		if !found {
			t.Fatalf("rendered screen is missing %q", text)
		}
	}
}

func TestPickerFooterMouseActionsAreDirect(t *testing.T) {
	picked := ""
	p := newDirPicker("Choose folder", t.TempDir(), false, nil, func(path string) tea.Cmd {
		picked = path
		return nil
	})
	a := &App{picker: p, cfg: config.Default()}
	if _, _ = a.pickerButtonClick("new"); !p.mkdirMode {
		t.Fatal("New Folder mouse action did not enter mkdir mode")
	}
	p.mkdirMode = false
	if _, _ = a.pickerButtonClick("use"); picked != p.cwd || a.picker != nil {
		t.Fatalf("Use Folder mouse action picked %q and left picker=%v", picked, a.picker != nil)
	}
}

func TestSettingsTabsRenderAsBorderedButtons(t *testing.T) {
	a := &App{width: 120, height: 40}
	m := newSettingsModel()
	block, height := a.renderSettingsTabs(&m)
	if height != 3 || lipgloss.Height(block) != 3 {
		t.Fatalf("settings tabs height = %d/rendered %d, want 3", height, lipgloss.Height(block))
	}
	hits := 0
	for _, hit := range a.hits {
		if strings.HasPrefix(hit.id, "stab:") && hit.y1-hit.y0+1 == 3 {
			hits++
		}
	}
	if hits != len(settingTabs) {
		t.Fatalf("bordered settings tab hitboxes = %d, want %d", hits, len(settingTabs))
	}
}

func lineAt(lines []string, index int) string {
	if index < 0 || index >= len(lines) {
		return "<out of range>"
	}
	return lines[index]
}
