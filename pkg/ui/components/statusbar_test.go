// statusbar_test.go — тесты нижней строки статуса (§15.2 rule 7, §2.6 refresh indicator).
package components

import (
	"strings"
	"testing"
)

func TestStatusBarBasics(t *testing.T) {
	sb := NewStatusBar("dashboard")
	sb.SetKeyHints([]string{"[Tab] widgets", "[r] refresh"})
	sb.SetFocus("today")

	got := sb.Render()
	if !strings.Contains(got, "dashboard focus: today") {
		t.Errorf("status bar must show screen and focus, got %q", got)
	}
	for _, hint := range []string{"[Tab] widgets", "[r] refresh"} {
		if !strings.Contains(got, hint) {
			t.Errorf("status bar missing key hint %q", hint)
		}
	}
}

func TestStatusBarRefreshIndicators(t *testing.T) {
	t.Run("idle bar has no refresh stamp", func(t *testing.T) {
		sb := NewStatusBar("dashboard")
		sb.SetRefreshCompleted() // LastRefresh ещё не установлен
		if strings.Contains(sb.Render(), "refreshed at") {
			t.Error("idle status bar must not show refresh timestamp")
		}
	})

	t.Run("refreshing state is visible (§2.6 never refresh silently)", func(t *testing.T) {
		sb := NewStatusBar("dashboard")
		sb.SetRefreshState("⟳ refreshing…")
		if !strings.Contains(sb.Render(), "⟳ refreshing…") {
			t.Error("refreshing indicator must render")
		}

		sb.SetRefreshCompleted()
		if !strings.Contains(sb.Render(), "✓ refreshed at") {
			t.Errorf("after refresh completion bar must show stamp, got %q", sb.Render())
		}
	})

	t.Run("SetRefreshIndicator formats and clears", func(t *testing.T) {
		sb := NewStatusBar("adapters")
		sb.SetRefreshIndicator("loading %s…", "adapters")
		if !strings.Contains(sb.Render(), "loading adapters…") {
			t.Errorf("formatted indicator missing, got %q", sb.Render())
		}
		sb.SetRefreshIndicator("")
		if strings.Contains(sb.Render(), "loading adapters") {
			t.Error("empty format must clear the indicator")
		}
	})
}

func TestStatusBarLongContentDoesNotPanic(t *testing.T) {
	sb := NewStatusBar("adapters")
	sb.SetFocus(strings.Repeat("very-long-focus-name", 10))
	sb.SetKeyHints([]string{"[i] install", "[c] claude", "[a] auto-detect", "[Enter] detail", "[r] refresh", "[?] help", "[q] quit"})

	if got := sb.Render(); !strings.Contains(got, "[q] quit") {
		t.Errorf("hints must render even when padding clamps to 0, got %q", got)
	}
}
