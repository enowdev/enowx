package ui

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// ShortcutID identifies a customizable application shortcut.
type ShortcutID string

const (
	ShortcutPalette       ShortcutID = "palette"
	ShortcutQuickOpen     ShortcutID = "quick_open"
	ShortcutNewSession    ShortcutID = "new_session"
	ShortcutOpenFolder    ShortcutID = "open_folder"
	ShortcutSidebar       ShortcutID = "sidebar"
	ShortcutSidePanel     ShortcutID = "side_panel"
	ShortcutSessionsPanel ShortcutID = "sessions_panel"
	ShortcutBrowserTool   ShortcutID = "browser_tool"
	ShortcutSplitView     ShortcutID = "split_view"
	ShortcutNextSession   ShortcutID = "next_session"
	ShortcutPrevSession   ShortcutID = "prev_session"
	ShortcutMaximize      ShortcutID = "maximize"
	ShortcutHibernate     ShortcutID = "hibernate"
)

// ShortcutDef describes a registered global application shortcut.
type ShortcutDef struct {
	ID          ShortcutID
	Name        string
	Description string
	DefaultMods ui.Modifiers
	DefaultKey  ui.Key
}

// ShortcutConfig holds user overrides and disable state for a shortcut.
type ShortcutConfig struct {
	Disabled bool         `json:"disabled,omitempty"`
	Mods     ui.Modifiers `json:"mods,omitempty"`
	Key      ui.Key       `json:"key,omitempty"`
	Custom   bool         `json:"custom,omitempty"`
	Text     string       `json:"text,omitempty"`
}

// GlobalShortcuts lists all configurable global shortcuts in presentation order.
var GlobalShortcuts = []ShortcutDef{
	{ID: ShortcutPalette, Name: "Command palette", Description: "Find and run commands", DefaultMods: ui.Cmd, DefaultKey: ui.KeyK},
	{ID: ShortcutQuickOpen, Name: "Go to file", Description: "Quick open a file of the session", DefaultMods: ui.Cmd, DefaultKey: ui.KeyP},
	{ID: ShortcutNewSession, Name: "New session", Description: "Start a new agent session", DefaultMods: ui.Cmd, DefaultKey: ui.KeyN},
	{ID: ShortcutOpenFolder, Name: "Open folder", Description: "Open a folder or project", DefaultMods: ui.Cmd, DefaultKey: ui.KeyO},
	{ID: ShortcutSidebar, Name: "Expand or collapse sidebar", Description: "Toggle wide / narrow rail", DefaultMods: ui.Cmd, DefaultKey: ui.KeyB},
	{ID: ShortcutSidePanel, Name: "Show or hide side panel", Description: "Toggle side panel tools", DefaultMods: ui.Cmd | ui.Shift, DefaultKey: ui.KeyD},
	{ID: ShortcutSessionsPanel, Name: "Show or hide Sessions panel", Description: "Toggle sessions list panel", DefaultMods: ui.Cmd | ui.Shift, DefaultKey: ui.KeyS},
	{ID: ShortcutBrowserTool, Name: "Browser tool", Description: "Open the built-in browser", DefaultMods: ui.Cmd | ui.Shift, DefaultKey: ui.KeyP},
	{ID: ShortcutSplitView, Name: "Sessions side by side", Description: "Toggle split view for sessions", DefaultMods: ui.Cmd | ui.Shift, DefaultKey: ui.KeyL},
	{ID: ShortcutNextSession, Name: "Next session", Description: "Switch to next session in project", DefaultMods: ui.Cmd | ui.Shift, DefaultKey: ui.KeyBracketRight},
	{ID: ShortcutPrevSession, Name: "Previous session", Description: "Switch to previous session in project", DefaultMods: ui.Cmd | ui.Shift, DefaultKey: ui.KeyBracketLeft},
	{ID: ShortcutMaximize, Name: "Maximize session", Description: "Expand session to full workspace", DefaultMods: ui.Cmd | ui.Shift, DefaultKey: ui.KeyEnter},
	{ID: ShortcutHibernate, Name: "Hibernate session", Description: "Hibernate selected idle session", DefaultMods: ui.Cmd | ui.Shift, DefaultKey: ui.KeyH},
}

// LookupShortcut finds a shortcut definition by its ID.
func LookupShortcut(id ShortcutID) (ShortcutDef, bool) {
	for _, def := range GlobalShortcuts {
		if def.ID == id {
			return def, true
		}
	}
	return ShortcutDef{}, false
}

// FormatKey formats modifiers and a key into a human-readable display string.
func FormatKey(mods ui.Modifiers, key ui.Key) string {
	keyName := formatSingleKey(key)
	if keyName == "" {
		return ""
	}

	if runtime.GOOS == "darwin" {
		var b strings.Builder
		if mods&ui.Ctrl != 0 {
			b.WriteString("^")
		}
		if mods&ui.Alt != 0 {
			b.WriteString("⌥")
		}
		if mods&ui.Shift != 0 {
			b.WriteString("⇧")
		}
		if mods&(ui.Super|ui.Cmd) != 0 {
			b.WriteString("⌘")
		}
		b.WriteString(keyName)
		return b.String()
	}

	var parts []string
	if mods&ui.Ctrl != 0 || (mods&ui.Cmd != 0 && ui.Cmd == ui.Ctrl) {
		parts = append(parts, "Ctrl")
	}
	if mods&ui.Alt != 0 {
		parts = append(parts, "Alt")
	}
	if mods&ui.Shift != 0 {
		parts = append(parts, "Shift")
	}
	if mods&ui.Super != 0 && ui.Cmd != ui.Super {
		parts = append(parts, "Win")
	}
	parts = append(parts, keyName)
	return strings.Join(parts, "+")
}

func formatSingleKey(key ui.Key) string {
	switch {
	case key >= ui.KeyA && key <= ui.KeyZ:
		return string(rune('A' + int(key-ui.KeyA)))
	case key >= ui.Key0 && key <= ui.Key9:
		return string(rune('0' + int(key-ui.Key0)))
	case key >= ui.KeyF1 && key <= ui.KeyF12:
		return fmt.Sprintf("F%d", int(key-ui.KeyF1)+1)
	}

	switch key {
	case ui.KeyEnter:
		if runtime.GOOS == "darwin" {
			return "↩"
		}
		return "Enter"
	case ui.KeyEscape:
		return "Esc"
	case ui.KeyTab:
		return "Tab"
	case ui.KeySpace:
		return "Space"
	case ui.KeyBackspace:
		return "Backspace"
	case ui.KeyDelete:
		return "Delete"
	case ui.KeyBracketLeft:
		return "["
	case ui.KeyBracketRight:
		return "]"
	case ui.KeySlash:
		return "/"
	case ui.KeyBackslash:
		return "\\"
	case ui.KeyMinus:
		return "-"
	case ui.KeyEqual:
		return "="
	case ui.KeyComma:
		return ","
	case ui.KeyPeriod:
		return "."
	case ui.KeySemicolon:
		return ";"
	case ui.KeyQuote:
		return "'"
	case ui.KeyBackquote:
		return "`"
	case ui.KeyUp:
		return "Up"
	case ui.KeyDown:
		return "Down"
	case ui.KeyLeft:
		return "Left"
	case ui.KeyRight:
		return "Right"
	case ui.KeyHome:
		return "Home"
	case ui.KeyEnd:
		return "End"
	case ui.KeyPageUp:
		return "PageUp"
	case ui.KeyPageDown:
		return "PageDown"
	default:
		return ""
	}
}

// ParseKey parses a shortcut string (e.g. "Ctrl+Shift+D" or "⌘⇧D") into modifiers and a key.
func ParseKey(s string) (ui.Modifiers, ui.Key, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ui.KeyUnknown, errors.New("shortcut cannot be empty")
	}

	// Normalize Mac glyphs into standard token representation
	s = strings.ReplaceAll(s, "⌘", "+cmd+")
	s = strings.ReplaceAll(s, "⇧", "+shift+")
	s = strings.ReplaceAll(s, "⌥", "+alt+")
	s = strings.ReplaceAll(s, "^", "+ctrl+")
	s = strings.ReplaceAll(s, "↩", "+enter+")

	rawTokens := strings.FieldsFunc(s, func(r rune) bool {
		return r == '+' || r == '-' || unicode.IsSpace(r)
	})

	var mods ui.Modifiers
	var foundKey ui.Key = ui.KeyUnknown

	for _, raw := range rawTokens {
		token := strings.TrimSpace(strings.ToLower(raw))
		if token == "" {
			continue
		}

		switch token {
		case "ctrl", "control":
			mods |= ui.Ctrl
		case "alt", "opt", "option":
			mods |= ui.Alt
		case "shift":
			mods |= ui.Shift
		case "cmd", "command":
			mods |= ui.Cmd
		case "super", "win", "windows":
			mods |= ui.Super
		default:
			// Parse key
			k, err := parseSingleKey(token)
			if err != nil {
				return 0, ui.KeyUnknown, err
			}
			if foundKey != ui.KeyUnknown {
				return 0, ui.KeyUnknown, fmt.Errorf("multiple keys specified (%s and %s)", formatSingleKey(foundKey), raw)
			}
			foundKey = k
		}
	}

	if foundKey == ui.KeyUnknown {
		return 0, ui.KeyUnknown, errors.New("a key (e.g. K, Enter, or F1) is required")
	}

	return mods, foundKey, nil
}

func parseSingleKey(s string) (ui.Key, error) {
	upper := strings.ToUpper(s)

	if len(upper) == 1 {
		r := rune(upper[0])
		if r >= 'A' && r <= 'Z' {
			return ui.Key(int(ui.KeyA) + int(r-'A')), nil
		}
		if r >= '0' && r <= '9' {
			return ui.Key(int(ui.Key0) + int(r-'0')), nil
		}
	}

	if strings.HasPrefix(upper, "F") && len(upper) >= 2 && len(upper) <= 3 {
		if n, err := strconv.Atoi(upper[1:]); err == nil && n >= 1 && n <= 12 {
			return ui.Key(int(ui.KeyF1) + (n - 1)), nil
		}
	}

	switch upper {
	case "ENTER", "RETURN":
		return ui.KeyEnter, nil
	case "ESC", "ESCAPE":
		return ui.KeyEscape, nil
	case "TAB":
		return ui.KeyTab, nil
	case "SPACE":
		return ui.KeySpace, nil
	case "BACKSPACE":
		return ui.KeyBackspace, nil
	case "DEL", "DELETE":
		return ui.KeyDelete, nil
	case "[", "BRACKETLEFT":
		return ui.KeyBracketLeft, nil
	case "]", "BRACKETRIGHT":
		return ui.KeyBracketRight, nil
	case "/", "SLASH":
		return ui.KeySlash, nil
	case "\\", "BACKSLASH":
		return ui.KeyBackslash, nil
	case "-", "MINUS":
		return ui.KeyMinus, nil
	case "=", "EQUAL":
		return ui.KeyEqual, nil
	case ",", "COMMA":
		return ui.KeyComma, nil
	case ".", "PERIOD":
		return ui.KeyPeriod, nil
	case ";", "SEMICOLON":
		return ui.KeySemicolon, nil
	case "'", "QUOTE":
		return ui.KeyQuote, nil
	case "`", "BACKQUOTE":
		return ui.KeyBackquote, nil
	case "UP":
		return ui.KeyUp, nil
	case "DOWN":
		return ui.KeyDown, nil
	case "LEFT":
		return ui.KeyLeft, nil
	case "RIGHT":
		return ui.KeyRight, nil
	case "HOME":
		return ui.KeyHome, nil
	case "END":
		return ui.KeyEnd, nil
	case "PAGEUP":
		return ui.KeyPageUp, nil
	case "PAGEDOWN":
		return ui.KeyPageDown, nil
	default:
		return ui.KeyUnknown, fmt.Errorf("unknown key %q", s)
	}
}

// shortcutConfig returns the user's config for id, or an empty config if unset.
func (a *App) shortcutConfig(id ShortcutID) ShortcutConfig {
	if a.state.Shortcuts != nil {
		if cfg, ok := a.state.Shortcuts[string(id)]; ok {
			return cfg
		}
	}
	return ShortcutConfig{}
}

// effectiveShortcut returns the active modifiers, key, and whether the shortcut is disabled.
func (a *App) effectiveShortcut(id ShortcutID) (mods ui.Modifiers, key ui.Key, disabled bool) {
	def, ok := LookupShortcut(id)
	if !ok {
		return 0, ui.KeyUnknown, true
	}
	cfg := a.shortcutConfig(id)
	if cfg.Disabled {
		return 0, ui.KeyUnknown, true
	}
	if cfg.Custom && cfg.Key != ui.KeyUnknown {
		return cfg.Mods, cfg.Key, false
	}
	return def.DefaultMods, def.DefaultKey, false
}

// isShortcutTriggered reports whether the shortcut for id fired in context c.
func (a *App) isShortcutTriggered(id ShortcutID, c *ui.Context) bool {
	if a.state.ShortcutsOff {
		return false
	}
	mods, key, disabled := a.effectiveShortcut(id)
	if disabled || key == ui.KeyUnknown {
		return false
	}
	return c.Shortcut(mods, key)
}

// shortcutLabel returns the formatted label for the shortcut, or "" if disabled or turned off.
func (a *App) shortcutLabel(id ShortcutID) string {
	if a.state.ShortcutsOff {
		return ""
	}
	mods, key, disabled := a.effectiveShortcut(id)
	if disabled || key == ui.KeyUnknown {
		return ""
	}
	return FormatKey(mods, key)
}

// setShortcutDisabled enables or disables a single shortcut.
func (a *App) setShortcutDisabled(id ShortcutID, disabled bool) {
	if a.state.Shortcuts == nil {
		a.state.Shortcuts = make(map[string]ShortcutConfig)
	}
	cfg := a.state.Shortcuts[string(id)]
	cfg.Disabled = disabled
	a.state.Shortcuts[string(id)] = cfg
	a.saveState()
}

// setShortcutKey changes the key combination of a shortcut.
func (a *App) setShortcutKey(id ShortcutID, text string) error {
	mods, key, err := ParseKey(text)
	if err != nil {
		return err
	}
	if a.state.Shortcuts == nil {
		a.state.Shortcuts = make(map[string]ShortcutConfig)
	}
	cfg := a.state.Shortcuts[string(id)]
	cfg.Custom = true
	cfg.Mods = mods
	cfg.Key = key
	cfg.Text = FormatKey(mods, key)
	a.state.Shortcuts[string(id)] = cfg
	a.saveState()
	return nil
}

// resetShortcut restores a shortcut to its factory default.
func (a *App) resetShortcut(id ShortcutID) {
	if a.state.Shortcuts != nil {
		delete(a.state.Shortcuts, string(id))
		a.saveState()
	}
}

// resetAllShortcuts enables all shortcuts and restores their default bindings.
func (a *App) resetAllShortcuts() {
	a.state.ShortcutsOff = false
	a.state.Shortcuts = nil
	a.saveState()
}

// hasCustomShortcuts reports whether any shortcut is disabled, custom, or master switch is off.
func (a *App) hasCustomShortcuts() bool {
	if a.state.ShortcutsOff {
		return true
	}
	return len(a.state.Shortcuts) > 0
}

// findShortcutConflict checks if mods and key are already used by another active shortcut.
func (a *App) findShortcutConflict(id ShortcutID, mods ui.Modifiers, key ui.Key) (ShortcutDef, bool) {
	for _, def := range GlobalShortcuts {
		if def.ID == id {
			continue
		}
		otherMods, otherKey, disabled := a.effectiveShortcut(def.ID)
		if !disabled && otherMods == mods && otherKey == key {
			return def, true
		}
	}
	return ShortcutDef{}, false
}
