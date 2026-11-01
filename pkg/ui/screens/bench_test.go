// bench_test.go — бенчмарки производительности рендера (PROMPT-MVP §6, §10).
//
// §6 бюджеты (проверяются на шаге 15):
//   - first paint < 100ms;
//   - переход состояния (state change) ≤ 100ms (INV-PROMPT-MVP-009).
//
// Замер = конструирование + View() для first paint и смена состояния + View()
// для transition. Данные фиксированные (см. фикстуры), чтобы бенчмарк мерил
// рендер, а не FS/время.
package screens

import (
	"errors"
	"testing"

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
