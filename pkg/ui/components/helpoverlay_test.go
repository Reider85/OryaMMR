package components

import (
	"strings"
	"testing"
)

func testSections() []HelpSection {
	return []HelpSection{
		{Title: "Navigation", Entries: []HelpEntry{{Key: "Tab", Desc: "Cycle widgets"}}},
		{Title: "Actions", Entries: []HelpEntry{{Key: "r", Desc: "Refresh"}, {Key: "q", Desc: "Quit"}}},
		{Title: "Screen-specific", Entries: []HelpEntry{{Key: "i", Desc: "Install opencode"}}},
	}
}

// ─── View: обязательные части оверлея (§18.4, INV-UI-010) ────────────────────

func TestHelpOverlay_View_GroupedCategoriesAndCloseHint(t *testing.T) {
	h := NewHelpOverlay("dashboard", testSections(), nil)
	got := h.View()

	// (1) заголовок с именем экрана
	if !strings.Contains(got, "? Help — dashboard") {
		t.Errorf("View() missing screen title, got:\n%s", got)
	}
	// (2) все 3 категории §18.4
	for _, cat := range []string{"Navigation:", "Actions:", "Screen-specific:"} {
		if !strings.Contains(got, cat) {
			t.Errorf("View() missing category %q (§18.4 grouped keybindings):\n%s", cat, got)
		}
	}
	// каждая клавиша экрана присутствует
	for _, key := range []string{"Tab", "r", "q", "i"} {
		if !strings.Contains(got, key) {
			t.Errorf("View() missing keybinding %q", key)
		}
	}
	// (3) глоссарий (§19.5)
	for _, term := range []string{"ACS", "LSP", "MCP"} {
		if !strings.Contains(got, term) {
			t.Errorf("View() missing glossary term %q (§19.5 no jargon)", term)
		}
	}
	// (4) подсказка закрытия — нет тупика (INV-UI-010)
	if !strings.Contains(got, "[?] or [Esc] close") {
		t.Errorf("View() missing close hint (INV-UI-010), got:\n%s", got)
	}
}

func TestHelpOverlay_View_EmptySections_StillCloses(t *testing.T) {
	h := NewHelpOverlay("onboarding", nil, nil)
	got := h.View()
	if !strings.Contains(got, "? Help — onboarding") {
		t.Errorf("empty-sections overlay lost title:\n%s", got)
	}
	if !strings.Contains(got, "[?] or [Esc] close") {
		t.Error("empty-sections overlay must still show close hint (no dead-end, INV-UI-010)")
	}
}

func TestHelpOverlay_NilTheme_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewHelpOverlay(..., nil) panicked: %v", r)
		}
	}()
	NewHelpOverlay("adapters", testSections(), nil).View()
}

// ─── AltText (INV-PROMPT-MVP-008, §19.3) ─────────────────────────────────────

func TestHelpOverlay_AltText_StatusFirst(t *testing.T) {
	h := NewHelpOverlay("adapters", testSections(), nil)
	alt := h.AltText()
	if !strings.HasPrefix(alt, "HELP:") {
		t.Errorf("AltText() = %q, want status-first prefix HELP: (UI.md §16.3)", alt)
	}
	for _, part := range []string{"adapters", "Refresh", "Install opencode", "Press ? or Esc to close"} {
		if !strings.Contains(alt, part) {
			t.Errorf("AltText() = %q, want part %q", alt, part)
		}
	}
}
