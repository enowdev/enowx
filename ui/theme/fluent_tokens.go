package theme

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// FluentTokens defines the Microsoft Fluent Design System token hierarchy
// for Windows 11 styling across surfaces, strokes, effects, accents, and motion.
type FluentTokens struct {
	IsDark bool

	// Surfaces
	SurfaceBackground ui.Color
	SurfaceCard       ui.Color
	SurfaceCardHover  ui.Color
	SurfaceFlyout     ui.Color
	SurfaceDialog     ui.Color
	SurfaceMica       ui.Color

	// Borders / Strokes
	StrokeSubtle  ui.Color
	StrokeDefault ui.Color
	StrokeCard    ui.Color
	StrokeFocus   ui.Color

	// Effects & Elevation
	ElevationRest      string
	ElevationHover     string
	ElevationFlyout    string
	ElevationDialog    string
	MaterialMicaFilter string

	// Accents
	AccentPrimary      ui.Color
	AccentSecondary    ui.Color
	AccentPressed      ui.Color
	AccentFocusOutline ui.Color
	OnAccent           ui.Color

	// Typography
	FontFamily   string
	TextCaption  float32
	TextBody     float32
	TextSubtitle float32
	TextTitle    float32

	// Corner Radii
	RadiusSmall   float32 // 4px (badge, checkbox, micro)
	RadiusControl float32 // 8px (buttons, inputs)
	RadiusCard    float32 // 12px (cards, containers, flyouts)
	RadiusPanel   float32 // 12px (dialogs, main panels)

	// Motion
	EasingStandard  string
	EasingEnterExit string
	DurationFast    string
	DurationNormal  string
}

// FluentFontFamily is the standard Windows 11 font stack with Segoe UI Variable first.
const FluentFontFamily = "Segoe UI Variable, Segoe UI, system-ui, -apple-system, Roboto, sans-serif"

// FluentDarkTokens holds the Windows 11 Fluent tokens for Dark mode.
var FluentDarkTokens = FluentTokens{
	IsDark: true,

	SurfaceBackground: ui.Hex("#202020"),
	SurfaceCard:       ui.RGBA(255, 255, 255, 0.05),
	SurfaceCardHover:  ui.RGBA(255, 255, 255, 0.08),
	SurfaceFlyout:     ui.RGBA(44, 44, 44, 0.96),
	SurfaceDialog:     ui.RGBA(44, 44, 44, 0.98),
	SurfaceMica:       ui.RGBA(32, 32, 32, 0.88),

	StrokeSubtle:  ui.RGBA(255, 255, 255, 0.08),
	StrokeDefault: ui.RGBA(255, 255, 255, 0.15),
	StrokeCard:    ui.RGBA(255, 255, 255, 0.08),
	StrokeFocus:   ui.Hex("#60CDFF"),

	ElevationRest:      "0 0 0 1px rgba(255, 255, 255, 0.08)",
	ElevationHover:     "0 2px 4px rgba(0, 0, 0, 0.20), 0 0 0 1px rgba(255, 255, 255, 0.12)",
	ElevationFlyout:    "0 8px 16px rgba(0, 0, 0, 0.28), 0 0 0 1px rgba(255, 255, 255, 0.10)",
	ElevationDialog:    "0 32px 64px rgba(0, 0, 0, 0.40), 0 0 0 1px rgba(255, 255, 255, 0.12)",
	MaterialMicaFilter: "blur(30px) saturate(125%)",

	AccentPrimary:      ui.Hex("#60CDFF"),
	AccentSecondary:    ui.Hex("#4CC2FF"),
	AccentPressed:      ui.Hex("#1CA8F8"),
	AccentFocusOutline: ui.Hex("#60CDFF"),
	OnAccent:           ui.Hex("#000000"),

	FontFamily:   FluentFontFamily,
	TextCaption:  12,
	TextBody:     13,
	TextSubtitle: 15,
	TextTitle:    18,

	RadiusSmall:   4,
	RadiusControl: 8,
	RadiusCard:    12,
	RadiusPanel:   12,

	EasingStandard:  "cubic-bezier(0, 0, 0, 1)",
	EasingEnterExit: "cubic-bezier(0.1, 0.9, 0.2, 1)",
	DurationFast:    "150ms",
	DurationNormal:  "250ms",
}

// FluentLightTokens holds the Windows 11 Fluent tokens for Light mode.
var FluentLightTokens = FluentTokens{
	IsDark: false,

	SurfaceBackground: ui.Hex("#F3F3F3"),
	SurfaceCard:       ui.RGBA(255, 255, 255, 0.70),
	SurfaceCardHover:  ui.RGBA(255, 255, 255, 0.90),
	SurfaceFlyout:     ui.RGBA(252, 252, 252, 0.96),
	SurfaceDialog:     ui.Hex("#FFFFFF"),
	SurfaceMica:       ui.RGBA(243, 243, 243, 0.88),

	StrokeSubtle:  ui.RGBA(0, 0, 0, 0.06),
	StrokeDefault: ui.RGBA(0, 0, 0, 0.14),
	StrokeCard:    ui.RGBA(0, 0, 0, 0.06),
	StrokeFocus:   ui.Hex("#0067C0"),

	ElevationRest:      "0 0 0 1px rgba(0, 0, 0, 0.06)",
	ElevationHover:     "0 2px 4px rgba(0, 0, 0, 0.08), 0 0 0 1px rgba(0, 0, 0, 0.08)",
	ElevationFlyout:    "0 8px 16px rgba(0, 0, 0, 0.14), 0 0 0 1px rgba(0, 0, 0, 0.06)",
	ElevationDialog:    "0 32px 64px rgba(0, 0, 0, 0.24), 0 0 0 1px rgba(0, 0, 0, 0.08)",
	MaterialMicaFilter: "blur(30px) saturate(125%)",

	AccentPrimary:      ui.Hex("#0067C0"),
	AccentSecondary:    ui.Hex("#0078D4"),
	AccentPressed:      ui.Hex("#005A9E"),
	AccentFocusOutline: ui.Hex("#0067C0"),
	OnAccent:           ui.Hex("#FFFFFF"),

	FontFamily:   FluentFontFamily,
	TextCaption:  12,
	TextBody:     13,
	TextSubtitle: 15,
	TextTitle:    18,

	RadiusSmall:   4,
	RadiusControl: 8,
	RadiusCard:    12,
	RadiusPanel:   12,

	EasingStandard:  "cubic-bezier(0, 0, 0, 1)",
	EasingEnterExit: "cubic-bezier(0.1, 0.9, 0.2, 1)",
	DurationFast:    "150ms",
	DurationNormal:  "250ms",
}

// GetFluentTokens returns the active token set based on light/dark mode.
func GetFluentTokens(isDark bool) FluentTokens {
	if isDark {
		return FluentDarkTokens
	}
	return FluentLightTokens
}

// ColorRGBAString formats a ui.Color into a standard CSS rgba(r, g, b, a) string.
func ColorRGBAString(c ui.Color) string {
	a := float32(c.A) / 255.0
	if a >= 0.999 {
		return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
	}
	return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", c.R, c.G, c.B, a)
}

// GenerateFluentCSSVariables exports the complete dictionary of Windows 11 Fluent design tokens as CSS variables.
func GenerateFluentCSSVariables(isDark bool) string {
	t := GetFluentTokens(isDark)
	var sb strings.Builder
	sb.WriteString(":root {\n")
	sb.WriteString(fmt.Sprintf("  --font-family: %s;\n", t.FontFamily))
	sb.WriteString(fmt.Sprintf("  --surface-background: %s;\n", ColorRGBAString(t.SurfaceBackground)))
	sb.WriteString(fmt.Sprintf("  --surface-card: %s;\n", ColorRGBAString(t.SurfaceCard)))
	sb.WriteString(fmt.Sprintf("  --surface-card-hover: %s;\n", ColorRGBAString(t.SurfaceCardHover)))
	sb.WriteString(fmt.Sprintf("  --surface-flyout: %s;\n", ColorRGBAString(t.SurfaceFlyout)))
	sb.WriteString(fmt.Sprintf("  --surface-dialog: %s;\n", ColorRGBAString(t.SurfaceDialog)))
	sb.WriteString(fmt.Sprintf("  --surface-mica: %s;\n", ColorRGBAString(t.SurfaceMica)))
	sb.WriteString(fmt.Sprintf("  --stroke-subtle: %s;\n", ColorRGBAString(t.StrokeSubtle)))
	sb.WriteString(fmt.Sprintf("  --stroke-default: %s;\n", ColorRGBAString(t.StrokeDefault)))
	sb.WriteString(fmt.Sprintf("  --stroke-card: %s;\n", ColorRGBAString(t.StrokeCard)))
	sb.WriteString(fmt.Sprintf("  --stroke-focus: %s;\n", ColorRGBAString(t.StrokeFocus)))
	sb.WriteString(fmt.Sprintf("  --elevation-rest: %s;\n", t.ElevationRest))
	sb.WriteString(fmt.Sprintf("  --elevation-hover: %s;\n", t.ElevationHover))
	sb.WriteString(fmt.Sprintf("  --elevation-flyout: %s;\n", t.ElevationFlyout))
	sb.WriteString(fmt.Sprintf("  --elevation-dialog: %s;\n", t.ElevationDialog))
	sb.WriteString(fmt.Sprintf("  --material-mica-filter: %s;\n", t.MaterialMicaFilter))
	sb.WriteString(fmt.Sprintf("  --accent-primary: %s;\n", ColorRGBAString(t.AccentPrimary)))
	sb.WriteString(fmt.Sprintf("  --accent-secondary: %s;\n", ColorRGBAString(t.AccentSecondary)))
	sb.WriteString(fmt.Sprintf("  --accent-pressed: %s;\n", ColorRGBAString(t.AccentPressed)))
	sb.WriteString(fmt.Sprintf("  --accent-focus-outline: %s;\n", ColorRGBAString(t.AccentFocusOutline)))
	sb.WriteString(fmt.Sprintf("  --on-accent: %s;\n", ColorRGBAString(t.OnAccent)))
	sb.WriteString(fmt.Sprintf("  --radius-small: %.0fpx;\n", t.RadiusSmall))
	sb.WriteString(fmt.Sprintf("  --radius-control: %.0fpx;\n", t.RadiusControl))
	sb.WriteString(fmt.Sprintf("  --radius-card: %.0fpx;\n", t.RadiusCard))
	sb.WriteString(fmt.Sprintf("  --radius-panel: %.0fpx;\n", t.RadiusPanel))
	sb.WriteString(fmt.Sprintf("  --easing-standard: %s;\n", t.EasingStandard))
	sb.WriteString(fmt.Sprintf("  --easing-enter-exit: %s;\n", t.EasingEnterExit))
	sb.WriteString(fmt.Sprintf("  --duration-fast: %s;\n", t.DurationFast))
	sb.WriteString(fmt.Sprintf("  --duration-normal: %s;\n", t.DurationNormal))
	sb.WriteString("}\n")

	// Fallback when backdrop-filter is unsupported: solid opaque fallback
	sb.WriteString("\n@supports not ((-webkit-backdrop-filter: blur(1px)) or (backdrop-filter: blur(1px))) {\n")
	sb.WriteString("  :root {\n")
	if isDark {
		sb.WriteString("    --surface-flyout: #2C2C2C;\n")
		sb.WriteString("    --surface-mica: #202020;\n")
	} else {
		sb.WriteString("    --surface-flyout: #FAFAFA;\n")
		sb.WriteString("    --surface-mica: #F3F3F3;\n")
	}
	sb.WriteString("  }\n")
	sb.WriteString("}\n")

	// Accessibility: prefers-reduced-motion media query
	sb.WriteString("\n@media (prefers-reduced-motion: reduce) {\n")
	sb.WriteString("  :root {\n")
	sb.WriteString("    --duration-fast: 0ms;\n")
	sb.WriteString("    --duration-normal: 0ms;\n")
	sb.WriteString("    --easing-standard: linear;\n")
	sb.WriteString("    --easing-enter-exit: linear;\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n")
	return sb.String()
}

// ToMap returns the tokens as key-value string pairs for serialization or inspector tools.
func (t FluentTokens) ToMap() map[string]string {
	return map[string]string{
		"surfaceBackground":  ColorRGBAString(t.SurfaceBackground),
		"surfaceCard":        ColorRGBAString(t.SurfaceCard),
		"surfaceCardHover":   ColorRGBAString(t.SurfaceCardHover),
		"surfaceFlyout":      ColorRGBAString(t.SurfaceFlyout),
		"surfaceDialog":      ColorRGBAString(t.SurfaceDialog),
		"surfaceMica":        ColorRGBAString(t.SurfaceMica),
		"strokeSubtle":       ColorRGBAString(t.StrokeSubtle),
		"strokeDefault":      ColorRGBAString(t.StrokeDefault),
		"strokeCard":         ColorRGBAString(t.StrokeCard),
		"strokeFocus":        ColorRGBAString(t.StrokeFocus),
		"elevationRest":      t.ElevationRest,
		"elevationHover":     t.ElevationHover,
		"elevationFlyout":    t.ElevationFlyout,
		"elevationDialog":    t.ElevationDialog,
		"materialMicaFilter": t.MaterialMicaFilter,
		"accentPrimary":      ColorRGBAString(t.AccentPrimary),
		"accentSecondary":    ColorRGBAString(t.AccentSecondary),
		"accentPressed":      ColorRGBAString(t.AccentPressed),
		"accentFocusOutline": ColorRGBAString(t.AccentFocusOutline),
		"onAccent":           ColorRGBAString(t.OnAccent),
		"fontFamily":         t.FontFamily,
		"radiusSmall":        fmt.Sprintf("%.0fpx", t.RadiusSmall),
		"radiusControl":      fmt.Sprintf("%.0fpx", t.RadiusControl),
		"radiusCard":         fmt.Sprintf("%.0fpx", t.RadiusCard),
		"radiusPanel":        fmt.Sprintf("%.0fpx", t.RadiusPanel),
		"easingStandard":     t.EasingStandard,
		"easingEnterExit":    t.EasingEnterExit,
		"durationFast":       t.DurationFast,
		"durationNormal":     t.DurationNormal,
	}
}
