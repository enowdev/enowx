package ui

import (
	"github.com/egoist/mygo/ui"

	"github.com/enowx/enowx/ui/icon"
	"github.com/enowx/enowx/ui/theme"
	"github.com/enowx/enowx/ui/widget"
)

func (a *App) windowControlStyleCard(c *ui.Context) {
	widget.CardTitle(c, nil, "Window controls", nil)
	selected := 0
	if a.isWindowsControlStyle() {
		selected = 1
	}
	ui.Row(c).FillWidth().Children(func() {
		if picked := widget.Segmented(c.Key("window-controls-picker"), []string{"Mac style", "Windows style"}, selected, nil); picked >= 0 {
			if picked == 1 {
				a.setWindowControlStyle(WindowControlStyleWindows)
			} else {
				a.setWindowControlStyle(WindowControlStyleMac)
			}
		}
	})
	ui.Text(c, "Choose between macOS traffic lights on the left or Windows caption buttons on the right.").
		FontSize(theme.TextLabel).TextColor(theme.TextSoft)

	// Visual preview of selected control style
	ui.Row(c).Height(32).AlignItems(ui.Center).Gap(theme.S3).Children(func() {
		ui.Text(c, "Preview:").FontSize(theme.TextLabel).TextColor(theme.TextSoft)
		if a.isWindowsControlStyle() {
			ui.Row(c).Height(24).Radius(theme.RadiusSmall).Background(theme.Panel).BorderColor(theme.Border).BorderWidth(1, 1, 1, 1).Clip().Children(func() {
				ui.Row(c).Size(28, 24).Center().Children(func() { ui.Icon(c, icon.Minus).FontSize(9).TextColor(theme.TextSoft) })
				ui.Row(c).Size(28, 24).Center().Children(func() { ui.Icon(c, icon.Square).FontSize(9).TextColor(theme.TextSoft) })
				ui.Row(c).Size(28, 24).Center().Background(ui.Hex("#E81123")).Children(func() { ui.Icon(c, icon.Close).FontSize(9).TextColor(ui.Hex("#FFFFFF")) })
			})
		} else {
			ui.Row(c).Height(24).PaddingX(theme.S2).Radius(theme.RadiusSmall).Background(theme.Panel).BorderColor(theme.Border).BorderWidth(1, 1, 1, 1).AlignItems(ui.Center).Gap(theme.S2).Children(func() {
				ui.Box(c).Size(10, 10).Radius(theme.RadiusFull).Background(windowDots[0])
				ui.Box(c).Size(10, 10).Radius(theme.RadiusFull).Background(windowDots[1])
				ui.Box(c).Size(10, 10).Radius(theme.RadiusFull).Background(windowDots[2])
			})
		}
	})
}
