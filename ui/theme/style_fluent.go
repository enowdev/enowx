package theme

import (
	"github.com/egoist/mygo/ui"
)

// Windows11 is the Windows 11 Fluent Design System style: Mica/Acrylic translucency,
// 8px rounded corners for controls, 12px for cards and panels, Segoe UI Variable,
// and subtle layered borders.
var Windows11 = Style{
	ID:   "windows-11",
	Name: "Windows 11",

	// Radii: 4px micro, 8px controls, 12px cards and panels
	RadiusFull:    8,
	RadiusControl: 8,
	RadiusTail:    4,
	RadiusSmall:   4,
	RadiusCard:    12,
	RadiusPanel:   12,

	// Spacing and control metrics
	BentoGap:      8,
	WindowInset:   10,
	ControlHeight: 32,
	BarHeight:     40,
	ScrollWidth:   8,
	BarRadius:     8,
	RailWidth:     44,
	ReadWidth:     760,
	HeaderHeight:  52,
	RowHeight:     34,
	RowTwo:        40,
	RowTall:       42,
	TabHeight:     28,
	Rule:          1,

	S1: 4, S2: 8, S3: 12, S4: 16, S5: 24, S6: 32,

	// Typography: Segoe UI Variable hierarchy
	TextLabel: 12,
	TextUI:    13,
	TextRead:  15,
	TextTitle: 18,

	Family:   FluentFontFamily,
	Scrim:    ui.RGBA(0, 0, 0, 0.45),
	Hairline: 1,
	surfaces: fluentSurfaces,
}

// fluentSurfaces maps a palette to Windows 11 Fluent Design System layers.
func fluentSurfaces(p Palette) Palette {
	t := GetFluentTokens(p.Light == false)

	if p.Light {
		p.Ground = t.SurfaceBackground
		p.Background = t.SurfaceBackground
		p.Panel = t.SurfaceCard
		p.Frame = ui.RGBA(249, 249, 249, 0.95)
		p.Raised = t.SurfaceCard
		p.Sunken = ui.RGBA(0, 0, 0, 0.04)
		p.Hover = t.SurfaceCardHover
		p.Selected = ui.RGBA(0, 0, 0, 0.08)
		p.Dialog = t.SurfaceDialog
		p.Popover = t.SurfaceFlyout
		p.composer = t.SurfaceDialog

		p.Border = t.StrokeSubtle
		p.BorderStrong = t.StrokeDefault
		p.floatLine = ui.RGBA(0, 0, 0, 0.10)

		p.Text = ui.Hex("#1A1A1A")
		p.TextStrong = ui.Hex("#000000")
		p.TextSoft = ui.RGBA(0, 0, 0, 0.62)
		p.Muted = ui.RGBA(0, 0, 0, 0.44)

		if p.own {
			p.Accent = t.AccentPrimary
			p.OnAccent = t.OnAccent
		}

		p.Working = t.AccentPrimary
		p.Done = ui.Hex("#107C10")
		p.Warning = ui.Hex("#9D5D00")
		p.Error = ui.Hex("#C42B1C")
		p.Idle = ui.Hex("#5F6673")
		p.Add = ui.Hex("#107C10")
		p.Del = ui.Hex("#C42B1C")
		p.Now = t.AccentPrimary
	} else {
		p.Ground = t.SurfaceBackground
		p.Background = t.SurfaceBackground
		p.Panel = ui.RGBA(39, 39, 39, 0.90)
		p.Frame = ui.RGBA(36, 36, 36, 0.95)
		p.Raised = t.SurfaceCard
		p.Sunken = ui.RGBA(0, 0, 0, 0.28)
		p.Hover = t.SurfaceCardHover
		p.Selected = ui.RGBA(255, 255, 255, 0.12)
		p.Dialog = t.SurfaceDialog
		p.Popover = t.SurfaceFlyout
		p.composer = ui.RGBA(30, 30, 30, 0.92)

		p.Border = t.StrokeSubtle
		p.BorderStrong = t.StrokeDefault
		p.floatLine = ui.RGBA(255, 255, 255, 0.12)

		p.Text = ui.Hex("#FFFFFF")
		p.TextStrong = ui.Hex("#FFFFFF")
		p.TextSoft = ui.RGBA(255, 255, 255, 0.72)
		p.Muted = ui.RGBA(255, 255, 255, 0.54)

		if p.own {
			p.Accent = t.AccentPrimary
			p.OnAccent = t.OnAccent
		}

		p.Working = t.AccentPrimary
		p.Done = ui.Hex("#6CCB5F")
		p.Warning = ui.Hex("#FCE100")
		p.Error = ui.Hex("#FF99A4")
		p.Idle = ui.Hex("#AEAEB2")
		p.Add = ui.Hex("#6CCB5F")
		p.Del = ui.Hex("#FF99A4")
		p.Now = t.AccentPrimary
	}

	p.OnAccent = contrastInk(p.Accent)
	return holdWords(p)
}
