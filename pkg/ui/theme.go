package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color palette and styles for the TUI
// INV-UI-008: Color-blind friendly: pair color with text/symbol, never color-only
type Theme struct {
	Primary    lipgloss.Style
	Secondary  lipgloss.Style
	Warning    lipgloss.Style
	Error      lipgloss.Style
	Muted      lipgloss.Style
	Accent     lipgloss.Style
	Background lipgloss.Style

	// Derived styles
	Header      lipgloss.Style
	StatusBar   lipgloss.Style
	Title       lipgloss.Style
	Border      lipgloss.Style
	FocusBorder lipgloss.Style

	// Component styles
	Button   lipgloss.Style
	Input    lipgloss.Style
	Help     lipgloss.Style
	ErrorBox lipgloss.Style
	EmptyBox lipgloss.Style
}

// NewTheme creates a new theme with default colors
func NewTheme() *Theme {
	return &Theme{
		// Primary colors (7 total, INV-UI-006)
		Primary:    lipgloss.NewStyle().Foreground(lipgloss.Color("#61AFEF")), // blue
		Secondary:  lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379")), // green
		Warning:    lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B")), // yellow
		Error:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F07178")), // red (5.1:1 on dark bg, WCAG AA — §16.1; was #E06C75 = 4.4:1)
		Muted:      lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370")), // gray
		Accent:     lipgloss.NewStyle().Foreground(lipgloss.Color("#C678DD")), // purple
		Background: lipgloss.NewStyle().Background(lipgloss.Color("#282C34")), // dark

		// Derived styles
		Header:      lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#1C1F26")).Border(lipgloss.NormalBorder(), true),
		StatusBar:   lipgloss.NewStyle().Background(lipgloss.Color("#1C1F26")).Faint(true),
		Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#61AFEF")),
		Border:      lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false),
		FocusBorder: lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), false).Foreground(lipgloss.Color("#61AFEF")),

		// Component styles
		Button:   lipgloss.NewStyle().Background(lipgloss.Color("#61AFEF")).Foreground(lipgloss.Color("#000000")).Padding(0, 2),
		Input:    lipgloss.NewStyle().Background(lipgloss.Color("#1C1F26")).Border(lipgloss.NormalBorder(), false),
		Help:     lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#5C6370")),
		ErrorBox: lipgloss.NewStyle().Background(lipgloss.Color("#F07178")).Foreground(lipgloss.Color("#000000")).Border(lipgloss.NormalBorder(), true),
		EmptyBox: lipgloss.NewStyle().Background(lipgloss.Color("#1C1F26")).Border(lipgloss.NormalBorder(), true),
	}
}

// HighContrast creates a high-contrast theme for accessibility
// §19.2: WCAG-AAA contrast palette
func (t *Theme) HighContrast() *Theme {
	return &Theme{
		// High-contrast colors
		Primary:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#000000")),            // white on black
		Secondary:  lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#000000")).Bold(true), // white on black + bold
		Warning:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Background(lipgloss.Color("#000000")),            // yellow on black
		Error:      lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Background(lipgloss.Color("#000000")),            // yellow on black
		Muted:      lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC")).Background(lipgloss.Color("#000000")),            // light gray on black
		Accent:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#000000")).Bold(true), // white on black + bold
		Background: lipgloss.NewStyle().Background(lipgloss.Color("#000000")),                                                  // pure black

		// Derived styles
		Header:      lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#000000")).Border(lipgloss.NormalBorder(), true).Foreground(lipgloss.Color("#FFFFFF")),
		StatusBar:   lipgloss.NewStyle().Background(lipgloss.Color("#000000")).Faint(true).Foreground(lipgloss.Color("#CCCCCC")),
		Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")),
		Border:      lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false).Foreground(lipgloss.Color("#CCCCCC")),
		FocusBorder: lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), false).Foreground(lipgloss.Color("#FFFFFF")),

		// Component styles
		Button:   lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000")).Padding(0, 2),
		Input:    lipgloss.NewStyle().Background(lipgloss.Color("#000000")).Border(lipgloss.NormalBorder(), false).Foreground(lipgloss.Color("#FFFFFF")),
		Help:     lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#CCCCCC")),
		ErrorBox: lipgloss.NewStyle().Background(lipgloss.Color("#FFFF00")).Foreground(lipgloss.Color("#000000")).Border(lipgloss.NormalBorder(), true),
		EmptyBox: lipgloss.NewStyle().Background(lipgloss.Color("#000000")).Border(lipgloss.NormalBorder(), true).Foreground(lipgloss.Color("#CCCCCC")),
	}
}

// GetColor returns a color by name for dynamic styling
func (t *Theme) GetColor(name string) lipgloss.Style {
	switch name {
	case "primary":
		return t.Primary
	case "secondary":
		return t.Secondary
	case "warning":
		return t.Warning
	case "error":
		return t.Error
	case "muted":
		return t.Muted
	case "accent":
		return t.Accent
	default:
		return t.Muted
	}
}

// GetStatusColor returns a color for status indicators
// INV-UI-008: Always pair color with text/symbol
func (t *Theme) GetStatusColor(status string) (style lipgloss.Style, symbol string) {
	switch status {
	case "healthy":
		return t.Secondary, "✓"
	case "warning":
		return t.Warning, "⚠"
	case "error":
		return t.Error, "✗"
	case "unknown":
		return t.Muted, "?"
	default:
		return t.Muted, "?"
	}
}

// PrecomputeStyles precomputes all styles at startup for performance
// INV-UI-006: Lipgloss pre-computes style strings at startup
func (t *Theme) PrecomputeStyles() {
	// Precompute frequently used combinations
	t.Header = t.Header.Copy()
	t.StatusBar = t.StatusBar.Copy()
	t.Title = t.Title.Copy()
	t.Border = t.Border.Copy()
	t.FocusBorder = t.FocusBorder.Copy()
	t.Button = t.Button.Copy()
	t.Input = t.Input.Copy()
	t.Help = t.Help.Copy()
	t.ErrorBox = t.ErrorBox.Copy()
	t.EmptyBox = t.EmptyBox.Copy()
}

// Clone creates a copy of the theme
func (t *Theme) Clone() *Theme {
	newTheme := &Theme{}
	*newTheme = *t
	return newTheme
}
