package theme

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestFluentTokensStructureAndValues(t *testing.T) {
	for _, dark := range []bool{true, false} {
		tokens := GetFluentTokens(dark)

		if tokens.FontFamily == "" {
			t.Errorf("dark=%v: FontFamily is empty", dark)
		}
		if tokens.RadiusSmall != 4 || tokens.RadiusControl != 8 || tokens.RadiusCard != 12 || tokens.RadiusPanel != 12 {
			t.Errorf("dark=%v: unexpected corner radii: small=%.0f, control=%.0f, card=%.0f, panel=%.0f",
				dark, tokens.RadiusSmall, tokens.RadiusControl, tokens.RadiusCard, tokens.RadiusPanel)
		}
		if tokens.AccentPrimary.A == 0 {
			t.Errorf("dark=%v: AccentPrimary is zero", dark)
		}
		if tokens.SurfaceBackground.A == 0 || tokens.SurfaceCard.A == 0 || tokens.SurfaceFlyout.A == 0 {
			t.Errorf("dark=%v: Surface colors have zero alpha", dark)
		}
		if tokens.StrokeSubtle.A == 0 || tokens.StrokeDefault.A == 0 {
			t.Errorf("dark=%v: Stroke colors have zero alpha", dark)
		}
		if tokens.ElevationRest == "" || tokens.ElevationFlyout == "" || tokens.ElevationDialog == "" {
			t.Errorf("dark=%v: Elevation shadows are empty", dark)
		}
		if tokens.EasingStandard == "" || tokens.DurationFast == "" {
			t.Errorf("dark=%v: Motion tokens are empty", dark)
		}

		dict := tokens.ToMap()
		requiredKeys := []string{
			"surfaceBackground", "surfaceCard", "surfaceCardHover", "surfaceFlyout",
			"strokeSubtle", "strokeDefault", "strokeCard", "strokeFocus",
			"elevationRest", "elevationHover", "elevationFlyout", "materialMicaFilter",
			"accentPrimary", "accentSecondary", "accentPressed", "accentFocusOutline",
			"radiusSmall", "radiusControl", "radiusCard", "radiusPanel",
			"easingStandard", "durationFast",
		}
		for _, key := range requiredKeys {
			if val, ok := dict[key]; !ok || val == "" {
				t.Errorf("dark=%v: missing or empty token key %q in ToMap()", dark, key)
			}
		}
	}
}

func TestGenerateFluentCSSVariables(t *testing.T) {
	for _, dark := range []bool{true, false} {
		css := GenerateFluentCSSVariables(dark)
		expectedVars := []string{
			"--font-family:",
			"--surface-background:",
			"--surface-card:",
			"--surface-card-hover:",
			"--surface-flyout:",
			"--surface-dialog:",
			"--surface-mica:",
			"--stroke-subtle:",
			"--stroke-default:",
			"--stroke-card:",
			"--stroke-focus:",
			"--elevation-rest:",
			"--elevation-hover:",
			"--elevation-flyout:",
			"--elevation-dialog:",
			"--material-mica-filter:",
			"--accent-primary:",
			"--accent-secondary:",
			"--accent-pressed:",
			"--accent-focus-outline:",
			"--on-accent:",
			"--radius-small: 4px;",
			"--radius-control: 8px;",
			"--radius-card: 12px;",
			"--radius-panel: 12px;",
			"--easing-standard:",
			"--duration-fast: 150ms;",
			"@supports not ((-webkit-backdrop-filter",
			"@media (prefers-reduced-motion: reduce)",
		}
		for _, exp := range expectedVars {
			if !strings.Contains(css, exp) {
				t.Errorf("dark=%v: CSS variables missing %q", dark, exp)
			}
		}
	}
}

func TestWindows11StyleLookupAndSurfaces(t *testing.T) {
	s := StyleByID("windows-11")
	if s.ID != Windows11.ID {
		t.Fatalf("expected style ID %q, got %q", Windows11.ID, s.ID)
	}
	if s.RadiusControl != 8 || s.RadiusCard != 12 || s.RadiusPanel != 12 {
		t.Fatalf("unexpected Windows11 style metrics: %+v", s)
	}

	pDark := fluentSurfaces(Windows11Palette)
	if pDark.Background.A == 0 || pDark.Text.A == 0 || pDark.Accent.A == 0 {
		t.Fatalf("dark fluent surfaces returned invalid palette: %+v", pDark)
	}

	pLight := fluentSurfaces(lightOf(Windows11Palette, ui.Color{}))
	if pLight.Background.A == 0 || pLight.Text.A == 0 || pLight.Accent.A == 0 {
		t.Fatalf("light fluent surfaces returned invalid palette: %+v", pLight)
	}
}

func TestWindows11PaletteLookup(t *testing.T) {
	p := Lookup("windows-11", ui.Color{})
	if p.ID != "windows-11" {
		t.Fatalf("expected 'windows-11', got %q", p.ID)
	}
	if p.Name != "Windows 11" {
		t.Fatalf("expected 'Windows 11', got %q", p.Name)
	}
}
