// bench_test.go — бенчмарки производительности рендера (PROMPT-MVP §6, §10).
//
// §6 бюджеты (проверяются на шаге 15):
//   - first paint < 100ms (INV-UI-001);
//   - frame render < 16ms (INV-UI-002);
//   - keystroke < 50ms (INV-UI-003);
//   - переход состояния (state change) ≤ 100ms (INV-PROMPT-MVP-009).
//
// Замер = конструирование + View() для first paint, только View() для кадра,
// Update+View для keystroke, смена состояния + View() для transition. Данные
// фиксированные (см. фикстуры), чтобы бенчмарк мерил рендер, а не FS/время.
// Enforcement (>2x → fail) — в budget_test.go.
package screens

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// BenchmarkFirstPaint — конструирование экрана + первый View() (§6: <100ms).
func BenchmarkFirstPaint(b *testing.B) {
	b.Run("dashboard", func(b *testing.B) {
		dir := b.TempDir()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			d := NewDashboardScreen(dir, "0.4.1-test")
			d.StateMachine.SetState(ui.StateLoading)
			_ = d.View()
		}
	})

	b.Run("adapters", func(b *testing.B) {
		dir := b.TempDir()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			a := NewAdaptersScreen(dir, "0.4.1-test")
			a.StateMachine.SetState(ui.StateLoading)
			_ = a.View()
		}
	})

	b.Run("onboarding", func(b *testing.B) {
		dir := b.TempDir()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			o := NewOnboardingScreen(dir, "0.4.1-test")
			o.CurrentStep = 1
			o.StateMachine.SetState(ui.StateSuccess)
			_ = o.View()
		}
	})
}

// BenchmarkFrameRender — только View() на success-фикстуре (§6: <16ms,
// INV-UI-002). Меряет «кадр», без конструирования и без смены состояния.
func BenchmarkFrameRender(b *testing.B) {
	b.Run("dashboard", func(b *testing.B) {
		d := newDashboardFixture(b)
		d.StateMachine.SetState(ui.StateSuccess)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = d.View()
		}
	})

	b.Run("adapters", func(b *testing.B) {
		a := newAdaptersFixture(b)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = a.View()
		}
	})

	b.Run("onboarding", func(b *testing.B) {
		o := newOnboardingFixture(b, 2)
		o.StateMachine.SetState(ui.StateSuccess)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = o.View()
		}
	})
}

// BenchmarkKeystroke — Update(клавиша) + View() (§6: <50ms, INV-UI-003).
// Простые действия: Tab (смена фокуса), Down (навигация), ? (help overlay).
func BenchmarkKeystroke(b *testing.B) {
	b.Run("dashboard", func(b *testing.B) {
		d := newDashboardFixture(b)
		d.StateMachine.SetState(ui.StateSuccess)
		key := tea.KeyMsg{Type: tea.KeyTab}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = d.Update(key)
			_ = d.View()
		}
	})

	b.Run("adapters", func(b *testing.B) {
		a := newAdaptersFixture(b)
		key := tea.KeyMsg{Type: tea.KeyDown}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = a.Update(key)
			_ = a.View()
		}
	})

	b.Run("onboarding", func(b *testing.B) {
		o := newOnboardingFixture(b, 2)
		o.StateMachine.SetState(ui.StateSuccess)
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = o.Update(key)
			_ = o.View()
		}
	})
}

// BenchmarkStateTransition — смена состояния + перерисовка (§6: ≤100ms,
// INV-PROMPT-MVP-009). Чередуем loading↔success — два самых тяжёлых рендера.
func BenchmarkStateTransition(b *testing.B) {
	b.Run("dashboard", func(b *testing.B) {
		d := newDashboardFixture(b)
		d.LoadError = errors.New("bench")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i%2 == 0 {
				d.StateMachine.SetState(ui.StateLoading)
			} else {
				d.StateMachine.SetState(ui.StateSuccess)
			}
			_ = d.View()
		}
	})

	b.Run("adapters", func(b *testing.B) {
		a := newAdaptersFixture(b)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i%2 == 0 {
				a.StateMachine.SetState(ui.StateLoading)
			} else {
				a.StateMachine.SetState(ui.StateSuccess)
			}
			_ = a.View()
		}
	})

	b.Run("onboarding", func(b *testing.B) {
		o := newOnboardingFixture(b, 2)
		o.Loading = true
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i%2 == 0 {
				o.StateMachine.SetState(ui.StateLoading)
			} else {
				o.StateMachine.SetState(ui.StateSuccess)
			}
			_ = o.View()
		}
	})
}
