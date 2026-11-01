// adapters_test.go — тесты TUI-менеджера адаптеров: 5 golden-состояний,
// capability matrix, detail, клавиатура, message-driven переходы (§10, §16, §7).
package screens

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// stubAdapter — минимальный caf.Adapter для рендера таблицы (легаси-контракт
// 5 методов, ADR-009). Capabilities() возвращает валидную матрицу T1.
type stubAdapter struct{ id string }

func (s stubAdapter) ID() string { return s.id }

func (s stubAdapter) Capabilities() caf.CapabilityMatrix {
	return caf.CapabilityMatrix{
		PreSession: caf.TierNative,
		PostEdit:   caf.TierNative,
		PreCommit:  caf.TierNative,
		MCP:        true,
		Tier:       1,
	}
}

func (s stubAdapter) Install() error   { return nil }
func (s stubAdapter) Uninstall() error { return nil }
func (s stubAdapter) Detect() bool     { return true }

// newAdaptersFixture собирает экран с тремя адаптерами в success-состоянии.
func newAdaptersFixture(tb testing.TB) *AdaptersScreen {
	tb.Helper()
	a := NewAdaptersScreen(tb.TempDir(), "0.4.1-test")
	a.Adapters = []caf.Adapter{
		stubAdapter{id: "opencode"},
		stubAdapter{id: "claude-code"},
		stubAdapter{id: "universal-lsp"},
	}
	a.Installed = []string{"opencode"}
	a.Available = []string{"claude-code", "universal-lsp"}
	a.StateMachine.SetState(ui.StateSuccess)
	return a
}

// TestAdapters5StateSnapshots — 5 golden-снапшотов (§10, INV-PROMPT-MVP-006).
func TestAdapters5StateSnapshots(t *testing.T) {
	states := []struct {
		name  string
		state ui.State
		setup func(*AdaptersScreen)
	}{
		{"adapters_loading", ui.StateLoading, nil},
		{"adapters_success", ui.StateSuccess, nil},
		{"adapters_error", ui.StateError, func(a *AdaptersScreen) {
			a.LoadError = errors.New("cannot read .caf/adapters: corrupted YAML")
		}},
		{"adapters_empty", ui.StateEmpty, nil},
		{"adapters_partial", ui.StatePartial, func(a *AdaptersScreen) {
			a.Failed = []string{"universal-lsp"}
		}},
	}

	for _, tc := range states {
		t.Run(tc.name, func(t *testing.T) {
			a := newAdaptersFixture(t)
			if tc.setup != nil {
				tc.setup(a)
			}
			a.StateMachine.SetState(tc.state)
			assertGolden(t, tc.name, a.View())
		})
	}
}

// TestAdaptersStateContent — содержимое веток рендера (§16.1).
func TestAdaptersStateContent(t *testing.T) {
	t.Run("success renders capability matrix with focus marker", func(t *testing.T) {
		a := newAdaptersFixture(t)
		view := a.View()
		for _, want := range []string{"capability matrix", "T1", "native", "▸ opencode"} {
			if !strings.Contains(view, want) {
				t.Errorf("success view missing %q", want)
			}
		}
		if !strings.Contains(view, "T4") {
			t.Error("universal-lsp must render its T4 badge (coverage map, §2.8)")
		}
	})

	t.Run("error shows message and retry CTA", func(t *testing.T) {
		a := newAdaptersFixture(t)
		a.LoadError = errors.New("adapters dir unreadable")
		a.StateMachine.SetState(ui.StateError)
		view := a.View()
		for _, want := range []string{"adapters dir unreadable", "[r] retry", "[Esc] dismiss"} {
			if !strings.Contains(view, want) {
				t.Errorf("error view missing %q", want)
			}
		}
	})

	t.Run("empty shows install CTA", func(t *testing.T) {
		a := newAdaptersFixture(t)
		a.StateMachine.SetState(ui.StateEmpty)
		view := a.View()
		for _, want := range []string{"No adapters", "[i]"} {
			if !strings.Contains(view, want) {
				t.Errorf("empty view missing %q", want)
			}
		}
	})

	t.Run("partial shows banner over matrix", func(t *testing.T) {
		a := newAdaptersFixture(t)
		a.Failed = []string{"universal-lsp"} // реальный сбой → баннер, не хардкод
		a.StateMachine.SetState(ui.StatePartial)
		view := a.View()
		for _, want := range []string{"Partial data: 2 of 3", "Missing: universal-lsp", "capability matrix"} {
			if !strings.Contains(view, want) {
				t.Errorf("partial view missing %q", want)
			}
		}
	})
}

// TestAdaptersDataLoadedMsg: DataLoaded → Success (§16.2).
func TestAdaptersDataLoadedMsg(t *testing.T) {
	a := NewAdaptersScreen(t.TempDir(), "0.4.1-test")
	_, cmd := a.Update(AdaptersDataLoaded{
		Adapters:  []caf.Adapter{stubAdapter{id: "opencode"}},
		Installed: []string{"opencode"},
		Available: nil,
		LoadTime:  time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC),
	})
	if cmd != nil {
		t.Error("DataLoaded must not spawn follow-up commands")
	}
	if got := a.StateMachine.Current(); got != ui.StateSuccess {
		t.Fatalf("state = %v, want success", got)
	}
	if len(a.Adapters) != 1 || len(a.Installed) != 1 {
		t.Errorf("adapters = %d, installed = %d; want 1/1", len(a.Adapters), len(a.Installed))
	}
	if a.Loading {
		t.Error("Loading must be false after DataLoaded")
	}
}

// TestAdaptersKeys — клавиатура (§18.4, INV-UI-009), реалистичное кодирование
// печатных клавиш (KeyRunes — см. runeOf).
func TestAdaptersKeys(t *testing.T) {
	t.Run("Up/Down navigate bounded", func(t *testing.T) {
		a := newAdaptersFixture(t)

		_, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})
		_, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})
		if a.SelectedAdapter != 2 {
			t.Fatalf("SelectedAdapter = %d, want 2", a.SelectedAdapter)
		}
		_, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown}) // за нижней границей
		if a.SelectedAdapter != 2 {
			t.Errorf("Down at bottom: SelectedAdapter = %d, want 2", a.SelectedAdapter)
		}
		_, _ = a.Update(tea.KeyMsg{Type: tea.KeyUp})
		if a.SelectedAdapter != 1 {
			t.Errorf("SelectedAdapter = %d, want 1", a.SelectedAdapter)
		}
		_, _ = a.Update(tea.KeyMsg{Type: tea.KeyUp})
		_, _ = a.Update(tea.KeyMsg{Type: tea.KeyUp}) // за верхней границей
		if a.SelectedAdapter != 0 {
			t.Errorf("Up at top: SelectedAdapter = %d, want 0", a.SelectedAdapter)
		}
	})

	t.Run("Enter opens detail, Esc closes", func(t *testing.T) {
		a := newAdaptersFixture(t)

		_, _ = a.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if !a.ShowDetail || a.DetailAdapter == nil {
			t.Fatal("Enter must open the detail panel")
		}
		view := a.View()
		if !strings.Contains(view, "Adapter Details: opencode") {
			t.Errorf("detail view missing adapter title, got:\n%s", view)
		}
		if !strings.Contains(view, "[Esc] to return") {
			t.Error("detail view must show Esc hint (no dead-ends, INV-UI-010)")
		}

		_, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if a.ShowDetail || a.DetailAdapter != nil {
			t.Error("Esc must close the detail panel")
		}
	})

	t.Run("Enter with empty adapter list does not panic", func(t *testing.T) {
		a := NewAdaptersScreen(t.TempDir(), "0.4.1-test") // Adapters == nil
		a.StateMachine.SetState(ui.StateSuccess)

		_, _ = a.Update(tea.KeyMsg{Type: tea.KeyEnter}) // раньше: паника index out of range
		if a.ShowDetail {
			t.Error("Enter on empty list must not open detail")
		}
	})

	t.Run("i and c spawn install commands", func(t *testing.T) {
		a := newAdaptersFixture(t)
		if cmd := keyRune(a, 'i'); cmd == nil {
			t.Error("i must return install-opencode command")
		}
		if cmd := keyRune(a, 'c'); cmd == nil {
			t.Error("c must return install-claude-code command")
		}
	})

	t.Run("r triggers refresh into Loading", func(t *testing.T) {
		a := newAdaptersFixture(t)
		cmd := keyRune(a, 'r')
		if cmd == nil {
			t.Fatal("r must return a data-loading command")
		}
		if got := a.StateMachine.Current(); got != ui.StateLoading {
			t.Errorf("state after r = %v, want loading (§16.4)", got)
		}
	})

	t.Run("? opens help overlay with all keybindings (§18.4)", func(t *testing.T) {
		a := newAdaptersFixture(t)
		if cmd := keyRune(a, '?'); cmd != nil {
			t.Error("help is an in-screen overlay — ? must not spawn a command")
		}
		if !a.ShowHelp {
			t.Fatal("? must open the help overlay")
		}
		view := a.View()
		for _, want := range []string{"? Help — adapters", "Navigation:", "Actions:", "Screen-specific:", "[?] or [Esc] close"} {
			if !strings.Contains(view, want) {
				t.Errorf("help overlay missing %q (INV-UI-010)", want)
			}
		}
		// Живые клавиши экрана обязаны быть в help — help не врёт (INV-UI-009).
		for _, key := range []string{"Enter", "r", "q", "i", "c", "a"} {
			if !strings.Contains(view, key) {
				t.Errorf("help overlay missing keybinding %q", key)
			}
		}

		_, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if a.ShowHelp {
			t.Error("Esc must close the help overlay (no dead-end, INV-UI-010)")
		}
	})

	t.Run("q quits (statusbar promises [q] quit, §18.4)", func(t *testing.T) {
		a := newAdaptersFixture(t)
		if _, ok := keyRune(a, 'q')().(tea.QuitMsg); !ok {
			t.Error("q must produce tea.QuitMsg")
		}
	})

	t.Run("Esc on list quits", func(t *testing.T) {
		a := newAdaptersFixture(t)
		_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if cmd == nil {
			t.Fatal("Esc must return a quit command")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Error("Esc must produce tea.QuitMsg")
		}
	})
}

// TestAdaptersInstallDetectMsgs — InstallSuccess/InstallError/AutoDetectComplete
// теперь реально обрабатываются: раньше они терялись в Update, [i]/[c]/[a] были
// мёртвыми клавишами без обратной связи (INV-UI-009, INV-UI-010, §16.3).
func TestAdaptersInstallDetectMsgs(t *testing.T) {
	t.Run("InstallSuccess moves adapter to Installed and returns Success", func(t *testing.T) {
		a := newAdaptersFixture(t) // opencode уже в Installed, claude-code/universal-lsp в Available
		_, _ = a.Update(InstallSuccess{AdapterID: "claude-code"})

		if got := a.StateMachine.Current(); got != ui.StateSuccess {
			t.Fatalf("state = %v, want success", got)
		}
		if !containsString(a.Installed, "claude-code") {
			t.Errorf("Installed = %v, want claude-code present", a.Installed)
		}
		if containsString(a.Available, "claude-code") {
			t.Errorf("Available = %v, claude-code must be removed", a.Available)
		}
	})

	t.Run("InstallError surfaces Partial banner with the failed adapter", func(t *testing.T) {
		a := newAdaptersFixture(t)
		_, _ = a.Update(InstallError{AdapterID: "claude-code", Err: errors.New("settings.json unreadable")})

		if got := a.StateMachine.Current(); got != ui.StatePartial {
			t.Fatalf("state = %v, want partial (install failure must be visible, INV-UI-010)", got)
		}
		if !containsString(a.Failed, "claude-code") {
			t.Errorf("Failed = %v, want claude-code", a.Failed)
		}
		view := a.View()
		for _, want := range []string{"Partial data", "Missing: claude-code", "[r] retry", "capability matrix"} {
			if !strings.Contains(view, want) {
				t.Errorf("partial view missing %q", want)
			}
		}
	})

	t.Run("retrying failed install clears Partial back to Success", func(t *testing.T) {
		a := newAdaptersFixture(t)
		_, _ = a.Update(InstallError{AdapterID: "universal-lsp", Err: errors.New("boom")})
		if got := a.StateMachine.Current(); got != ui.StatePartial {
			t.Fatalf("state after error = %v, want partial", got)
		}

		_, _ = a.Update(InstallSuccess{AdapterID: "universal-lsp"})
		if got := a.StateMachine.Current(); got != ui.StateSuccess {
			t.Errorf("state after retry success = %v, want success", got)
		}
		if len(a.Failed) != 0 {
			t.Errorf("Failed = %v, want empty after successful retry", a.Failed)
		}
	})

	t.Run("AutoDetectComplete with agents transitions to Success", func(t *testing.T) {
		a := newAdaptersFixture(t)
		_, _ = a.Update(AutoDetectComplete{Detected: []string{"opencode"}})

		if got := a.StateMachine.Current(); got != ui.StateSuccess {
			t.Fatalf("state = %v, want success", got)
		}
		if len(a.Installed) != 1 || a.Installed[0] != "opencode" {
			t.Errorf("Installed = %v, want [opencode]", a.Installed)
		}
		if !containsString(a.Available, "claude-code") || !containsString(a.Available, "universal-lsp") {
			t.Errorf("Available = %v, want undetected adapters", a.Available)
		}
	})

	t.Run("AutoDetectComplete with nothing detected transitions to Empty", func(t *testing.T) {
		a := newAdaptersFixture(t)
		_, _ = a.Update(AutoDetectComplete{Detected: nil})

		if got := a.StateMachine.Current(); got != ui.StateEmpty {
			t.Fatalf("state = %v, want empty (no agents detected, §2.8)", got)
		}
		view := a.View()
		for _, want := range []string{"No adapters", "[i]"} {
			if !strings.Contains(view, want) {
				t.Errorf("empty view missing %q", want)
			}
		}
	})
}
