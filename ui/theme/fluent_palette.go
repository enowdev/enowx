package theme

import "github.com/egoist/mygo/ui"

// Windows11Palette is the Windows 11 Fluent Design System palette: Mica dark #202020 base,
// subtle acrylic layering, and official Windows 11 royal blue / cyan accents.
var Windows11Palette = Palette{
	ID:   "windows-11",
	Name: "Windows 11",

	Ground:       ui.Hex("#202020"),
	Wall:         [3]ui.Color{light("#0078D4", 0.35), light("#60CDFF", 0.25), light("#1F1F1F", 0.35)},
	Background:   ui.Hex("#202020"),
	Panel:        ui.RGBA(39, 39, 39, 0.85),
	Raised:       ui.RGBA(255, 255, 255, 0.05),
	Sunken:       ui.RGBA(0, 0, 0, 0.25),
	Frame:        ui.RGBA(42, 42, 42, 0.92),
	Popover:      ui.RGBA(44, 44, 44, 0.98),
	composer:     ui.RGBA(30, 30, 30, 0.92),
	Hover:        ui.RGBA(255, 255, 255, 0.08),
	Selected:     ui.RGBA(255, 255, 255, 0.15),
	Dialog:       ui.RGBA(44, 44, 44, 0.98),
	Border:       ui.RGBA(255, 255, 255, 0.08),
	BorderStrong: ui.Hex("#8A8A8A"),
	Text:         ui.Hex("#F5F6FA"),
	TextStrong:   ui.Hex("#FFFFFF"),
	TextSoft:     ui.Hex("#D0D2DE"),
	Muted:        ui.Hex("#B4B6C4"),
	Accent:       ui.Hex("#60CDFF"),
	OnAccent:     ui.Hex("#000000"),
	Working:      ui.Hex("#60CDFF"),
	Done:         ui.Hex("#6CCB5F"),
	Warning:      ui.Hex("#FFD60A"),
	Error:        ui.Hex("#FF6961"),
	Idle:         ui.Hex("#AEAEB2"),
	Add:          ui.Hex("#6CCB5F"),
	Del:          ui.Hex("#FF6961"),
	Now:          ui.Hex("#60CDFF"),
	own:          true,
}
