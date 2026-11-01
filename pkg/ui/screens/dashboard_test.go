// dashboard_test.go — тесты TUI-дашборда: 5 golden-состояний, фокус,
// клавиатура, message-driven переходы (PROMPT-MVP §10, §16, §18).
package screens

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/storage"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// newDashboardFixture собирает дашборд с фиксированными данными: без
// time.Now() и без вызова updateWidgets (там зависимость от «сегодня») —
// вывод View() детерминирован (§10 snapshot stability).
func newDashboardFixture(tb testing.TB) *DashboardScreen {
	tb.Helper()
	d := NewDashboardScreen(tb.TempDir(), "0.4.1-test")

	d.TodayWidget.CardsDue = 5
	d.TodayWidget.PendingDecisions = 3
	d.TodayWidget.NewBlindspots = 1

	d.AuthorshipWidget.ACS = 3.8
	d.AuthorshipWidget.Recall = 0.72
	d.AuthorshipWidget.Coverage = 0.64
	d.AuthorshipWidget.Trend = "→"

	d.AgentsWidget.Agents = []AgentStatus{
		{Name: "opencode", ID: "opencode", Active: true, Health: "healthy"},
		{Name: "claude-code", ID: "claude-code", Active: true, Health: "degraded"},
	}

	d.DecisionsWidget.Decisions = []storage.Decision{
		{ID: "d1", Title: "Use token bucket for rate limiting", Timestamp: time.Date(2026, 10, 24, 11, 2, 14, 0, time.UTC)},
		{ID: "d2", Title: "Prefer YAML over SQLite", Timestamp: time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)},
	}

	d.CrossRepoWidget.Enabled = true
	d.CrossRepoWidget.Stale = true
	d.CrossRepoWidget.Repos = []string{"OryaSecurity"}
	d.CrossRepoWidget.LastSync = time.Date(2026, 10, 24, 8, 0, 0, 0, time.UTC)

	return d
}

// keyRune отправляет на экран реалистичный для bubbletea v0.24 нажатый символ
// (Type=KeyRunes + Runes) — не Type=символ.
func keyRune(s ui.Screen, r rune) tea.Cmd {
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return cmd
}

// TestDashboard5StateSnapshots — 5 golden-снапшотов (§10, INV-PROMPT-MVP-006).
// Отсутствующий файл валит тест → экран без любого из 5 состояний не проходит CI.
func TestDashboard5StateSnapshots(t *testing.T) {
	states := []struct {
		name  string
		state ui.State
		setup func(*DashboardScreen)
	}{
		{"dashboard_loading", ui.StateLoading, nil},
		{"dashboard_success", ui.StateSuccess, nil},
		{"dashboard_error", ui.StateError, func(d *DashboardScreen) {
			d.LoadError = errors.New("permission denied reading .caf/sessions")
		}},
		{"dashboard_empty", ui.StateEmpty, nil},
		{"dashboard_partial", ui.StatePartial, nil},
	}

	for _, tc := range states {
		t.Run(tc.name, func(t *testing.T) {
			d := newDashboardFixture(t)
			if tc.setup != nil {
				tc.setup(d)
			}
			d.StateMachine.SetState(tc.state)
			assertGolden(t, tc.name, d.View())
		})
	}
}

// TestDashboardStateContent проверяет содержимое каждого состояния без
// привязки к точной разметке (быстрый сигнал, что ветка рендера та самая).
func TestDashboardStateContent(t *testing.T) {
	t.Run("loading shows skeleton placeholders", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.StateMachine.SetState(ui.StateLoading)
		if !strings.Contains(d.View(), "┌") {
			t.Error("loading state must render skeleton widget frames")
		}
	})

	t.Run("error shows message and retry CTA", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.LoadError = errors.New("permission denied")
		d.StateMachine.SetState(ui.StateError)
		view := d.View()
		for _, want := range []string{"permission denied", "[r] retry", "[Esc] dismiss"} {
			if !strings.Contains(view, want) {
				t.Errorf("error view missing %q", want)
			}
		}
	})

	t.Run("empty shows CTA", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.StateMachine.SetState(ui.StateEmpty)
		view := d.View()
		for _, want := range []string{"No sessions found", "caf init"} {
			if !strings.Contains(view, want) {
				t.Errorf("empty view missing CTA text %q", want)
			}
		}
	})

	t.Run("partial shows banner over real data", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.StateMachine.SetState(ui.StatePartial)
		view := d.View()
		for _, want := range []string{"Partial data: 1 of 2", "Missing: cross-repo", "[r] retry"} {
			if !strings.Contains(view, want) {
				t.Errorf("partial view missing %q", want)
			}
		}
		if !strings.Contains(view, "ACS") {
			t.Error("partial banner must sit over real data, not replace it (§16.3)")
		}
	})
}

// TestDashboardDataLoadedMsg: DataLoaded → Success + данные в виджетах (§16.2).
func TestDashboardDataLoadedMsg(t *testing.T) {
	d := NewDashboardScreen(t.TempDir(), "0.4.1-test")
	loadTime := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	sessions := []*storage.Session{
		{
			SessionID:    "2026-10-24-a1b2c3",
			StartedAt:    time.Date(2001, 1, 1, 10, 0, 0, 0, time.UTC), // не «сегодня» → детерминировано
			FilesTouched: []string{"internal/api/policies.go"},
			DecisionsMade: []storage.Decision{
				{ID: "d1", Title: "Add rate limiter", Timestamp: time.Date(2001, 1, 1, 10, 30, 0, 0, time.UTC)},
			},
		},
	}

	screen, cmd := d.Update(DashboardDataLoaded{Sessions: sessions, LoadTime: loadTime})
	if screen == nil {
		t.Fatal("Update returned nil screen")
	}
	if cmd != nil {
		t.Error("DataLoaded must not spawn follow-up commands")
	}
	if got := d.StateMachine.Current(); got != ui.StateSuccess {
		t.Fatalf("state = %v, want success", got)
	}
	if len(d.Sessions) != 1 {
		t.Errorf("Sessions = %d, want 1", len(d.Sessions))
	}
	if !d.RefreshTime.Equal(loadTime) {
		t.Errorf("RefreshTime = %v, want %v", d.RefreshTime, loadTime)
	}
	// 1 файл / 1 сессия → ACS = 1 * 0.1 (наивный прокси MVP)
	if d.AuthorshipWidget.ACS != 0.1 {
		t.Errorf("ACS = %v, want 0.1", d.AuthorshipWidget.ACS)
	}
	if len(d.DecisionsWidget.Decisions) != 1 {
		t.Errorf("Decisions = %d, want 1", len(d.DecisionsWidget.Decisions))
	}
}

// TestDashboardLoadErrorMsg: ошибка загрузки → Error state + ErrorBox (§16.2).
func TestDashboardLoadErrorMsg(t *testing.T) {
	d := NewDashboardScreen(t.TempDir(), "0.4.1-test")
	_, _ = d.Update(DashboardLoadError{Err: errors.New("sessions dir unreadable")})

	if got := d.StateMachine.Current(); got != ui.StateError {
		t.Fatalf("state = %v, want error", got)
	}
	view := d.View()
	if !strings.Contains(view, "sessions dir unreadable") {
		t.Error("error view must show the load error message")
	}
	if d.LoadError == nil {
		t.Error("LoadError must be stored")
	}
}

// TestDashboardKeys — полнота клавиатуры (§18.4, INV-UI-009) с реалистичным
// кодированием печатных клавиш (KeyRunes — см. runeOf).
func TestDashboardKeys(t *testing.T) {
	t.Run("Tab cycles widgets with wrap", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.StateMachine.SetState(ui.StateSuccess)

		wantOrder := []int{1, 2, 3, 4, 0} // wrap после 4-го Tab
		for i, want := range wantOrder {
			_, _ = d.Update(tea.KeyMsg{Type: tea.KeyTab})
			if d.CurrentWidget != want {
				t.Fatalf("Tab #%d: CurrentWidget = %d, want %d", i+1, d.CurrentWidget, want)
			}
		}

		_, _ = d.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
		if d.CurrentWidget != 4 {
			t.Errorf("ShiftTab from 0: CurrentWidget = %d, want 4 (wrap back)", d.CurrentWidget)
		}
	})

	t.Run("r triggers refresh into Loading", func(t *testing.T) {
		d := newDashboardFixture(t)
		d.StateMachine.SetState(ui.StateSuccess)

		cmd := keyRune(d, 'r')
		if cmd == nil {
			t.Fatal("r must return a data-loading command")
		}
		if got := d.StateMachine.Current(); got != ui.StateLoading {
			t.Errorf("state after r = %v, want loading (§16.4 Success→Loading on refresh)", got)
		}
		if !d.Loading {
			t.Error("Loading flag must be set")
		}
	})

	t.Run("? opens help overlay with all keybindings (§18.4)", func(t *testing.T) {
		d := newDashboardFixture(t)
		if cmd := keyRune(d, '?'); cmd != nil {
			t.Error("help is an in-screen overlay — ? must not spawn a command")
		}
		if !d.ShowHelp {
			t.Fatal("? must open the help overlay")
		}
		view := d.View()
		for _, want := range []string{"? Help — dashboard", "Navigation:", "Actions:", "[?] or [Esc] close"} {
			if !strings.Contains(view, want) {
				t.Errorf("help overlay missing %q (INV-UI-010)", want)
			}
		}
		// Живые клавиши экрана обязаны быть в help — help не врёт (INV-UI-009).
		for _, key := range []string{"Tab", "Shift+Tab", "r", "q"} {
			if !strings.Contains(view, key) {
				t.Errorf("help overlay missing keybinding %q", key)
			}
		}
		// Глоссарий аббревиатур (§19.5: no jargon without definition).
		for _, term := range []string{"ACS", "LSP", "MCP"} {
			if !strings.Contains(view, term) {
				t.Errorf("help overlay missing glossary term %q", term)
			}
		}

		_, _ = d.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if d.ShowHelp {
			t.Error("Esc must close the help overlay (no dead-end, INV-UI-010)")
		}
	})

	t.Run("q quits (statusbar promises [q] quit, §18.4)", func(t *testing.T) {
		d := newDashboardFixture(t)
		if _, ok := keyRune(d, 'q')().(tea.QuitMsg); !ok {
			t.Error("q must produce tea.QuitMsg")
		}
	})

	t.Run("Esc quits", func(t *testing.T) {
		d := newDashboardFixture(t)
		_, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if cmd == nil {
			t.Fatal("Esc must return a quit command")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Error("Esc must produce tea.QuitMsg")
		}
	})
}
