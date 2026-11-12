package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/enowx/enowx/core/diag"
	"github.com/enowx/enowx/ui/theme"
)

func TestWindows11FluentStyleIntegration(t *testing.T) {
	app := &App{}
	app.state = appState{
		UIStyle:   "windows-11",
		Theme:     "windows-11",
		ThemeMode: "dark",
	}

	app.applyTheme()

	if style := theme.CurrentStyle(); style.ID != "windows-11" {
		t.Fatalf("expected active style 'windows-11', got %q", style.ID)
	} else {
		if style.RadiusControl != 8 {
			t.Errorf("expected RadiusControl=8, got %.0f", style.RadiusControl)
		}
		if style.RadiusCard != 12 {
			t.Errorf("expected RadiusCard=12, got %.0f", style.RadiusCard)
		}
		if style.RadiusPanel != 12 {
			t.Errorf("expected RadiusPanel=12, got %.0f", style.RadiusPanel)
		}
		if style.Family != theme.FluentFontFamily {
			t.Errorf("expected Family=%q, got %q", theme.FluentFontFamily, style.Family)
		}
	}

	curPal := theme.Current()
	if curPal.ID != "windows-11" {
		t.Fatalf("expected active palette 'windows-11', got %q", curPal.ID)
	}
	if theme.Hex(curPal.Background) != "#202020" {
		t.Errorf("expected dark background #202020, got %s", theme.Hex(curPal.Background))
	}
	if theme.Hex(curPal.Accent) != "#60CDFF" {
		t.Errorf("expected dark accent #60CDFF, got %s", theme.Hex(curPal.Accent))
	}

	// Switch to light mode
	app.setAppearanceMode("light")
	lightPal := theme.Current()
	if !lightPal.Light {
		t.Fatalf("expected light palette, got Light=false")
	}
	if theme.Hex(lightPal.Background) != "#F3F3F3" {
		t.Errorf("expected light background #F3F3F3, got %s", theme.Hex(lightPal.Background))
	}
	if theme.Hex(lightPal.Accent) != "#0067C0" {
		t.Errorf("expected light accent #0067C0, got %s", theme.Hex(lightPal.Accent))
	}
}

func TestWindows11StatePersistenceAndFallback(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")

	// 1. Save valid Windows 11 state
	st := appState{
		UIStyle:   "windows-11",
		Theme:     "windows-11",
		ThemeMode: "light",
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, data, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded := loadState(stateFile)
	if loaded.UIStyle != "windows-11" || loaded.Theme != "windows-11" || loaded.ThemeMode != "light" {
		t.Fatalf("failed to restore Windows 11 state: %+v", loaded)
	}

	// 2. Fallback on corrupt or invalid theme / style IDs
	corruptSt := appState{
		UIStyle: "unknown-corrupt-style",
		Theme:   "unknown-corrupt-theme",
	}
	s := theme.StyleByID(corruptSt.UIStyle)
	if s.ID != theme.LiquidGlass.ID {
		t.Errorf("expected fallback to LiquidGlass on unknown style, got %q", s.ID)
	}
	p := theme.Lookup(corruptSt.Theme, ui.Color{})
	if p.ID != theme.Default.ID {
		t.Errorf("expected fallback to Default on unknown palette, got %q", p.ID)
	}
}

func TestThemeChangedDiagnosticsLogging(t *testing.T) {
	dir := t.TempDir()
	log := diag.Open(dir, "test.log")
	defer log.Close()

	app := &App{
		diag: log,
	}
	app.state = appState{
		UIStyle:   "windows-11",
		Theme:     "windows-11",
		ThemeMode: "dark",
	}

	app.applyTheme()
	log.Flush()

	// Read log file content to verify theme_changed event was written
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no log file generated in %s", dir)
	}
	logBytes, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	logContent := string(logBytes)
	if !strings.Contains(logContent, "theme_changed") {
		t.Errorf("log missing 'theme_changed' event: %s", logContent)
	}
	if !strings.Contains(logContent, "windows-11") {
		t.Errorf("log missing 'windows-11' theme identifier: %s", logContent)
	}
}
