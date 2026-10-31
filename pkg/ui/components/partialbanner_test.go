package components

import (
	"strings"
	"testing"
)

// ─── View: баннер показывает что загружено и чего не хватает (§16.3) ────────

func TestPartialBanner_View_CoreLine(t *testing.T) {
	p := NewPartialBanner(3, 5, []string{"cross-repo (timeout)", "agents"}, nil)
	got := p.View()

	if !strings.Contains(got, "⚠ Partial data: 3 of 5 loaded.") {
		t.Errorf("View() missing core counts line, got:\n%s", got)
	}
	if !strings.Contains(got, "Missing: cross-repo (timeout), agents") {
		t.Errorf("View() missing «what's missing» list (никогда не деградируем молча, INV-PROMPT-MVP-006), got:\n%s", got)
	}
	if !strings.Contains(got, "[r] retry") {
		t.Errorf("View() missing [r] retry CTA (INV-UI-010), got:\n%s", got)
	}
	// INV-PROMPT-MVP-008: символ ⚠ несёт смысл без цвета
	if !strings.Contains(got, "⚠") {
		t.Errorf("View() missing ⚠ symbol (INV-PROMPT-MVP-008):\n%s", got)
	}
}

func TestPartialBanner_View_EmptyMissingList(t *testing.T) {
	p := NewPartialBanner(4, 5, nil, nil)
	got := p.View()
	if strings.Contains(got, "Missing:") {
		t.Errorf("View() must not show Missing line when list is empty:\n%s", got)
	}
}

// ─── Нормализация счётчиков: баннер не должен врать ──────────────────────────

func TestPartialBanner_Normalization_TotalBelowLoaded(t *testing.T) {
	p := NewPartialBanner(7, 3, nil, nil)
	if p.Total != 7 {
		t.Errorf("Total = %d, want 7 (total < loaded нормализуется к loaded)", p.Total)
	}
	if !strings.Contains(p.View(), "7 of 7 loaded.") {
		t.Errorf("View() shows un-normalized counts:\n%s", p.View())
	}
}

func TestPartialBanner_Normalization_NegativeLoaded(t *testing.T) {
	p := NewPartialBanner(-2, 5, nil, nil)
	if p.Loaded != 0 {
		t.Errorf("Loaded = %d, want 0", p.Loaded)
	}
	if p.Total != 5 {
		t.Errorf("Total = %d, want 5", p.Total)
	}
}

func TestPartialBanner_NilTheme_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewPartialBanner(..., nil) panicked: %v", r)
		}
	}()
	NewPartialBanner(1, 2, []string{"x"}, nil).View()
}

// ─── AltText (INV-PROMPT-MVP-008, §19.3) ─────────────────────────────────────

func TestPartialBanner_AltText_StatusFirst(t *testing.T) {
	p := NewPartialBanner(3, 5, []string{"cross-repo"}, nil)
	alt := p.AltText()
	if !strings.HasPrefix(alt, "PARTIAL:") {
		t.Errorf("AltText() = %q, want status-first prefix PARTIAL: (UI.md §16.3)", alt)
	}
	if !strings.Contains(alt, "3 of 5") || !strings.Contains(alt, "cross-repo") {
		t.Errorf("AltText() = %q, want counts and missing list", alt)
	}
}
