// dashboard_tui_test.go — тесты обёрток tea.Model для трёх экранов:
// конструирование, делегирование Update/View без запуска интерактивной сессии.
package main

import (
	"strings"
	"testing"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui/screens"
)

func TestDashboardModelLifecycle(t *testing.T) {
	repoRoot := t.TempDir()
	m := NewDashboardModel(repoRoot, "0.4.1-test")

	if m.Screen == nil {
		t.Fatal("NewDashboardModel must wrap a screen")
	}
	if !m.Loading {
		t.Error("fresh model must be in loading state")
	}

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init must delegate to screen LoadData")
	}

	// Успешная загрузка гасит loading-флаг обёртки.
	loaded, ok := cmd().(screens.DashboardDataLoaded)
	if !ok {
		t.Fatalf("Init must produce DashboardDataLoaded, got %T", cmd())
	}
	_, _ = m.Update(loaded)
	if m.Loading {
		t.Error("Loading must be false after DashboardDataLoaded")
	}

	// Ошибка загрузки запоминается обёрткой.
	_, _ = m.Update(screens.DashboardLoadError{Err: errFake("boom")})
	if m.Loading {
		t.Error("Loading must be false after DashboardLoadError")
	}
	if m.LoadError == nil || m.LoadError.Error() != "boom" {
		t.Errorf("LoadError = %v, want boom", m.LoadError)
	}

	// View делегируется экрану и рендерится (скелет состояния loading).
	m2 := NewDashboardModel(repoRoot, "0.4.1-test")
	view := m2.View()
	if view == "" {
		t.Fatal("View must not be empty")
	}
	if !strings.Contains(view, "CodeAuthorityFirst") {
		t.Errorf("View must render screen content, got:\n%s", view)
	}
}

// errFake — минимальная ошибка для проверки делегирования.
type errFake string

func (e errFake) Error() string { return string(e) }

func TestAdaptersModelLifecycle(t *testing.T) {
	repoRoot := t.TempDir()
	m := NewAdaptersModel(repoRoot, "0.4.1-test")

	if m.Screen == nil {
		t.Fatal("NewAdaptersModel must wrap a screen")
	}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init must delegate to screen LoadData")
	}

	raw := cmd()
	if _, ok := raw.(screens.AdaptersDataLoaded); !ok {
		t.Fatalf("Init must produce AdaptersDataLoaded, got %T", raw)
	}
	// Обёртка не ведёт собственный loading (см. комментарий в коде) —
	// важно, что делегирование не паникует и экран меняет состояние.
	_, _ = m.Update(raw)
	if m.Screen.StateMachine.Current().String() == "loading" {
		t.Error("screen must leave loading state after data arrives")
	}

	view := m.View()
	if !strings.Contains(view, "CodeAuthorityFirst") {
		t.Errorf("View must render screen content, got:\n%s", view)
	}
}

func TestOnboardingModelLifecycle(t *testing.T) {
	repoRoot := t.TempDir()
	m := NewOnboardingModel(repoRoot, "0.4.1-test")

	if m.Screen == nil {
		t.Fatal("NewOnboardingModel must wrap a screen")
	}
	if cmd := m.Init(); cmd != nil {
		t.Error("onboarding Init (welcome step) must not spawn commands")
	}

	// Делегирование DetectionComplete экрану.
	_, _ = m.Update(screens.DetectionComplete{Detected: []string{"opencode"}})
	if got := m.Screen.StateMachine.Current().String(); got != "success" {
		t.Errorf("screen state after detection = %s, want success", got)
	}

	view := m.View()
	if !strings.Contains(view, "CodeAuthorityFirst") {
		t.Errorf("View must render screen content, got:\n%s", view)
	}
}
