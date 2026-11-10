package ui

import (
	"github.com/egoist/mygo/ui"

	"github.com/enowx/enowx/ui/icon"
	"github.com/enowx/enowx/ui/theme"
	"github.com/enowx/enowx/ui/widget"
)

type shortcutDialogState struct {
	open      bool
	editingID ShortcutID
	editText  string
	editErr   string
}

// openShortcutSheet opens the keyboard shortcuts customization dialog.
func (a *App) openShortcutSheet(initialID ...ShortcutID) {
	f := &shortcutDialogState{open: true}
	if len(initialID) > 0 {
		id := initialID[0]
		f.editingID = id
		mods, key, _ := a.effectiveShortcut(id)
		f.editText = FormatKey(mods, key)
	}
	a.shortcutsForm = f
}

// shortcutDialog draws the customizable shortcut form sheet.
func (a *App) shortcutDialog(c *ui.Context) {
	f := a.shortcutsForm
	if f == nil {
		return
	}
	if !f.open {
		a.shortcutsForm = nil
		return
	}

	widget.FormSheet(c, &f.open, 600, "Keyboard shortcuts",
		"Customize keyboard shortcuts or disable the ones you don't need.",
		func() {
			// Master toggle
			masterOn := !a.state.ShortcutsOff
			if switchRowKey(c, "shortcuts-master", &masterOn, "Enable keyboard shortcuts",
				"When turned off, no application keyboard shortcuts will trigger.") {
				a.state.ShortcutsOff = !masterOn
				a.saveState()
			}

			ui.Box(c).Height(theme.Rule).FillWidth().Shrink(0).Background(theme.Border).MarginY(theme.S3)

			ui.Column(c.Key("shortcuts-list")).FillWidth().Gap(theme.S2).Children(func() {
				for _, def := range GlobalShortcuts {
					a.shortcutDialogRow(c, f, def)
				}
			})
		},
		func() {
			if a.hasCustomShortcuts() {
				if widget.GhostButton(c.Key("reset-all-shortcuts"), "Reset all to defaults").Clicked() {
					a.resetAllShortcuts()
					f.editingID = ""
					f.editErr = ""
				}
			}
			ui.Spacer(c)
			if widget.Button(c.Key("done-shortcuts"), widget.Secondary, nil, "Done", "").Clicked() {
				f.open = false
			}
		},
	)
}

func (a *App) shortcutDialogRow(c *ui.Context, f *shortcutDialogState, def ShortcutDef) {
	cfg := a.shortcutConfig(def.ID)
	mods, key, disabled := a.effectiveShortcut(def.ID)
	globallyOff := a.state.ShortcutsOff

	ui.Column(c.Key(string(def.ID))).FillWidth().Padding(theme.S2, theme.S3).
		Radius(theme.RadiusSmall).Background(theme.Fade(theme.Border, 0.4)).Gap(theme.S2).Children(func() {
		ui.Row(c.Key("header")).FillWidth().Gap(theme.S3).AlignItems(ui.Center).Children(func() {
			// Individual shortcut switch
			on := !cfg.Disabled && !globallyOff
			sw := on
			if globallyOff {
				// Master switch is off, so individual switch cannot be enabled
				widget.Switch(c.Key("switch-"+string(def.ID)), &sw, def.Name)
			} else {
				if widget.Switch(c.Key("switch-"+string(def.ID)), &sw, def.Name) {
					a.setShortcutDisabled(def.ID, !sw)
				}
			}

			// Name and Description
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(theme.S2).AlignItems(ui.Center).Children(func() {
					ui.Text(c, def.Name).FontSize(theme.TextUI).FontWeight(500)
					if cfg.Custom {
						widget.Chip(c, "Custom", ui.Color{})
					}
					if disabled || globallyOff {
						widget.Chip(c, "Off", ui.Color{})
					}
				})
				ui.Text(c, def.Description).FontSize(theme.TextLabel).TextColor(theme.TextSoft)
			})

			// Actions: Current shortcut pill and edit/reset buttons
			if f.editingID != def.ID {
				keyStr := FormatKey(mods, key)
				if keyStr == "" || disabled || globallyOff {
					keyStr = "None"
				}
				ui.Text(c, keyStr).FontSize(theme.TextLabel).TextColor(theme.TextSoft).
					Padding(2, 6).Radius(theme.RadiusSmall).Background(theme.Border)

				if !globallyOff {
					if widget.GhostButton(c.Key("btn-edit-"+string(def.ID)), "Change").Clicked() {
						f.editingID = def.ID
						f.editText = FormatKey(mods, key)
						f.editErr = ""
					}
					if cfg.Custom || cfg.Disabled {
						if widget.GhostButton(c.Key("btn-reset-"+string(def.ID)), "Reset").Clicked() {
							a.resetShortcut(def.ID)
							f.editErr = ""
						}
					}
				}
			}
		})

		// Inline editing panel
		if f.editingID == def.ID && !globallyOff {
			ui.Column(c.Key("edit-panel")).FillWidth().Gap(theme.S2).PaddingY(theme.S2).Children(func() {
				ui.Row(c).FillWidth().Gap(theme.S2).AlignItems(ui.Center).Children(func() {
					ui.Row(c).Grow(1).MinWidth(0).Children(func() {
						widget.Input(c.Key("key-input"), &f.editText, "e.g. Ctrl+Shift+K")
					})
					if widget.Button(c.Key("apply-key"), widget.Primary, nil, "Apply", "").Clicked() {
						if err := a.setShortcutKey(def.ID, f.editText); err != nil {
							f.editErr = err.Error()
						} else {
							f.editingID = ""
							f.editErr = ""
						}
					}
					if widget.GhostButton(c.Key("cancel-key"), "Cancel").Clicked() {
						f.editingID = ""
						f.editErr = ""
					}
				})

				// Check potential conflict
				if f.editText != "" {
					if testMods, testKey, err := ParseKey(f.editText); err == nil {
						if conflict, found := a.findShortcutConflict(def.ID, testMods, testKey); found {
							ui.Row(c).Gap(theme.S1).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, icon.Alert).FontSize(12).TextColor(theme.Warning)
								ui.Text(c, "Warning: already assigned to "+conflict.Name).
									FontSize(theme.TextLabel).TextColor(theme.Warning)
							})
						}
					}
				}

				if f.editErr != "" {
					ui.Row(c).Gap(theme.S1).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, icon.Alert).FontSize(12).TextColor(theme.Error)
						ui.Text(c, f.editErr).FontSize(theme.TextLabel).TextColor(theme.Error)
					})
				}
			})
		}
	})
}
