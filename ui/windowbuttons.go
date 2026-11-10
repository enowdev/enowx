package ui

import (
	"github.com/egoist/mygo/ui"

	"github.com/enowx/enowx/ui/icon"
	"github.com/enowx/enowx/ui/theme"
	"github.com/enowx/enowx/ui/widget"
)

// Window control style options.
const (
	WindowControlStyleMac     = "mac"
	WindowControlStyleWindows = "windows"
)

// isWindowsControlStyle reports whether Windows caption buttons are chosen.
func (a *App) isWindowsControlStyle() bool {
	return a.state.WindowControlStyle == WindowControlStyleWindows
}

// setWindowControlStyle sets the window controls appearance ("mac" or "windows").
func (a *App) setWindowControlStyle(style string) {
	if style != WindowControlStyleWindows {
		style = WindowControlStyleMac
	}
	if a.state.WindowControlStyle == style {
		return
	}
	a.state.WindowControlStyle = style
	a.saveState()
	if a.win != nil {
		a.win.Invalidate()
	}
}

// The window's close, minimize and maximize buttons, drawn by the app so
// they look the same on macOS, Windows and Linux: three dots on the left,
// their signs showing while the pointer is over them, grey while the
// window isn't the focused one. The green one does what the system's
// does: full screen on macOS (Option+click zooms), maximize elsewhere.

// windowDots are the buttons' colors, as people know them.
var windowDots = [3]ui.Color{ui.Hex("#FF5F57"), ui.Hex("#FEBC2E"), ui.Hex("#28C840")}

// windowControl is what the buttons do to the window: a *mygo.Window,
// or a test's.
type windowControl interface {
	Close()
	Minimize()
	ToggleMaximize()
	ToggleFullScreen()
}

// windowButtons draws them in the title bar (Mac style traffic lights).
func (a *App) windowButtons(c *ui.Context) {
	win := a.winCtl
	if win == nil && a.win != nil {
		win = a.win
	}
	if win == nil {
		return
	}
	group := ui.Row(c.Key("window-buttons")).Gap(theme.S2).Shrink(0)
	over := group.Hovered()
	group.Children(func() {
		for i, label := range []string{"Close", "Minimize", "Zoom"} {
			col := windowDots[i]
			if a.blurred && !over {
				col = theme.Idle
			}
			b := ui.ButtonBase(c.Key(label)).Size(12, 12).Radius(theme.RadiusSmall).Shrink(0).Background(col).Label(label).Center()
			b.Children(func() {
				if !over {
					return
				}
				sign := ui.RGBA(0, 0, 0, 0.55)
				switch i {
				case 0:
					ui.Icon(c, icon.Close).FontSize(8).TextColor(sign)
				case 1:
					ui.Box(c).Size(6, 1.2).Background(sign)
				default:
					ui.Icon(c, icon.Plus).FontSize(8).TextColor(sign)
				}
			})
			if b.Clicked() {
				switch i {
				case 0:
					win.Close()
				case 1:
					win.Minimize()
				case 2:
					if zoomGoesFullScreen && b.ClickModifiers()&ui.Alt == 0 {
						win.ToggleFullScreen()
					} else {
						win.ToggleMaximize()
					}
				}
			}
		}
	})
}

// windowsWindowButtons draws Windows-style caption buttons (Minimize, Maximize, Close)
// at the right side of the title bar.
func (a *App) windowsWindowButtons(c *ui.Context) {
	win := a.winCtl
	if win == nil && a.win != nil {
		win = a.win
	}
	if win == nil {
		return
	}

	group := ui.Row(c.Key("windows-buttons")).Shrink(0).Height(theme.BarHeight)
	group.Children(func() {
		// 1. Minimize
		minBtn := ui.ButtonBase(c.Key("win-minimize")).
			Height(theme.BarHeight).Width(44).Shrink(0).Center().
			Label("Minimize").Tooltip("Minimize").Cursor(ui.CursorPointer)
		minBg := theme.Fade(theme.Selected, 0)
		if minBtn.Hovered() {
			minBg = theme.Selected
		}
		fg := theme.TextSoft
		if minBtn.Hovered() {
			fg = theme.TextStrong
		} else if a.blurred {
			fg = theme.Idle
		}
		minBtn.Background(widget.QuickTint(minBtn, "bg", minBg)).Children(func() {
			ui.Icon(c, icon.Minus).FontSize(11).TextColor(fg)
		})
		if minBtn.Clicked() {
			win.Minimize()
		}

		// 2. Maximize / Restore
		maxBtn := ui.ButtonBase(c.Key("win-maximize")).
			Height(theme.BarHeight).Width(44).Shrink(0).Center().
			Label("Maximize").Tooltip("Maximize").Cursor(ui.CursorPointer)
		maxBg := theme.Fade(theme.Selected, 0)
		if maxBtn.Hovered() {
			maxBg = theme.Selected
		}
		maxFg := theme.TextSoft
		if maxBtn.Hovered() {
			maxFg = theme.TextStrong
		} else if a.blurred {
			maxFg = theme.Idle
		}
		maxBtn.Background(widget.QuickTint(maxBtn, "bg", maxBg)).Children(func() {
			ui.Icon(c, icon.Square).FontSize(10).TextColor(maxFg)
		})
		if maxBtn.Clicked() {
			if zoomGoesFullScreen && maxBtn.ClickModifiers()&ui.Alt == 0 {
				win.ToggleFullScreen()
			} else {
				win.ToggleMaximize()
			}
		}

		// 3. Close
		closeBtn := ui.ButtonBase(c.Key("win-close")).
			Height(theme.BarHeight).Width(46).Shrink(0).Center().
			Label("Close").Tooltip("Close").Cursor(ui.CursorPointer)
		closeBg := theme.Fade(theme.Selected, 0)
		closeFg := theme.TextSoft
		if closeBtn.Hovered() {
			closeBg = ui.Hex("#E81123") // Windows close red
			closeFg = ui.Hex("#FFFFFF")
		} else if a.blurred {
			closeFg = theme.Idle
		}
		closeBtn.Background(widget.QuickTint(closeBtn, "bg", closeBg)).Children(func() {
			ui.Icon(c, icon.Close).FontSize(11).TextColor(closeFg)
		})
		if closeBtn.Clicked() {
			win.Close()
		}
	})
}

