package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screen represents a screen in the TUI
type Screen interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Screen, tea.Cmd)
	View() string
	GetID() string
	IsActive() bool
	SetActive(active bool)
}

// PlaceholderScreen is a temporary screen for MVP
type PlaceholderScreen struct {
	ID    string
	Theme *Theme
}

// Init initializes the placeholder screen
func (p *PlaceholderScreen) Init() tea.Cmd {
	return nil
}

// Update handles messages for the placeholder screen
func (p *PlaceholderScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	return p, nil
}

// View returns the placeholder screen view
func (p *PlaceholderScreen) View() string {
	return fmt.Sprintf("🚧 %s screen - Implementation coming in steps 10-13\n\nPress ? for help, Esc to quit", p.ID)
}

// GetID returns the screen ID
func (p *PlaceholderScreen) GetID() string {
	return p.ID
}

// IsActive returns if the screen is active
func (p *PlaceholderScreen) IsActive() bool {
	return true
}

// SetActive sets the active state
func (p *PlaceholderScreen) SetActive(active bool) {
	// No-op for placeholder
}

// ScreenType represents the type of screen
type ScreenType int

const (
	ScreenDashboard ScreenType = iota
	ScreenDoctor
	ScreenSettings
)

// String returns the string representation of the screen type
func (st ScreenType) String() string {
	switch st {
	case ScreenDashboard:
		return "dashboard"
	case ScreenDoctor:
		return "doctor"
	case ScreenSettings:
		return "settings"
	default:
		return "unknown"
	}
}

// AppModel represents the main application model
type AppModel struct {
	Theme         *Theme
	Keys          *Keys
	A11y          *A11y
	PerfTracker   *PerfTracker
	Screens       map[ScreenType]Screen
	CurrentScreen ScreenType
	ActiveScreen  Screen
	FocusManager  *FocusManager
	ScreenReader  *ScreenReaderManager
	Width         int
	Height        int
	Quit          bool
	RepoRoot      string // Repository root path for screens
}

// NewAppModel creates a new application model
func NewAppModel(repoRoot string) *AppModel {
	theme := NewTheme()
	keys := NewKeys()
	a11y := NewA11y(NewAccessibilityConfig(), theme, keys)
	perfTracker := NewPerfTracker(NewPerformanceBudget())
	focusManager := NewFocusManager(a11y)
	screenReader := NewScreenReaderManager(a11y)
	
	return &AppModel{
		Theme:        theme,
		Keys:         keys,
		A11y:         a11y,
		PerfTracker:  perfTracker,
		Screens:      make(map[ScreenType]Screen),
		CurrentScreen: ScreenDashboard,
		FocusManager: focusManager,
		ScreenReader: screenReader,
		Width:        0,
		Height:       0,
		Quit:         false,
		RepoRoot:     repoRoot,
	}
}

// Init initializes the application
func (m *AppModel) Init() tea.Cmd {
	// Precompute styles for performance
	m.Theme.PrecomputeStyles()
	
	// MVP screens will be initialized when implemented (screens/ package)
	// For now, create a placeholder dashboard
	m.CurrentScreen = ScreenDashboard
	m.ActiveScreen = &PlaceholderScreen{ID: "dashboard", Theme: m.Theme}
	
	// Announce application start
	m.ScreenReader.AddAnnouncement("app_start", "CodeAuthorityFirst TUI started", AnnouncementImportant, "Application", false)
	
	// Initialize active screen
	return m.ActiveScreen.Init()
}

// Update updates the application
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
		
	case tea.KeyMsg:
		// Handle global keyboard shortcuts
		switch msg.String() {
		case "ctrl+c", "esc":
			m.Quit = true
			return m, tea.Quit
			
		case "?":
			// Show help
			return m, m.showHelp()
			
		case "r":
			// Refresh current screen
			return m, m.refreshScreen()
			
		case "1":
			// Switch to dashboard
			return m, m.switchScreen(ScreenDashboard)
			
		case "2":
			// Switch to doctor
			return m, m.switchScreen(ScreenDoctor)
			
		case "3":
			// Switch to settings
			return m, m.switchScreen(ScreenSettings)
		}
		
		// Handle screen-specific updates
		screen, cmd := m.ActiveScreen.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		
		// Update screen if it changed
		if screen != m.ActiveScreen {
			m.ActiveScreen = screen
			cmds = append(cmds, screen.Init())
		}
		
	case ScreenSwitchMsg:
		// Handle screen switch (only for existing screens in MVP scope)
		if msg.ScreenType != m.CurrentScreen && m.Screens[msg.ScreenType] != nil {
			m.CurrentScreen = msg.ScreenType
			m.ActiveScreen = m.Screens[msg.ScreenType]
			cmds = append(cmds, m.ActiveScreen.Init())
		}
		
	case RefreshScreenMsg:
		// Handle screen refresh
		cmds = append(cmds, m.ActiveScreen.Init())
		
	case HelpMsg:
		// Show help
		return m, m.showHelp()
		
	case PerformanceWarningMsg:
		// Handle performance warnings
		m.A11y.Announce(msg.Level, msg.Message)
		return m, nil
	}
	
	return m, tea.Batch(cmds...)
}

// View returns the application view
func (m *AppModel) View() string {
	if m.Quit {
		return "Goodbye!"
	}
	
	// Track performance for view rendering
	m.PerfTracker.Start("view_render")
	defer m.PerfTracker.End("view_render")
	
	// Get the active screen view
	screenView := m.ActiveScreen.View()
	
	// Add status bar
	statusBar := m.renderStatusBar()
	
	// Combine views
	view := fmt.Sprintf("%s\n%s", screenView, statusBar)
	
	// Apply accessibility transformations
	if m.A11y.IsLargeText() {
		view = lipgloss.NewStyle().Width(m.Width).Height(m.Height).Render(view)
	}
	
	return view
}

// renderStatusBar renders the status bar
func (m *AppModel) renderStatusBar() string {
	status := fmt.Sprintf("Screen: %s | Performance: %s | Press ? for help", 
		m.CurrentScreen.String(), 
		m.getPerformanceStatus())
	
	return m.Theme.StatusBar.Render(status)
}

// getPerformanceStatus returns the performance status
func (m *AppModel) getPerformanceStatus() string {
	// Check performance budgets
	if within, _, _ := m.PerfTracker.CheckBudget("frame_render"); within {
		return "Good"
	}
	return "Slow"
}

// switchScreen switches to a different screen
func (m *AppModel) switchScreen(screenType ScreenType) tea.Cmd {
	return func() tea.Msg {
		return ScreenSwitchMsg{ScreenType: screenType}
	}
}

// refreshScreen refreshes the current screen
func (m *AppModel) refreshScreen() tea.Cmd {
	return func() tea.Msg {
		return RefreshScreenMsg{Force: true}
	}
}

// showHelp shows help information
func (m *AppModel) showHelp() tea.Cmd {
	return func() tea.Msg {
		return HelpMsg{ScreenID: m.CurrentScreen.String()}
	}
}

// GetScreen returns a screen by type
func (m *AppModel) GetScreen(screenType ScreenType) Screen {
	return m.Screens[screenType]
}

// GetCurrentScreen returns the current screen
func (m *AppModel) GetCurrentScreen() Screen {
	return m.ActiveScreen
}

// GetScreenType returns the current screen type
func (m *AppModel) GetScreenType() ScreenType {
	return m.CurrentScreen
}

// SetScreen sets a screen by type
func (m *AppModel) SetScreen(screenType ScreenType, screen Screen) {
	m.Screens[screenType] = screen
	if screenType == m.CurrentScreen {
		m.ActiveScreen = screen
	}
}

// ScreenSwitchMsg represents a screen switch message
type ScreenSwitchMsg struct {
	ScreenType ScreenType
}

// RefreshScreenMsg represents a screen refresh message
type RefreshScreenMsg struct {
	Force bool
}

// PerformanceWarningMsg represents a performance warning message
type PerformanceWarningMsg struct {
	Level   AnnouncementLevel
	Message string
}

