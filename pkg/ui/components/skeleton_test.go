package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// ─── NewSkeleton: дефолты и защита от пустого вывода ─────────────────────────

func TestNewSkeleton_Defaults(t *testing.T) {
	s := NewSkeleton(0, 0, nil)
	if s.Width != 20 {
		t.Errorf("NewSkeleton(0,0).Width = %d, want 20 (защита от пустого скелетона, §16.3)", s.Width)
	}
	if s.Lines != 3 {
		t.Errorf("NewSkeleton(0,0).Lines = %d, want 3", s.Lines)
	}
}

func TestNewSkeleton_NegativeInput(t *testing.T) {
	s := NewSkeleton(-5, -1, nil)
	if s.Width <= 0 || s.Lines <= 0 {
		t.Errorf("NewSkeleton(-5,-1) produced non-positive shape: %dx%d", s.Width, s.Lines)
	}
}

// ─── View: форма, символ, отсутствие пустого вывода ──────────────────────────

func TestSkeleton_View_LineCount(t *testing.T) {
	s := NewSkeleton(10, 5, nil)
	got := s.View()
	if strings.Count(got, "\n") != 4 {
		t.Errorf("View() has %d newlines, want 4 (5 lines)", strings.Count(got, "\n"))
	}
	if strings.TrimSpace(got) == "" {
		t.Error("View() must never render blank (INV-PROMPT-MVP-006: loading state)")
	}
}

func TestSkeleton_View_RowWidth(t *testing.T) {
	s := NewSkeleton(12, 3, nil)
	for i, line := range strings.Split(s.View(), "\n") {
		if w := lipgloss.Width(line); w != 12 {
			t.Errorf("line %d width = %d, want 12 (форма = форма реальных данных, §17.4)", i, w)
		}
	}
}

func TestSkeleton_View_PlaceholderGlyph(t *testing.T) {
	// INV-PROMPT-MVP-008: символ несёт смысл, цвет не единственный носитель.
	if !strings.Contains(NewSkeleton(8, 2, nil).View(), "░") {
		t.Error("View() must contain placeholder glyph ░ (INV-PROMPT-MVP-008)")
	}
}

func TestSkeleton_View_LargeLineCount_NoPanic(t *testing.T) {
	// Регресс фикса: Lines > 8 раньше падал slice-out-of-range.
	s := NewSkeleton(5, 50, nil)
	if got := strings.Count(s.View(), "\n"); got != 49 {
		t.Errorf("View() for 50 lines has %d newlines, want 49", got)
	}
}

// ─── AltText (INV-PROMPT-MVP-008, §19.3) ─────────────────────────────────────

func TestSkeleton_AltText_StatusFirst(t *testing.T) {
	alt := NewSkeleton(10, 3, nil).AltText()
	if !strings.HasPrefix(alt, "LOADING:") {
		t.Errorf("AltText() = %q, want status-first prefix LOADING: (UI.md §16.3)", alt)
	}
	if !strings.Contains(alt, "3 lines") || !strings.Contains(alt, "10 columns") {
		t.Errorf("AltText() = %q, want shape details", alt)
	}
}
