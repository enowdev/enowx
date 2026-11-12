package ui

import (
	"encoding/json"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestParseAndFormatKey(t *testing.T) {
	tests := []struct {
		input       string
		wantKey     ui.Key
		wantMods    ui.Modifiers
		expectError bool
	}{
		{"Ctrl+K", ui.KeyK, ui.Ctrl, false},
		{"Ctrl+Shift+D", ui.KeyD, ui.Ctrl | ui.Shift, false},
		{"Alt+P", ui.KeyP, ui.Alt, false},
		{"Shift+Enter", ui.KeyEnter, ui.Shift, false},
		{"F5", ui.KeyF5, 0, false},
		{"Ctrl+F12", ui.KeyF12, ui.Ctrl, false},
		{"Ctrl+[", ui.KeyBracketLeft, ui.Ctrl, false},
		{"Ctrl+]", ui.KeyBracketRight, ui.Ctrl, false},
		{"⌘K", ui.KeyK, ui.Cmd, false},
		{"⌘⇧D", ui.KeyD, ui.Cmd | ui.Shift, false},
		{"", 0, 0, true},
		{"Ctrl", 0, 0, true},
		{"Ctrl+Shift", 0, 0, true},
		{"UnknownKey123", 0, 0, true},
		{"Ctrl+Shift+Alt+Unknown", 0, 0, true},
	}

	for _, tc := range tests {
		mods, key, err := ParseKey(tc.input)
		if tc.expectError {
			if err == nil {
				t.Errorf("ParseKey(%q): expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseKey(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if key != tc.wantKey {
			t.Errorf("ParseKey(%q): key = %v, want %v", tc.input, key, tc.wantKey)
		}
		if mods&tc.wantMods != tc.wantMods {
			t.Errorf("ParseKey(%q): mods = %v, want %v", tc.input, mods, tc.wantMods)
		}

		formatted := FormatKey(mods, key)
		if formatted == "" {
			t.Errorf("FormatKey(%v, %v) returned empty string", mods, key)
		}
	}
}

func TestAppShortcutsConfigAndState(t *testing.T) {
	app := New(Options{})

	// 1. Initial defaults
	mods, key, disabled := app.effectiveShortcut(ShortcutPalette)
	if disabled {
		t.Fatal("expected ShortcutPalette to be enabled by default")
	}
	if key != ui.KeyK {
		t.Fatalf("expected ShortcutPalette default key to be ui.KeyK, got %v", key)
	}
	if mods&ui.Cmd == 0 {
		t.Fatalf("expected ShortcutPalette default mods to include ui.Cmd, got %v", mods)
	}
	if app.shortcutLabel(ShortcutPalette) == "" {
		t.Fatal("expected non-empty shortcut label for ShortcutPalette")
	}
	if app.hasCustomShortcuts() {
		t.Fatal("expected no custom shortcuts initially")
	}

	// 2. Disable individual shortcut
	app.setShortcutDisabled(ShortcutPalette, true)
	if !app.hasCustomShortcuts() {
		t.Fatal("expected hasCustomShortcuts to be true after disabling one")
	}
	_, _, disabled = app.effectiveShortcut(ShortcutPalette)
	if !disabled {
		t.Fatal("expected ShortcutPalette to be disabled")
	}
	if app.shortcutLabel(ShortcutPalette) != "" {
		t.Fatalf("expected empty label for disabled shortcut, got %q", app.shortcutLabel(ShortcutPalette))
	}

	// 3. Re-enable and customize key
	app.setShortcutDisabled(ShortcutPalette, false)
	if err := app.setShortcutKey(ShortcutPalette, "Ctrl+Shift+K"); err != nil {
		t.Fatalf("unexpected error setting shortcut key: %v", err)
	}
	mods, key, disabled = app.effectiveShortcut(ShortcutPalette)
	if disabled {
		t.Fatal("expected shortcut to remain enabled after key change")
	}
	if key != ui.KeyK || mods&(ui.Ctrl|ui.Shift) != (ui.Ctrl|ui.Shift) {
		t.Fatalf("expected custom mods and key, got mods=%v key=%v", mods, key)
	}

	// 4. Conflict detection
	conflict, found := app.findShortcutConflict(ShortcutQuickOpen, mods, key)
	if !found || conflict.ID != ShortcutPalette {
		t.Fatalf("expected conflict with ShortcutPalette, found=%v conflict=%+v", found, conflict)
	}

	// 5. Master switch disables all
	app.state.ShortcutsOff = true
	if app.shortcutLabel(ShortcutPalette) != "" {
		t.Fatal("expected empty label when master switch is off")
	}
	if app.shortcutLabel(ShortcutNewSession) != "" {
		t.Fatal("expected empty label for all shortcuts when master switch is off")
	}

	// 6. Reset all shortcuts
	app.resetAllShortcuts()
	if app.state.ShortcutsOff {
		t.Fatal("expected ShortcutsOff to be false after reset")
	}
	if app.hasCustomShortcuts() {
		t.Fatal("expected hasCustomShortcuts to be false after reset")
	}
	mods, key, disabled = app.effectiveShortcut(ShortcutPalette)
	if disabled || key != ui.KeyK {
		t.Fatalf("expected reset to restore default ui.KeyK, got %v (disabled: %v)", key, disabled)
	}
}

func TestShortcutStateJSONPersistence(t *testing.T) {
	st := appState{
		ShortcutsOff: true,
		Shortcuts: map[string]ShortcutConfig{
			string(ShortcutPalette): {
				Disabled: true,
			},
			string(ShortcutNewSession): {
				Custom: true,
				Mods:   ui.Ctrl | ui.Shift,
				Key:    ui.KeyT,
				Text:   "Ctrl+Shift+T",
			},
		},
	}

	data, err := json.Marshal(st)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var restored appState
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if !restored.ShortcutsOff {
		t.Fatal("expected restored ShortcutsOff to be true")
	}
	if len(restored.Shortcuts) != 2 {
		t.Fatalf("expected 2 restored shortcuts, got %d", len(restored.Shortcuts))
	}
	palCfg := restored.Shortcuts[string(ShortcutPalette)]
	if !palCfg.Disabled {
		t.Fatal("expected restored ShortcutPalette to be disabled")
	}
	newCfg := restored.Shortcuts[string(ShortcutNewSession)]
	if !newCfg.Custom || newCfg.Key != ui.KeyT || newCfg.Text != "Ctrl+Shift+T" {
		t.Fatalf("restored ShortcutNewSession mismatch: %+v", newCfg)
	}
}

func TestShortcutDialogUI(t *testing.T) {
	app := New(Options{})

	// Open dialog
	app.openShortcutSheet(ShortcutPalette)
	if app.shortcutsForm == nil || !app.shortcutsForm.open {
		t.Fatal("expected shortcut dialog to be open")
	}
	if app.shortcutsForm.editingID != ShortcutPalette {
		t.Fatalf("expected editingID to be %s, got %s", ShortcutPalette, app.shortcutsForm.editingID)
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(700).Height(800).Children(func() {
			app.shortcutDialog(c)
		})
	}, 700, 800)
	tt.Frame()

	if !tt.HasText("Keyboard shortcuts") {
		t.Fatalf("expected dialog title in %q", tt.Texts())
	}
	if !tt.HasText("Enable keyboard shortcuts") {
		t.Fatalf("expected master switch in %q", tt.Texts())
	}
	if !tt.HasText("Command palette") {
		t.Fatalf("expected Command palette row in %q", tt.Texts())
	}

	// Change key to Ctrl+Shift+K
	app.shortcutsForm.editText = "Ctrl+Shift+K"
	if err := tt.Click("Apply"); err != nil {
		t.Fatalf("clicking Apply failed: %v", err)
	}
	tt.Frame()

	cfg := app.shortcutConfig(ShortcutPalette)
	if !cfg.Custom || cfg.Key != ui.KeyK {
		t.Fatalf("expected custom key to be applied, got %+v", cfg)
	}

	// Click Done to close
	if err := tt.Click("Done"); err != nil {
		t.Fatalf("clicking Done failed: %v", err)
	}
	tt.Frame()
	if app.shortcutsForm != nil {
		t.Fatal("expected dialog to be closed after clicking Done")
	}
}
