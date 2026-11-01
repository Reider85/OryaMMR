// a11y_screens_test.go — accessibility-аудит трёх MVP-экранов (§10):
// фокус-видимость (INV-PROMPT-MVP-007), alt-text/парность символов
// (INV-PROMPT-MVP-008), рендер в high-contrast режиме (§19.2).
package screens

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// statusSymbols — символы статуса, которым обязана сопутствовать текстовая
// подпись (INV-PROMPT-MVP-008: цвет/символ никогда не единственный носитель).
const statusSymbols = "✓⚠✗◇•▸"

// assertSymbolTextPairing: каждая строка со статусным символом содержит буквы —
// символ всегда стоит рядом с текстовым объяснением.
func assertSymbolTextPairing(t *testing.T, label, view string) {
	t.Helper()
	for i, line := range strings.Split(view, "\n") {
		if !strings.ContainsAny(line, statusSymbols) {
			continue
		}
		hasText := strings.ContainsFunc(line, func(r rune) bool {
			return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		})
		if !hasText {
			t.Errorf("%s: line %d has status symbol without text: %q (INV-PROMPT-MVP-008)", label, i+1, line)
		}
	}
}

// TestFocusVisibility — INV-PROMPT-MVP-007: фокус всегда видим на каждом
// экране (§21 item 3, CI-автоматическая часть).
func TestFocusVisibility(t *testing.T) {
	t.Run("dashboard focused widget has marker and Tab moves it", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.StateMachine.SetState(ui.StateSuccess)

		view := d.View()
		if !ui.HasFocusMarker(view) {
			t.Fatal("dashboard success view has no focus marker (INV-PROMPT-MVP-007)")
		}
		if !strings.Contains(view, "▸ Today") {
			t.Error("initial focus must be on the Today widget")
		}
		if got := strings.Count(view, "▸ "); got != 1 {
			t.Errorf("expected exactly one focus marker, got %d", got)
		}

		_, _ = d.Update(tea.KeyMsg{Type: tea.KeyTab})
		if !strings.Contains(d.View(), "▸ Authorship Score") {
			t.Error("Tab must move the visible focus marker to Authorship Score")
		}
	})

	t.Run("adapters selected row has marker", func(t *testing.T) {
		a := newAdaptersFixture(t)
		view := a.View()
		if !ui.HasFocusMarker(view) {
			t.Fatal("adapters success view has no focus marker (INV-PROMPT-MVP-007)")
		}
		if !strings.Contains(view, "▸ opencode") {
			t.Error("selected adapter row must show the focus marker")
		}
		if got := strings.Count(view, "▸ "); got != 1 {
			t.Errorf("expected exactly one focus marker, got %d", got)
		}
	})

	t.Run("onboarding current step has marker", func(t *testing.T) {
		o := newOnboardingFixture(t, 1)
		o.StateMachine.SetState(ui.StateSuccess)
		view := o.View()
		if !ui.HasFocusMarker(view) {
			t.Fatal("onboarding view has no focus marker (INV-PROMPT-MVP-007)")
		}
		if !strings.Contains(view, "▸ Step 1/4") {
			t.Error("welcome step title must carry the focus marker")
		}
	})
}

// TestAltTextPresence — INV-PROMPT-MVP-008: alt-text/текстовые альтернативы
// на каждом экране (§21 item 5, CI-автоматическая часть).
func TestAltTextPresence(t *testing.T) {
	t.Run("dashboard", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.StateMachine.SetState(ui.StateSuccess)
		view := d.View()

		assertSymbolTextPairing(t, "dashboard success", view)

		// Sparkline-эквивалент (30-day тренд) обязан иметь текстовую строку рядом (§2.6).
		if !strings.Contains(view, "30-day:") {
			t.Error("trend indicator must have adjacent text line (§2.6 sparkline alt-text)")
		}
	})

	t.Run("adapters", func(t *testing.T) {
		a := newAdaptersFixture(t)
		assertSymbolTextPairing(t, "adapters success", a.View())

		// Тир-бейджи и capability-ячейки — текст, не только цвет (§2.8).
		view := a.View()
		for _, want := range []string{"T1", "T4", "native", "watcher", "universal"} {
			if !strings.Contains(view, want) {
				t.Errorf("capability cell missing text label %q (INV-UI-008)", want)
			}
		}
	})

	t.Run("onboarding", func(t *testing.T) {
		o := newOnboardingFixture(t, 2)
		o.StateMachine.SetState(ui.StateEmpty)
		assertSymbolTextPairing(t, "onboarding empty", o.View())
	})

	t.Run("error empty and partial states keep text alternatives", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.LoadError = errors.New("boom")
		d.StateMachine.SetState(ui.StateError)
		assertSymbolTextPairing(t, "dashboard error", d.View())

		d.StateMachine.SetState(ui.StateEmpty)
		assertSymbolTextPairing(t, "dashboard empty", d.View())

		d.StateMachine.SetState(ui.StatePartial)
		assertSymbolTextPairing(t, "dashboard partial", d.View())
	})
}

// TestHighContrastRender — §19.2: каждый экран рендерится в high-contrast
// режиме (WCAG-AAA) без потери структуры (§21 item 8).
func TestHighContrastRender(t *testing.T) {
	allStates := []ui.State{ui.StateLoading, ui.StateSuccess, ui.StateError, ui.StateEmpty, ui.StatePartial}

	enableHC := func(a11y *ui.A11y, theme **ui.Theme, sb interface{ SetFocus(string) }) {
		a11y.Config.HighContrast = true
		*theme = a11y.GetTheme() // новый HC-темплейт, исходная тема не мутирует
	}

	t.Run("dashboard", func(t *testing.T) {
		d := newDashboardFixture(t)
		enableHC(d.A11y, &d.Theme, d.StatusBar)
		d.StatusBar.Theme = d.Theme
		d.LoadError = errors.New("boom")

		for _, st := range allStates {
			d.StateMachine.SetState(st)
			view := d.View()
			if view == "" {
				t.Errorf("state %v: empty view in high-contrast", st)
				continue
			}
			if !strings.Contains(view, "CodeAuthorityFirst") {
				t.Errorf("state %v: header lost in high-contrast (§19.2 layout)", st)
			}
		}

		// Фокус не должен исчезать в high-contrast (INV-PROMPT-MVP-007).
		d.StateMachine.SetState(ui.StateSuccess)
		if !ui.HasFocusMarker(d.View()) {
			t.Error("focus marker lost in high-contrast mode")
		}
	})

	t.Run("adapters", func(t *testing.T) {
		a := newAdaptersFixture(t)
		enableHC(a.A11y, &a.Theme, a.StatusBar)
		a.StatusBar.Theme = a.Theme
		a.LoadError = errors.New("boom")

		for _, st := range allStates {
			a.StateMachine.SetState(st)
			view := a.View()
			if view == "" {
				t.Errorf("state %v: empty view in high-contrast", st)
				continue
			}
			if !strings.Contains(view, "CodeAuthorityFirst") {
				t.Errorf("state %v: header lost in high-contrast (§19.2 layout)", st)
			}
		}
	})

	t.Run("onboarding", func(t *testing.T) {
		o := newOnboardingFixture(t, 2)
		enableHC(o.A11y, &o.Theme, o.StatusBar)
		o.StatusBar.Theme = o.Theme
		o.DetectionError = errors.New("boom")
		o.InstallSuccess = []string{"opencode"}
		o.InstallErrors = map[string]error{"universal-lsp": errors.New("failed")}

		for _, st := range allStates {
			o.StateMachine.SetState(st)
			view := o.View()
			if view == "" {
				t.Errorf("state %v: empty view in high-contrast", st)
				continue
			}
			if !strings.Contains(view, "CodeAuthorityFirst") {
				t.Errorf("state %v: header lost in high-contrast (§19.2 layout)", st)
			}
		}
	})
}
