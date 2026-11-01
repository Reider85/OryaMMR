// dashboard_tui.go — direct TUI integration for dashboard (temporary MVP workaround)
// This avoids the import cycle issue between ui and screens packages
package main

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/storage"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui/components"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui/screens"
)

// runTUI — единственная точка запуска интерактивной сессии bubbletea.
// Выделена в переменную ради тестов (§10): тесты перехватывают запуск
// вместо вечного event loop, поведение прод-кода не меняется.
var runTUI = func(model tea.Model, opts ...tea.ProgramOption) (tea.Model, error) {
	return tea.NewProgram(model, opts...).Run()
}

// DashboardModel is a direct model for the dashboard screen
type DashboardModel struct {
	RepoRoot     string
	Version      string
	Theme        *ui.Theme
	Keys         *ui.Keys
	A11y         *ui.A11y
	StatusBar    *components.StatusBar
	
	// Screen
	Screen       *screens.DashboardScreen
	
	// Loading state
	Loading      bool
	LoadError    error
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(repoRoot, version string) *DashboardModel {
	theme := ui.NewTheme()
	keys := ui.NewKeys()
	a11y := ui.NewA11y(ui.NewAccessibilityConfig(), theme, keys)
	
	screen := screens.NewDashboardScreen(repoRoot, version)
	
	return &DashboardModel{
		RepoRoot:  repoRoot,
		Version:   version,
		Theme:     theme,
		Keys:      keys,
		A11y:      a11y,
		StatusBar: components.NewStatusBar("dashboard"),
		Screen:    screen,
		Loading:   true,
	}
}

// Init initializes the dashboard model
func (m *DashboardModel) Init() tea.Cmd {
	return m.Screen.LoadData()
}

// Update handles messages for the dashboard model
func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	// Handle screen updates
	screen, cmd := m.Screen.Update(msg)
	if dashboardScreen, ok := screen.(*screens.DashboardScreen); ok {
		m.Screen = dashboardScreen
	}
	
	// Update loading state
	switch msg := msg.(type) {
	case screens.DashboardDataLoaded:
		m.Loading = false
		m.LoadError = nil
	case screens.DashboardLoadError:
		m.Loading = false
		m.LoadError = msg.Err
	}
	
	return m, cmd
}

// View returns the dashboard view
func (m *DashboardModel) View() string {
	return m.Screen.View()
}

// runDashboardDirect runs the dashboard directly (avoiding AppModel cycle issue)
func runDashboardDirect(repoRoot string) error {
	version := "0.4.1" // TODO: get from build info
	model := NewDashboardModel(repoRoot, version)
	if _, err := runTUI(model); err != nil {
		return fmt.Errorf("dashboard failed: %w", err)
	}
	return nil
}

// AdaptersModel is a direct model for the adapters screen
type AdaptersModel struct {
	RepoRoot     string
	Version      string
	Theme        *ui.Theme
	Keys         *ui.Keys
	A11y         *ui.A11y
	StatusBar    *components.StatusBar
	
	// Screen
	Screen       *screens.AdaptersScreen
	
	// Loading state
	Loading      bool
	LoadError    error
}

// NewAdaptersModel creates a new adapters model
func NewAdaptersModel(repoRoot, version string) *AdaptersModel {
	theme := ui.NewTheme()
	keys := ui.NewKeys()
	a11y := ui.NewA11y(ui.NewAccessibilityConfig(), theme, keys)
	
	screen := screens.NewAdaptersScreen(repoRoot, version)
	
	return &AdaptersModel{
		RepoRoot:  repoRoot,
		Version:   version,
		Theme:     theme,
		Keys:      keys,
		A11y:      a11y,
		StatusBar: components.NewStatusBar("adapters"),
		Screen:    screen,
		Loading:   true,
	}
}

// Init initializes the adapters model
func (m *AdaptersModel) Init() tea.Cmd {
	return m.Screen.LoadData()
}

// Update handles messages for the adapters model
func (m *AdaptersModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	// Handle screen updates
	screen, cmd := m.Screen.Update(msg)
	if adaptersScreen, ok := screen.(*screens.AdaptersScreen); ok {
		m.Screen = adaptersScreen
	}
	
	// Update loading state
	// Note: AdaptersScreen doesn't have explicit loading messages yet
	// This will be updated when the screen is fully implemented
	
	return m, cmd
}

// View returns the adapters view
func (m *AdaptersModel) View() string {
	return m.Screen.View()
}

// runAdaptersTUI runs the adapters TUI directly (avoiding AppModel cycle issue)
func runAdaptersTUI(repoRoot string) error {
	version := "0.4.1" // TODO: get from build info
	model := NewAdaptersModel(repoRoot, version)
	if _, err := runTUI(model, tea.WithAltScreen()); err != nil {
		return fmt.Errorf("adapters TUI failed: %w", err)
	}
	return nil
}

// OnboardingModel is a direct model for the onboarding screen
type OnboardingModel struct {
	RepoRoot     string
	Version      string
	Theme        *ui.Theme
	Keys         *ui.Keys
	A11y         *ui.A11y
	StatusBar    *components.StatusBar
	
	// Screen
	Screen       *screens.OnboardingScreen
	
	// Loading state
	Loading      bool
	LoadError    error
}

// NewOnboardingModel creates a new onboarding model
func NewOnboardingModel(repoRoot, version string) *OnboardingModel {
	theme := ui.NewTheme()
	keys := ui.NewKeys()
	a11y := ui.NewA11y(ui.NewAccessibilityConfig(), theme, keys)
	
	screen := screens.NewOnboardingScreen(repoRoot, version)
	
	return &OnboardingModel{
		RepoRoot:  repoRoot,
		Version:   version,
		Theme:     theme,
		Keys:      keys,
		A11y:      a11y,
		StatusBar: components.NewStatusBar("onboarding"),
		Screen:    screen,
		Loading:   true,
	}
}

// Init initializes the onboarding model
func (m *OnboardingModel) Init() tea.Cmd {
	return m.Screen.Init()
}

// Update handles messages for the onboarding model
func (m *OnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	// Handle screen updates
	screen, cmd := m.Screen.Update(msg)
	if onboardingScreen, ok := screen.(*screens.OnboardingScreen); ok {
		m.Screen = onboardingScreen
	}
	
	// Update loading state
	// Note: OnboardingScreen doesn't have explicit loading messages yet
	// This will be updated when the screen is fully implemented
	
	return m, cmd
}

// View returns the onboarding view
func (m *OnboardingModel) View() string {
	return m.Screen.View()
}

// runInitTUI runs the onboarding TUI directly (avoiding AppModel cycle issue)
func runInitTUI(repoRoot string) error {
	version := "0.4.1" // TODO: get from build info
	
	// First initialize the repository structure
	if err := storage.InitializeCAF(repoRoot); err != nil {
		return fmt.Errorf("failed to initialize .caf/: %w", err)
	}
	
	// Then launch the onboarding TUI
	model := NewOnboardingModel(repoRoot, version)
	if _, err := runTUI(model, tea.WithAltScreen()); err != nil {
		return fmt.Errorf("onboarding TUI failed: %w", err)
	}
	return nil
}