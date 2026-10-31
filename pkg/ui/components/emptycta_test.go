package components

import (
	"strings"
	"testing"
)

// ─── View: обязательные 4 части empty-состояния (§16.3) ──────────────────────

func TestEmptyCTA_View_AllFourParts(t *testing.T) {
	e := NewEmptyCTA(
		"No sessions yet",
		"caf hasn't been invoked in this repo",
		"Run `caf init` to start tracking",
		"[i] run caf init now",
		nil,
	)
	got := e.View()

	// (1) что пусто
	if !strings.Contains(got, "No sessions yet") {
		t.Errorf("View() missing «what is empty», got:\n%s", got)
	}
	// (2) почему
	if !strings.Contains(got, "caf hasn't been invoked in this repo") {
		t.Errorf("View() missing «why», got:\n%s", got)
	}
	// (3) CTA — что делать
	if !strings.Contains(got, "Run `caf init` to start tracking") {
		t.Errorf("View() missing CTA action (§2.6 empty-state CTA), got:\n%s", got)
	}
	// (4) клавиатурная подсказка
	if !strings.Contains(got, "[i] run caf init now") {
		t.Errorf("View() missing keyboard hint (INV-UI-009), got:\n%s", got)
	}
	// INV-PROMPT-MVP-008: символ ○ несёт смысл без цвета
	if !strings.Contains(got, "○") {
		t.Errorf("View() missing ○ symbol (INV-PROMPT-MVP-008):\n%s", got)
	}
}

func TestEmptyCTA_View_EmptyStrings_FilledWithDefaults(t *testing.T) {
	// §16.3: пустой фрейм без объяснения запрещён — пустые поля заменяются.
	e := NewEmptyCTA("", "", "", "", nil)
	got := e.View()
	for _, part := range []string{"Nothing here yet", "No data available", "caf init", "[i]"} {
		if !strings.Contains(got, part) {
			t.Errorf("default-filled View() missing %q:\n%s", part, got)
		}
	}
}

func TestEmptyCTA_NilTheme_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewEmptyCTA(..., nil) panicked: %v", r)
		}
	}()
	NewEmptyCTA("a", "b", "c", "d", nil).View()
}

// ─── AltText (INV-PROMPT-MVP-008, §19.3) ─────────────────────────────────────

func TestEmptyCTA_AltText_StatusFirst(t *testing.T) {
	e := NewEmptyCTA("No adapters installed", "none detected", "install one", "[i] install", nil)
	alt := e.AltText()
	if !strings.HasPrefix(alt, "EMPTY:") {
		t.Errorf("AltText() = %q, want status-first prefix EMPTY: (UI.md §16.3)", alt)
	}
	for _, part := range []string{"No adapters installed", "none detected", "install one", "[i] install"} {
		if !strings.Contains(alt, part) {
			t.Errorf("AltText() = %q, want part %q", alt, part)
		}
	}
}
