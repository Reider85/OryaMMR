// budget_test.go — enforcement бюджетов §6: падение только при >2x бюджета
// (§6:255 «If a benchmark exceeds budget by >2x, CI fails the PR»).
//
// Замер = медиана 10 итераций (устойчивость к шуму CI). Пороги:
//   - first paint  100ms → fail > 200ms (INV-UI-001);
//   - frame render  16ms → fail >  32ms (INV-UI-002);
//   - keystroke     50ms → fail > 100ms (INV-UI-003);
//   - transition   100ms → fail > 200ms (INV-PROMPT-MVP-009).
//
// Всё в одном пакете screens — фикстуры переиспользуются из *_test.go.
package screens

import (
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// medianDuration возвращает медиану дистуриций (без упорядочивания входа).
func medianDuration(ds []time.Duration) time.Duration {
	sorted := append([]time.Duration(nil), ds...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)/2]
}

// measureMedian замеряет fn n раз и возвращает медиану.
func measureMedian(n int, fn func()) time.Duration {
	ds := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		fn()
		ds = append(ds, time.Since(start))
	}
	return medianDuration(ds)
}

// enforceFails проверяет: медиана > 2x бюджета → t.Errorf (CI fail).
// within 1x..2x — warn (бюджет превышен, но >2x порог не достигнут).
// Всё логируется — отчёт по §6 (шаг 15) собирается из -v вывода.
func enforceFails(t *testing.T, op string, median, budget time.Duration) {
	t.Helper()
	warning, within := ui.EnforceBudget(op, median, budget)
	if within {
		t.Logf("✓ %s median=%v (budget %v, %.2fx)", op, median, budget, float64(median)/float64(budget))
		return
	}
	if warning.IsCritical() {
		t.Errorf("PERF BUDGET VIOLATION: %s median=%v exceeds 2x budget %v (>2x → CI fails, §6)",
			op, median, budget)
		return
	}
	t.Logf("⚠ %s median=%v exceeds budget %v (%.2fx) — within 2x, warning only",
		op, median, budget, warning.Ratio)
}

// TestBudgetFirstPaint — конструирование + первый View() <100ms (INV-UI-001).
func TestBudgetFirstPaint(t *testing.T) {
	budget := ui.NewPerformanceBudget().FirstPaint

	t.Run("dashboard", func(t *testing.T) {
		median := measureMedian(10, func() {
			d := NewDashboardScreen(t.TempDir(), "0.4.1-test")
			d.StateMachine.SetState(ui.StateLoading)
			_ = d.View()
		})
		enforceFails(t, "first_paint/dashboard", median, budget)
	})

	t.Run("adapters", func(t *testing.T) {
		median := measureMedian(10, func() {
			a := NewAdaptersScreen(t.TempDir(), "0.4.1-test")
			a.StateMachine.SetState(ui.StateLoading)
			_ = a.View()
		})
		enforceFails(t, "first_paint/adapters", median, budget)
	})

	t.Run("onboarding", func(t *testing.T) {
		median := measureMedian(10, func() {
			o := NewOnboardingScreen(t.TempDir(), "0.4.1-test")
			o.CurrentStep = 1
			o.StateMachine.SetState(ui.StateSuccess)
			_ = o.View()
		})
		enforceFails(t, "first_paint/onboarding", median, budget)
	})
}

// TestBudgetFrameRender — только View() на loaded-состоянии <16ms (INV-UI-002).
func TestBudgetFrameRender(t *testing.T) {
	budget := ui.NewPerformanceBudget().FrameRender

	t.Run("dashboard", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.StateMachine.SetState(ui.StateSuccess)
		median := measureMedian(10, func() { _ = d.View() })
		enforceFails(t, "frame_render/dashboard", median, budget)
	})

	t.Run("adapters", func(t *testing.T) {
		a := newAdaptersFixture(t)
		median := measureMedian(10, func() { _ = a.View() })
		enforceFails(t, "frame_render/adapters", median, budget)
	})

	t.Run("onboarding", func(t *testing.T) {
		o := newOnboardingFixture(t, 2)
		o.StateMachine.SetState(ui.StateSuccess)
		median := measureMedian(10, func() { _ = o.View() })
		enforceFails(t, "frame_render/onboarding", median, budget)
	})
}

// TestBudgetKeystroke — Update(клавиша) + View() <50ms (INV-UI-003).
func TestBudgetKeystroke(t *testing.T) {
	budget := ui.NewPerformanceBudget().Keystroke

	t.Run("dashboard", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.StateMachine.SetState(ui.StateSuccess)
		key := tea.KeyMsg{Type: tea.KeyTab}
		median := measureMedian(10, func() {
			_, _ = d.Update(key)
			_ = d.View()
		})
		enforceFails(t, "keystroke/dashboard", median, budget)
	})

	t.Run("adapters", func(t *testing.T) {
		a := newAdaptersFixture(t)
		key := tea.KeyMsg{Type: tea.KeyDown}
		median := measureMedian(10, func() {
			_, _ = a.Update(key)
			_ = a.View()
		})
		enforceFails(t, "keystroke/adapters", median, budget)
	})

	t.Run("onboarding", func(t *testing.T) {
		o := newOnboardingFixture(t, 2)
		o.StateMachine.SetState(ui.StateSuccess)
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		median := measureMedian(10, func() {
			_, _ = o.Update(key)
			_ = o.View()
		})
		enforceFails(t, "keystroke/onboarding", median, budget)
	})
}

// TestBudgetTransition — смена loading↔success + View() ≤100ms
// (INV-PROMPT-MVP-009).
func TestBudgetTransition(t *testing.T) {
	budget := ui.NewPerformanceBudget().FirstPaint // 100ms — тот же лимит

	t.Run("dashboard", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.LoadError = errors.New("budget")
		median := measureMedian(10, func() {
			if d.StateMachine.Current() == ui.StateLoading {
				d.StateMachine.SetState(ui.StateSuccess)
			} else {
				d.StateMachine.SetState(ui.StateLoading)
			}
			_ = d.View()
		})
		enforceFails(t, "transition/dashboard", median, budget)
	})

	t.Run("adapters", func(t *testing.T) {
		a := newAdaptersFixture(t)
		median := measureMedian(10, func() {
			if a.StateMachine.Current() == ui.StateLoading {
				a.StateMachine.SetState(ui.StateSuccess)
			} else {
				a.StateMachine.SetState(ui.StateLoading)
			}
			_ = a.View()
		})
		enforceFails(t, "transition/adapters", median, budget)
	})

	t.Run("onboarding", func(t *testing.T) {
		o := newOnboardingFixture(t, 2)
		o.Loading = true
		median := measureMedian(10, func() {
			if o.StateMachine.Current() == ui.StateLoading {
				o.StateMachine.SetState(ui.StateSuccess)
			} else {
				o.StateMachine.SetState(ui.StateLoading)
			}
			_ = o.View()
		})
		enforceFails(t, "transition/onboarding", median, budget)
	})
}
