package components

import (
	"strings"
	"testing"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// ─── View: обязательные 4 элемента состояния error (§16.3) ───────────────────

func TestErrorBox_View_RequiredElements(t *testing.T) {
	e := NewErrorBox("Cannot load sessions: permission denied", nil, nil)
	got := e.View()

	// (1) сообщение простым языком
	if !strings.Contains(got, "Cannot load sessions: permission denied") {
		t.Errorf("View() missing plain-language message, got:\n%s", got)
	}
	// (2) CTA retry
	if !strings.Contains(got, "[r] retry") {
		t.Errorf("View() missing [r] retry CTA (INV-UI-010: recovery path), got:\n%s", got)
	}
	// (3) подсказка dismiss
	if !strings.Contains(got, "[Esc] dismiss") {
		t.Errorf("View() missing [Esc] dismiss hint, got:\n%s", got)
	}
	// INV-PROMPT-MVP-008: символ ✗ рядом с цветом
	if !strings.Contains(got, "✗") {
		t.Errorf("View() missing ✗ symbol (INV-PROMPT-MVP-008: color is not sole carrier), got:\n%s", got)
	}
}

func TestErrorBox_View_PlainLanguage_NoStack(t *testing.T) {
	// §16.3: «no stack trace, no raw Go error» — компонент ничего не добавляет
	// к переданному сообщению, кроме своего заголовка/CTA.
	e := NewErrorBox("file not found", nil, nil)
	if strings.Contains(e.View(), "goroutine") || strings.Contains(e.View(), ".go:") {
		t.Errorf("View() must not render stack traces:\n%s", e.View())
	}
}

func TestErrorBox_WithFix(t *testing.T) {
	e := NewErrorBox("cannot write to .caf/", nil, nil)
	e.SetFix("check directory permissions")
	got := e.View()
	if !strings.Contains(got, "Fix: check directory permissions") {
		t.Errorf("View() missing suggested fix line (§16.3 recoverable errors):\n%s", got)
	}
	if !strings.Contains(e.AltText(), "Suggested fix: check directory permissions") {
		t.Errorf("AltText() missing suggested fix, got %q", e.AltText())
	}
}

func TestErrorBox_WithoutFix_NoFixLine(t *testing.T) {
	e := NewErrorBox("disk full", nil, nil)
	if strings.Contains(e.View(), "Fix:") {
		t.Errorf("View() must not show Fix line when unset:\n%s", e.View())
	}
}

// ─── Nil-безопасность и high-contrast (§19.2) ────────────────────────────────

func TestErrorBox_NilDeps_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewErrorBox(nil,nil) panicked: %v", r)
		}
	}()
	NewErrorBox("oops", nil, nil).View()
}

func TestErrorBox_HighContrast(t *testing.T) {
	theme := ui.NewTheme()
	keys := ui.NewKeys()
	cfg := ui.NewAccessibilityConfig()
	cfg.HighContrast = true
	a11y := ui.NewA11y(cfg, theme, keys)

	got := NewErrorBox("cannot read config", theme, a11y).View()
	if !strings.Contains(got, "✗") || !strings.Contains(got, "[r] retry") {
		t.Errorf("high-contrast View() lost required elements:\n%s", got)
	}
}

// ─── AltText (INV-PROMPT-MVP-008, §19.3) ─────────────────────────────────────

func TestErrorBox_AltText_StatusFirst(t *testing.T) {
	e := NewErrorBox("timeout while fetching", nil, nil)
	alt := e.AltText()
	if !strings.HasPrefix(alt, "ERROR:") {
		t.Errorf("AltText() = %q, want status-first prefix ERROR: (UI.md §16.3)", alt)
	}
	if !strings.Contains(alt, "Press r to retry") {
		t.Errorf("AltText() = %q, want retry instruction", alt)
	}
}
