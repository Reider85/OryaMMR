// screens_coverage_test.go — добивка покрытия pkg/ui/screens (§10 ≥80%):
// Init/LoadData/FS-команды, lifecycle-геттеры, недостающие ветки рендера.
package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/storage"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// TestScreenLifecycleAccessors — контракт ui.Screen для всех трёх экранов.
func TestScreenLifecycleAccessors(t *testing.T) {
	d := newDashboardFixture(t)
	if d.GetID() != "dashboard" || !d.IsActive() {
		t.Errorf("dashboard identity: id=%q active=%v", d.GetID(), d.IsActive())
	}
	d.SetActive(true)
	if got := d.StatusBar.Focus; got != "today" {
		t.Errorf("SetActive must focus first widget, got %q", got)
	}
	d.SetKeyKeys([]string{"[x] test"})
	if !strings.Contains(d.StatusBar.Render(), "[x] test") {
		t.Error("SetKeyKeys must update status bar hints")
	}

	a := newAdaptersFixture(t)
	if a.GetID() != "adapters" || !a.IsActive() {
		t.Errorf("adapters identity: id=%q active=%v", a.GetID(), a.IsActive())
	}
	a.SetActive(true)
	if got := a.StatusBar.Focus; got != "adapters" {
		t.Errorf("SetActive focus = %q", got)
	}
	a.SetKeyKeys([]string{"[y] test"})
	if !strings.Contains(a.StatusBar.Render(), "[y] test") {
		t.Error("SetKeyKeys must update status bar hints")
	}

	o := newOnboardingFixture(t, 1)
	if o.GetID() != "onboarding" || !o.IsActive() {
		t.Errorf("onboarding identity: id=%q active=%v", o.GetID(), o.IsActive())
	}
	o.SetActive(true)
	if got := o.StatusBar.Focus; got != "onboarding" {
		t.Errorf("SetActive focus = %q", got)
	}
	o.SetKeyKeys([]string{"[z] test"})
	if !strings.Contains(o.StatusBar.Render(), "[z] test") {
		t.Error("SetKeyKeys must update status bar hints")
	}
}

// TestDashboardInitAndLoadData — Init возвращает команду, команда читает
// .caf/ реального (temp) репозитория и отдаёт DataLoaded.
func TestDashboardInitAndLoadData(t *testing.T) {
	d := NewDashboardScreen(t.TempDir(), "0.4.1-test")

	cmd := d.Init()
	if cmd == nil {
		t.Fatal("Init must return LoadData command")
	}
	msg := cmd()
	loaded, ok := msg.(DashboardDataLoaded)
	if !ok {
		t.Fatalf("LoadData must produce DashboardDataLoaded, got %T", msg)
	}
	if loaded.LoadTime.IsZero() {
		t.Error("LoadTime must be set")
	}
	if len(loaded.Sessions) != 0 {
		t.Fatalf("fresh temp repo must have 0 sessions, got %d", len(loaded.Sessions))
	}

	// Свежий репо (нет .caf/): загрузка НЕ падает в error — empty-with-CTA
	// «Run caf init» (§2.6, INV-PROMPT-MVP-006).
	_, _ = d.Update(loaded)
	if got := d.StateMachine.Current(); got != ui.StateEmpty {
		t.Errorf("state on fresh repo = %v, want empty (§2.6)", got)
	}
}

// TestDashboardUpdateWidgetsToday — ветка «сессия начата сегодня» (time.Now-
// зависимая: StartedAt = сейчас → совпадение по год/месяц/день детерминировано).
func TestDashboardUpdateWidgetsToday(t *testing.T) {
	d := newDashboardFixture(t)
	now := time.Now()

	loaded := DashboardDataLoaded{
		Sessions: []*storage.Session{
			{
				SessionID:    "today-session",
				StartedAt:    now,
				FilesTouched: []string{"a.go", "b.go"},
				// CommitSHA пуст → blindspot
				DecisionsMade: []storage.Decision{
					{ID: "d1", Title: "Later", Timestamp: now.Add(-time.Hour)},
					{ID: "d2", Title: "Newer", Timestamp: now.Add(-time.Minute)},
				},
			},
		},
		Manifest: &storage.Manifest{
			LastSync: now.Add(-2 * time.Hour), // старше 30 минут → stale
			CrossRepo: storage.CrossRepo{
				Enabled: true,
				Repos:   []string{"OryaSecurity"},
			},
		},
		LoadTime: now,
	}
	_, _ = d.Update(loaded)

	if d.TodayWidget.CardsDue != 2 {
		t.Errorf("CardsDue = %d, want 2 (files of today's session)", d.TodayWidget.CardsDue)
	}
	if d.TodayWidget.NewBlindspots != 1 {
		t.Errorf("NewBlindspots = %d, want 1 (uncommitted session)", d.TodayWidget.NewBlindspots)
	}
	if !d.CrossRepoWidget.Stale {
		t.Error("LastSync 2h ago must mark cross-repo data stale (§2.6)")
	}
	if !d.CrossRepoWidget.Enabled || len(d.CrossRepoWidget.Repos) != 1 {
		t.Errorf("CrossRepo not propagated: %+v", d.CrossRepoWidget)
	}
	// Сортировка решений: новейшее первым.
	if len(d.DecisionsWidget.Decisions) != 2 || d.DecisionsWidget.Decisions[0].ID != "d2" {
		t.Errorf("decisions not sorted desc: %+v", d.DecisionsWidget.Decisions)
	}
	// ACS наивный прокси: 2 файла / 1 сессия * 0.1 = 0.2
	if d.AuthorshipWidget.ACS != 0.2 {
		t.Errorf("ACS = %v, want 0.2", d.AuthorshipWidget.ACS)
	}
}

// TestAdaptersInitLoadAndInstalledProbe — Init/LoadData + детект установленных.
func TestAdaptersInitLoadAndInstalledProbe(t *testing.T) {
	repoRoot := t.TempDir()
	a := NewAdaptersScreen(repoRoot, "0.4.1-test")

	if a.isAdapterInstalled("opencode") {
		t.Error("fresh repo must have no installed adapters")
	}
	installedDir := filepath.Join(repoRoot, ".caf", "adapters")
	if err := os.MkdirAll(installedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installedDir, "opencode.yml"), []byte("id: opencode\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !a.isAdapterInstalled("opencode") {
		t.Error("existing .caf/adapters/opencode.yml must count as installed")
	}

	cmd := a.Init()
	if cmd == nil {
		t.Fatal("Init must return LoadData command")
	}
	raw := cmd()
	msg, ok := raw.(AdaptersDataLoaded)
	if !ok {
		t.Fatalf("LoadData must produce AdaptersDataLoaded, got %T", raw)
	}
	if msg.LoadTime.IsZero() {
		t.Error("LoadTime must be set")
	}

	// Реестр в тестовом бинаре пуст → empty-with-CTA «[i] install» (§2.8).
	_, _ = a.Update(msg)
	if got := a.StateMachine.Current(); got != ui.StateEmpty {
		t.Errorf("state with no registered adapters = %v, want empty (§2.8)", got)
	}
}

// TestAdaptersDetailFallbackAndUnknownTier — недостающие ветки рендера.
func TestAdaptersDetailFallbackAndUnknownTier(t *testing.T) {
	t.Run("detail with nil adapter falls back to matrix", func(t *testing.T) {
		a := newAdaptersFixture(t)
		a.ShowDetail = true
		a.DetailAdapter = nil
		view := a.View()
		if !strings.Contains(view, "capability matrix") {
			t.Errorf("nil-detail must fall back to matrix view, got:\n%s", view)
		}
	})

	t.Run("unknown adapter tier and capabilities render as T2/dash", func(t *testing.T) {
		a := newAdaptersFixture(t)
		a.Adapters = append(a.Adapters, stubAdapter{id: "cursor"})
		view := a.View()
		if !strings.Contains(view, "T2") {
			t.Error("non-T1/T4 adapter must show muted T2 badge (§2.8)")
		}
		if !strings.Contains(view, "cursor") {
			t.Error("adapter row missing")
		}
	})
}

// TestOnboardingInitStartStepAndSummaryCmd — Init/StartStep/GenerateSummary.
func TestOnboardingInitStartStepAndSummaryCmd(t *testing.T) {
	t.Run("Init lands on welcome step", func(t *testing.T) {
		o := NewOnboardingScreen(t.TempDir(), "0.4.1-test")
		cmd := o.Init()
		if cmd != nil {
			t.Error("welcome step needs no async command")
		}
		if o.CurrentStep != 1 {
			t.Errorf("CurrentStep = %d, want 1", o.CurrentStep)
		}
		if got := o.StateMachine.Current(); got != ui.StateSuccess {
			t.Errorf("welcome state = %v, want success (§16.5)", got)
		}
	})

	t.Run("StartStep(4) loads summary, unknown step is safe", func(t *testing.T) {
		o := newOnboardingFixture(t, 3)
		if cmd := o.StartStep(4); cmd == nil {
			t.Error("step 4 must return summary-generation command")
		}
		if got := o.StateMachine.Current(); got != ui.StateLoading {
			t.Errorf("state on step 4 = %v, want loading", got)
		}

		if cmd := o.StartStep(99); cmd != nil {
			t.Error("unknown step must return nil command")
		}
	})

	t.Run("GenerateSummary produces SummaryComplete", func(t *testing.T) {
		o := newOnboardingFixture(t, 4)
		o.InstallSuccess = []string{"opencode"}
		msg, ok := o.GenerateSummary()().(SummaryComplete)
		if !ok {
			t.Fatal("GenerateSummary must produce SummaryComplete")
		}
		if !strings.Contains(msg.Message, "1 adapters installed") {
			t.Errorf("summary message = %q", msg.Message)
		}
	})

	t.Run("summary without installs warns", func(t *testing.T) {
		o := newOnboardingFixture(t, 4)
		msg, ok := o.GenerateSummary()().(SummaryComplete)
		if !ok {
			t.Fatal("want SummaryComplete")
		}
		if !strings.Contains(msg.Message, "warnings") {
			t.Errorf("empty-install summary = %q", msg.Message)
		}
	})
}

// TestOnboardingInstallationLoadingRender — ветка «установка в процессе».
func TestOnboardingInstallationLoadingRender(t *testing.T) {
	o := newOnboardingFixture(t, 3)
	o.Loading = true
	o.DetectedAdapters = []string{"opencode"}
	o.InstallingAdapters = map[string]bool{"opencode": true}
	o.StateMachine.SetState(ui.StateLoading)

	view := o.View()
	if !strings.Contains(view, "Installing adapters") {
		t.Errorf("loading install view missing marker, got:\n%s", view)
	}
	if !strings.Contains(view, "installing...") {
		t.Errorf("per-adapter progress line missing (§2.7), got:\n%s", view)
	}
}

// TestOnboardingWelcomeAndDetectionRenderContent — контент шагов 1-2 (без golden).
func TestOnboardingWelcomeAndDetectionRenderContent(t *testing.T) {
	o := newOnboardingFixture(t, 1)
	o.StateMachine.SetState(ui.StateSuccess)
	if view := o.View(); !strings.Contains(view, "Welcome to CodeAuthorityFirst") {
		t.Errorf("welcome content missing:\n%s", view)
	}

	o2 := newOnboardingFixture(t, 2)
	o2.StateMachine.SetState(ui.StateSuccess)
	o2.DetectedAdapters = []string{"opencode"}
	view := o2.View()
	if !strings.Contains(view, "✓ opencode detected") {
		t.Errorf("detection results missing:\n%s", view)
	}
}

// TestStubAdapterCapabilitiesValid — стаб-адаптер отдаёт валидную матрицу
// (используется фикстурами success-рендера).
func TestStubAdapterCapabilitiesValid(t *testing.T) {
	s := stubAdapter{id: "x"}
	if err := caf.ValidateCapabilityMatrix(s.Capabilities()); err != nil {
		t.Errorf("stub matrix must be valid: %v", err)
	}
	if s.ID() != "x" {
		t.Errorf("ID = %q", s.ID())
	}
	if err := s.Install(); err != nil {
		t.Errorf("stub Install: %v", err)
	}
	if err := s.Uninstall(); err != nil {
		t.Errorf("stub Uninstall: %v", err)
	}
}
