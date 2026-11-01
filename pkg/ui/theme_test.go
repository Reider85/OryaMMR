// theme_test.go — тесты палитры темы: 7 цветов, парность символов, клонирование (§15.2, INV-UI-008).
package ui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	// Фиксируем цветовой профиль: под пайпом (go test) termenv даёт Ascii и
	// все цвета рендерятся одинаково — сравнение палитр было бы бессмысленно.
	lipgloss.SetColorProfile(termenv.ANSI256)
	lipgloss.SetHasDarkBackground(true)
	os.Exit(m.Run())
}

func TestGetColor(t *testing.T) {
	theme := NewTheme()

	for _, name := range []string{"primary", "secondary", "warning", "error", "muted", "accent"} {
		got := theme.GetColor(name).Render("x")
		var want lipgloss.Style
		switch name {
		case "primary":
			want = theme.Primary
		case "secondary":
			want = theme.Secondary
		case "warning":
			want = theme.Warning
		case "error":
			want = theme.Error
		case "muted":
			want = theme.Muted
		case "accent":
			want = theme.Accent
		}
		if w := want.Render("x"); got != w {
			t.Errorf("GetColor(%q) style mismatch", name)
		}
	}

	if got, want := theme.GetColor("no-such-color").Render("x"), theme.Muted.Render("x"); got != want {
		t.Error("unknown color must fall back to Muted")
	}
}

func TestGetStatusColorPairsSymbolWithColor(t *testing.T) {
	// INV-UI-008: каждый статус возвращает и цвет, и текстовый символ.
	theme := NewTheme()
	tests := []struct {
		status string
		symbol string
	}{
		{"healthy", "✓"},
		{"warning", "⚠"},
		{"error", "✗"},
		{"unknown", "?"},
		{"anything-else", "?"}, // default → Muted + "?"
	}
	for _, tc := range tests {
		style, symbol := theme.GetStatusColor(tc.status)
		if symbol != tc.symbol {
			t.Errorf("GetStatusColor(%q) symbol = %q, want %q", tc.status, symbol, tc.symbol)
		}
		if style.Render("s") == "" {
			t.Errorf("GetStatusColor(%q) returned empty style", tc.status)
		}
	}
}

func TestPrecomputeStylesKeepsRendering(t *testing.T) {
	// INV-UI-006: стили предвычисляются на старте и продолжают работать.
	theme := NewTheme()
	theme.PrecomputeStyles()

	if got := theme.Header.Render("hdr"); got == "" {
		t.Error("Header must still render after precompute")
	}
	if got := theme.StatusBar.Render("status"); got == "" {
		t.Error("StatusBar must still render after precompute")
	}
	if got := theme.FocusBorder.Render("focus"); got == "" {
		t.Error("FocusBorder must still render after precompute")
	}
}

func TestCloneIsIndependentCopy(t *testing.T) {
	theme := NewTheme()
	clone := theme.Clone()

	if clone == theme {
		t.Fatal("Clone must return a distinct struct")
	}
	if got, want := clone.Header.Render("x"), theme.Header.Render("x"); got != want {
		t.Error("clone must start identical to the source")
	}

	clone.Header = theme.Error // мутация клона
	if theme.Header.Render("x") == theme.Error.Render("x") {
		t.Error("mutating clone must not affect the source theme")
	}
}

func TestHighContrastThemeDistinctAndWCAG(t *testing.T) {
	theme := NewTheme()
	hc := theme.HighContrast()

	if hc == theme {
		t.Fatal("HighContrast must return a new theme")
	}
	// §19.2: HC-палитра использует только белый/жёлтый на чёрном.
	if hc.Primary.Render("x") == theme.Primary.Render("x") {
		t.Error("HC Primary must differ from default palette")
	}
	// WCAG AAA: парность проверяется в a11y_test (ContrastRatio ≥ 7).
	if hc.Warning.Render("x") == theme.Warning.Render("x") {
		t.Error("HC Warning must differ from default palette")
	}
}
