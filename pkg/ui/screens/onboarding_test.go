// onboarding_test.go — тесты онбординг-визарда: 5 golden-состояний,
// message-driven переходы (Detection/Installation/Summary), клавиатура (§10, §16, §2.7).
package screens

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// newOnboardingFixture: визард с фиксированным шагом — без вызова StartStep
// (он возвращает cmd с time.Sleep(2-3s)); cmd из тестов не исполняются.
func newOnboardingFixture(tb testing.TB, step int) *OnboardingScreen {
	tb.Helper()
	o := NewOnboardingScreen(tb.TempDir(), "0.4.1-test")
	o.CurrentStep = step
	o.Loading = false
	return o
}

// TestOnboarding5StateSnapshots — 5 golden-снапшотов (§10, INV-PROMPT-MVP-006).
// Empty/Partial для онбординга формально N/A в §16.5, но INV-PROMPT-MVP-006
// (critical) требует все 5 состояний: empty = детекция без агентов (§2.7),
// partial = часть адаптеров не встала (§16.3).
func TestOnboarding5StateSnapshots(t *testing.T) {
	states := []struct {
		name  string
		state ui.State
		step  int
		setup func(*OnboardingScreen)
	}{
		{"onboarding_loading", ui.StateLoading, 2, func(o *OnboardingScreen) {
			o.Loading = true
			o.DetectionSpinner.Start()
		}},
		{"onboarding_success", ui.StateSuccess, 1, nil},
		{"onboarding_error", ui.StateError, 2, func(o *OnboardingScreen) {
			o.DetectionError = errors.New("detection command failed")
		}},
		{"onboarding_empty", ui.StateEmpty, 2, nil},
		{"onboarding_partial", ui.StatePartial, 3, func(o *OnboardingScreen) {
			o.InstallSuccess = []string{"opencode"}
			o.InstallErrors = map[string]error{"universal-lsp": errors.New("installation failed")}
		}},
	}

	for _, tc := range states {
		t.Run(tc.name, func(t *testing.T) {
			o := newOnboardingFixture(t, tc.step)
			if tc.setup != nil {
				tc.setup(o)
			}
			o.StateMachine.SetState(tc.state)
			assertGolden(t, tc.name, o.View())
		})
	}
}

// TestOnboardingStateRouting — state-машина перекрывает пошаговый рендер (§16.2).
func TestOnboardingStateRouting(t *testing.T) {
	t.Run("error overrides step view", func(t *testing.T) {
		o := newOnboardingFixture(t, 2)
		o.DetectionError = errors.New("detection command failed")
		o.StateMachine.SetState(ui.StateError)
		view := o.View()
		for _, want := range []string{"Onboarding failed", "detection command failed", "[r] retry"} {
			if !strings.Contains(view, want) {
				t.Errorf("error view missing %q", want)
			}
		}
		if strings.Contains(view, "Detecting agents") {
			t.Error("error state must not show step content underneath")
		}
	})

	t.Run("empty shows recovery options", func(t *testing.T) {
		o := newOnboardingFixture(t, 2)
		o.StateMachine.SetState(ui.StateEmpty)
		view := o.View()
		for _, want := range []string{"No agents detected", "[i]", "[Enter]"} {
			if !strings.Contains(view, want) {
				t.Errorf("empty view missing %q (§2.7 error recovery)", want)
			}
		}
	})

	t.Run("partial shows banner plus install results", func(t *testing.T) {
		o := newOnboardingFixture(t, 3)
		o.InstallSuccess = []string{"opencode"}
		o.InstallErrors = map[string]error{"universal-lsp": errors.New("installation failed")}
		o.StateMachine.SetState(ui.StatePartial)
		view := o.View()
		for _, want := range []string{"Partial data: 1 of 2", "Missing: universal-lsp", "✓ opencode installed", "✗ universal-lsp failed"} {
			if !strings.Contains(view, want) {
				t.Errorf("partial view missing %q", want)
			}
		}
	})

	t.Run("step views keep focus marker", func(t *testing.T) {
		for step := 1; step <= 4; step++ {
			o := newOnboardingFixture(t, step)
			o.StateMachine.SetState(ui.StateSuccess)
			view := o.View()
			if !strings.Contains(view, "▸ Step") {
				t.Errorf("step %d: no focus marker on step title (INV-PROMPT-MVP-007)", step)
			}
			if !strings.Contains(view, "CodeAuthorityFirst") {
				t.Errorf("step %d: header missing (§15.2)", step)
			}
		}
	})
}

// TestOnboardingDetectionMsgs: 0 агентов → Empty, есть → Success (§2.7, §16.2).
func TestOnboardingDetectionMsgs(t *testing.T) {
	t.Run("no agents detected transitions to Empty", func(t *testing.T) {
		o := newOnboardingFixture(t, 2)
		o.Loading = true
		o.DetectionSpinner.Start()

		_, _ = o.Update(DetectionComplete{Detected: nil})

		if got := o.StateMachine.Current(); got != ui.StateEmpty {
			t.Fatalf("state = %v, want empty (0 detected agents)", got)
		}
		if o.Loading {
			t.Error("Loading must be false after detection completes")
		}
		if o.DetectionSpinner.IsAnimating() {
			t.Error("spinner must stop after detection")
		}
	})

	t.Run("agents detected transitions to Success", func(t *testing.T) {
		o := newOnboardingFixture(t, 2)
		_, _ = o.Update(DetectionComplete{Detected: []string{"opencode", "claude-code"}})

		if got := o.StateMachine.Current(); got != ui.StateSuccess {
			t.Fatalf("state = %v, want success", got)
		}
		if len(o.DetectedAdapters) != 2 {
			t.Errorf("DetectedAdapters = %v, want 2 entries", o.DetectedAdapters)
		}
	})
}

// TestOnboardingInstallationMsgs: частичный сбой → Partial; без сбоев → Success.
// Регресс: раньше InstallSuccess дублировался (msg.Success + повторный append).
func TestOnboardingInstallationMsgs(t *testing.T) {
	t.Run("partial failure transitions to Partial without duplicates", func(t *testing.T) {
		o := newOnboardingFixture(t, 3)
		_, _ = o.Update(InstallationComplete{
			Success: []string{"opencode"},
			Errors:  map[string]error{"universal-lsp": errors.New("installation failed")},
		})

		if got := o.StateMachine.Current(); got != ui.StatePartial {
			t.Fatalf("state = %v, want partial", got)
		}
		if len(o.InstallSuccess) != 1 || o.InstallSuccess[0] != "opencode" {
			t.Errorf("InstallSuccess = %v, want exactly [opencode] (dup-append regression)", o.InstallSuccess)
		}
	})

	t.Run("all installed transitions to Success", func(t *testing.T) {
		o := newOnboardingFixture(t, 3)
		_, _ = o.Update(InstallationComplete{
			Success: []string{"opencode", "claude-code"},
			Errors:  map[string]error{},
		})
		if got := o.StateMachine.Current(); got != ui.StateSuccess {
			t.Fatalf("state = %v, want success", got)
		}
	})
}

// TestOnboardingSummaryMsg: SummaryComplete → Success + CTA следующего шага (§2.7).
func TestOnboardingSummaryMsg(t *testing.T) {
	o := newOnboardingFixture(t, 4)
	_, _ = o.Update(SummaryComplete{Message: "Setup complete. 2 adapters installed."})

	if got := o.StateMachine.Current(); got != ui.StateSuccess {
		t.Fatalf("state = %v, want success", got)
	}
	if !o.SummaryComplete || o.SummaryMessage == "" {
		t.Fatal("summary fields must be stored")
	}
	view := o.View()
	for _, want := range []string{"Setup complete. 2 adapters installed.", "caf dashboard"} {
		if !strings.Contains(view, want) {
			t.Errorf("summary view missing %q (CTA of next step, §2.7)", want)
		}
	}
}

// TestOnboardingKeys — клавиатура шагов (§18.4, INV-UI-009).
func TestOnboardingKeys(t *testing.T) {
	t.Run("Enter advances step", func(t *testing.T) {
		o := newOnboardingFixture(t, 1)
		_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Fatal("Enter on step 1 must return step-2 detection command")
		}
		if o.CurrentStep != 2 {
			t.Errorf("CurrentStep = %d, want 2", o.CurrentStep)
		}
		if got := o.StateMachine.Current(); got != ui.StateLoading {
			t.Errorf("state = %v, want loading (detection in flight, §16.5)", got)
		}
	})

	t.Run("Esc goes back then quits on welcome", func(t *testing.T) {
		o := newOnboardingFixture(t, 3)
		_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if cmd == nil {
			t.Fatal("Esc on step 3 must return step-2 command")
		}
		if o.CurrentStep != 2 {
			t.Errorf("CurrentStep = %d, want 2", o.CurrentStep)
		}

		// Esc на первом шаге — выход (нет тупика, INV-UI-010)
		o1 := newOnboardingFixture(t, 1)
		_, quit := o1.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if quit == nil {
			t.Fatal("Esc on welcome must quit")
		}
		if _, ok := quit().(tea.QuitMsg); !ok {
			t.Error("Esc on welcome must produce tea.QuitMsg")
		}
	})

	t.Run("q and Ctrl+C quit", func(t *testing.T) {
		o := newOnboardingFixture(t, 1)
		if _, ok := keyRune(o, 'q')().(tea.QuitMsg); !ok {
			t.Error("q must produce tea.QuitMsg")
		}
		_, ctrlC := o.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if ctrlC == nil {
			t.Fatal("Ctrl+C must return a quit command")
		}
		if _, ok := ctrlC().(tea.QuitMsg); !ok {
			t.Error("Ctrl+C must produce tea.QuitMsg")
		}
	})

	t.Run("? returns help command", func(t *testing.T) {
		o := newOnboardingFixture(t, 1)
		if cmd := keyRune(o, '?'); cmd != nil {
			t.Error("? currently stubs help overlay with nil cmd (§18.4 overlay — step 16)")
		}
	})
}
