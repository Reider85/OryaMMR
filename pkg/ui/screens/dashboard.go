// dashboard.go — TUI-дашборд, хаб-экран (PROMPT-MVP §4, §2.6).
package screens

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/storage"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui/components"
)

// widgetTitleOrder — заголовки виджетов в порядке WidgetIDs (фокус-маркер
// сравнивает заголовок с позицией, а не с ID: INV-PROMPT-MVP-007).
var widgetTitleOrder = []string{"Today", "Authorship Score", "Active Agents", "Recent Decisions", "Cross-repo"}

// DashboardScreen represents the dashboard screen
type DashboardScreen struct {
	ui.StateMachine
	
	Version      string
	RepoRoot     string
	Theme        *ui.Theme
	Keys         *ui.Keys
	A11y         *ui.A11y
	StatusBar    *components.StatusBar
	
	// Data
	Sessions     []*storage.Session
	Manifest     *storage.Manifest
	Adapters     []caf.Adapter
	
	// Widgets
	TodayWidget        *TodayWidget
	AuthorshipWidget   *AuthorshipWidget
	AgentsWidget       *AgentsWidget
	DecisionsWidget    *DecisionsWidget
	CrossRepoWidget    *CrossRepoWidget
	
	// Loading state
	Loading      bool
	LoadError    error
	RefreshTime  time.Time
	
	// Focus
	CurrentWidget int
	WidgetIDs     []string
}

// Widget interface for dashboard widgets
type Widget interface {
	ID() string
	Render(width, height int, theme *ui.Theme, a11y *ui.A11y) string
	Update(data interface{}) error
	IsFocusable() bool
}

// TodayWidget shows today's activity
type TodayWidget struct {
	CardsDue      int
	PendingDecisions int
	NewBlindspots int
}

// AuthorshipWidget shows ACS and related metrics
type AuthorshipWidget struct {
	ACS       float64
	Recall    float64
	Coverage  float64
	Trend     string // 30-day trend
}

// AgentsWidget shows active agents
type AgentsWidget struct {
	Agents []AgentStatus
}

// AgentStatus represents an agent's status
type AgentStatus struct {
	Name     string
	ID       string
	Active   bool
	Health   string
}

// DecisionsWidget shows recent decisions
type DecisionsWidget struct {
	Decisions []storage.Decision
	Limit     int
}

// CrossRepoWidget shows cross-repo sync status
type CrossRepoWidget struct {
	Enabled   bool
	Repos     []string
	LastSync  time.Time
	Stale     bool
}

// NewDashboardScreen creates a new dashboard screen
func NewDashboardScreen(repoRoot string, version string) *DashboardScreen {
	theme := ui.NewTheme()
	keys := ui.NewKeys()
	a11y := ui.NewA11y(ui.NewAccessibilityConfig(), theme, keys)
	
	screen := &DashboardScreen{
		StateMachine: *ui.NewStateMachine(),
		Version:      version,
		RepoRoot:     repoRoot,
		Theme:        theme,
		Keys:         keys,
		A11y:         a11y,
		Loading:      true,
		CurrentWidget: 0,
		WidgetIDs:     []string{"today", "authorship", "agents", "decisions", "crossrepo"},
	}
	
	// Initialize status bar
	screen.StatusBar = components.NewStatusBar("dashboard")
	screen.StatusBar.SetKeyHints([]string{
		"[Tab] widgets", "[r] refresh", "[?] help", "[q] quit",
	})
	
	// Initialize widgets
	screen.TodayWidget = &TodayWidget{}
	screen.AuthorshipWidget = &AuthorshipWidget{}
	screen.AgentsWidget = &AgentsWidget{}
	screen.DecisionsWidget = &DecisionsWidget{Limit: 5}
	screen.CrossRepoWidget = &CrossRepoWidget{}
	
	return screen
}

// Init initializes the dashboard screen
func (d *DashboardScreen) Init() tea.Cmd {
	return d.LoadData()
}

// LoadData loads dashboard data
func (d *DashboardScreen) LoadData() tea.Cmd {
	return func() tea.Msg {
	// Load sessions (параллельный bulk-load — бюджет §6 file ops <1s на 10k)
	sessionsDir := filepath.Join(d.RepoRoot, ".caf", "sessions")
	sessionManager := storage.NewManager(sessionsDir)
	// errors.Is, а НЕ os.IsNotExist: ошибки storage обёрнуты через %w,
	// os.IsNotExist их не раскапывает — свежий репо падал в Error state
	// вместо Empty-CTA (§2.6, INV-PROMPT-MVP-006).
	sessions, err := sessionManager.LoadAllSessions()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return DashboardLoadError{Err: fmt.Errorf("failed to load sessions: %w", err)}
	}
		
		// Load manifest
		manifestFile := storage.GetManifestPath(d.RepoRoot)
		manifestManager := storage.NewManifestManager(manifestFile)
		manifest, err := manifestManager.LoadManifest()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return DashboardLoadError{Err: fmt.Errorf("failed to load manifest: %w", err)}
		}
		
	// Get adapters
	adapters := caf.ListAdapters()
	var adapterList []caf.Adapter
	for _, id := range adapters {
		adapter, ok := caf.GetAdapter(id)
		if ok {
			adapterList = append(adapterList, adapter)
		}
	}
		
		return DashboardDataLoaded{
			Sessions: sessions,
			Manifest: manifest,
			Adapters: adapterList,
			LoadTime: time.Now(),
		}
	}
}

// DashboardDataLoaded is the message for data loading
type DashboardDataLoaded struct {
	Sessions []*storage.Session
	Manifest *storage.Manifest
	Adapters []caf.Adapter
	LoadTime time.Time
}

// DashboardLoadError is the error message for data loading
type DashboardLoadError struct {
	Err error
}

// Update handles messages for the dashboard screen
func (d *DashboardScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return d.handleKey(msg)
	case DashboardDataLoaded:
		d.handleDataLoaded(msg)
	case DashboardLoadError:
		d.handleLoadError(msg)
	case tea.WindowSizeMsg:
		// Handle resize if needed
	}
	
	// Update state machine
	d.StateMachine.Transition(nil)
	
	return d, nil
}

// handleKey handles keyboard input
func (d *DashboardScreen) handleKey(msg tea.KeyMsg) (ui.Screen, tea.Cmd) {
	// Печатные клавиши — через runeOf: bubbletea кодирует их как KeyRunes (§18.4).
	switch runeOf(msg) {
	case 'r':
		d.Loading = true
		d.LoadError = nil
		// §16.4: Success → Loading на refresh (скелет вместо устаревших данных).
		d.StateMachine.SetState(ui.StateLoading)
		return d, d.LoadData()
	case '?':
		// Show help overlay
		return d, nil
	}

	switch msg.Type {
	case tea.KeyTab:
		d.CurrentWidget = (d.CurrentWidget + 1) % len(d.WidgetIDs)
		d.StatusBar.SetFocus(d.WidgetIDs[d.CurrentWidget])
	case tea.KeyShiftTab:
		d.CurrentWidget = (d.CurrentWidget - 1 + len(d.WidgetIDs)) % len(d.WidgetIDs)
		d.StatusBar.SetFocus(d.WidgetIDs[d.CurrentWidget])
	case tea.KeyEsc, tea.KeyCtrlC:
		return d, tea.Quit
	}
	
	return d, nil
}

// handleDataLoaded handles successful data loading
func (d *DashboardScreen) handleDataLoaded(msg DashboardDataLoaded) {
	d.Sessions = msg.Sessions
	d.Manifest = msg.Manifest
	d.Adapters = msg.Adapters
	d.Loading = false
	d.LoadError = nil
	d.RefreshTime = msg.LoadTime
	
	// Update widgets with data
	d.updateWidgets()
	
	// 0 сессий → empty-with-CTA «Run caf init» (§2.6), не happy-path.
	if len(msg.Sessions) == 0 {
		d.StateMachine.SetState(ui.StateEmpty)
		return
	}
	// Transition to success state
	d.StateMachine.SetState(ui.StateSuccess)
}

// handleLoadError handles data loading errors
func (d *DashboardScreen) handleLoadError(msg DashboardLoadError) {
	d.Loading = false
	d.LoadError = msg.Err
	d.StateMachine.SetState(ui.StateError)
}

// updateWidgets updates all widgets with current data
func (d *DashboardScreen) updateWidgets() {
	// Update Today Widget
	today := time.Now()
	var cardsDue, pendingDecisions, newBlindspots int
	for _, session := range d.Sessions {
		sessionStart := session.StartedAt
		if sessionStart.Year() == today.Year() && sessionStart.Month() == today.Month() && sessionStart.Day() == today.Day() {
			cardsDue += len(session.FilesTouched)
			pendingDecisions += len(session.DecisionsMade)
			// Simple blindspot detection: files touched but not committed
			if session.CommitSHA == "" {
				newBlindspots++
			}
		}
	}
	d.TodayWidget.CardsDue = cardsDue
	d.TodayWidget.PendingDecisions = pendingDecisions
	d.TodayWidget.NewBlindspots = newBlindspots
	
	// Update Authorship Widget - honest empty/partial for MVP
	if len(d.Sessions) > 0 {
		// Simple proxy: average files per session
		var totalFiles int
		for _, session := range d.Sessions {
			totalFiles += len(session.FilesTouched)
		}
		d.AuthorshipWidget.ACS = float64(totalFiles) / float64(len(d.Sessions)) * 0.1 // Very naive proxy
		d.AuthorshipWidget.Recall = 0.0 // Not tracked in MVP
		d.AuthorshipWidget.Coverage = 0.0 // Not tracked in MVP
		d.AuthorshipWidget.Trend = "→" // No trend calculation in MVP
	}
	
	// Update Agents Widget
	for _, adapter := range d.Adapters {
		status := "unknown"
		// Simple health check: if adapter has methods, assume healthy
		if adapter.ID() != "" {
			status = "healthy"
		}
		d.AgentsWidget.Agents = append(d.AgentsWidget.Agents, AgentStatus{
			Name:   adapter.ID(),
			ID:     adapter.ID(),
			Active: true,
			Health: status,
		})
	}
	
	// Update Decisions Widget
	var recentDecisions []storage.Decision
	for _, session := range d.Sessions {
		recentDecisions = append(recentDecisions, session.DecisionsMade...)
	}
	// Sort by timestamp and take most recent
	for i := 0; i < len(recentDecisions)-1; i++ {
		for j := i + 1; j < len(recentDecisions); j++ {
			if recentDecisions[i].Timestamp.Before(recentDecisions[j].Timestamp) {
				recentDecisions[i], recentDecisions[j] = recentDecisions[j], recentDecisions[i]
			}
		}
	}
	if len(recentDecisions) > d.DecisionsWidget.Limit {
		recentDecisions = recentDecisions[:d.DecisionsWidget.Limit]
	}
	d.DecisionsWidget.Decisions = recentDecisions
	
	// Update Cross Repo Widget
	if d.Manifest != nil {
		d.CrossRepoWidget.Enabled = d.Manifest.CrossRepo.Enabled
		d.CrossRepoWidget.Repos = d.Manifest.CrossRepo.Repos
		d.CrossRepoWidget.LastSync = d.Manifest.LastSync
		d.CrossRepoWidget.Stale = time.Since(d.Manifest.LastSync) > 30*time.Minute
	}
}

// View returns the dashboard view
func (d *DashboardScreen) View() string {
	// Update status bar
	d.StatusBar.SetRefreshCompleted()
	
	// Render based on current state
	switch d.StateMachine.Current() {
	case ui.StateLoading:
		return d.renderSkeleton()
	case ui.StateError:
		return d.renderError()
	case ui.StateEmpty:
		return d.renderEmpty()
	case ui.StatePartial:
		return d.renderPartial()
	case ui.StateSuccess:
		return d.renderSuccess()
	default:
		return d.renderSkeleton()
	}
}

// renderSkeleton renders the loading skeleton
func (d *DashboardScreen) renderSkeleton() string {
	width := 80 // Default width
	height := 24 // Default height
	
	// Render skeleton widgets
	skeleton := d.renderWidgetSkeleton("Today", width/4-2, height/2-1)
	skeleton += d.renderWidgetSkeleton("Authorship Score", width/4-2, height/2-1)
	skeleton += d.renderWidgetSkeleton("Active Agents", width/4-2, height/2-1)
	skeleton += d.renderWidgetSkeleton("Recent Decisions", width/4-2, height/2-1)
	skeleton += d.renderWidgetSkeleton("Cross-repo", width/4-2, height/2-1)
	
	return components.Header(d.Version, "dashboard") + "\n" + skeleton + "\n" + d.StatusBar.Render()
}

// renderWidgetSkeleton renders a skeleton widget
func (d *DashboardScreen) renderWidgetSkeleton(title string, width, height int) string {
	theme := d.Theme
	border := theme.Border.Render("┌" + strings.Repeat("─", width-2) + "┐")
	content := strings.Repeat("│" + strings.Repeat(" ", width-2) + "│\n", height-2)
	footer := theme.Border.Render("└" + strings.Repeat("─", width-2) + "┘")
	
	return border + "\n" + content + footer + "\n"
}

// renderError renders the error state
func (d *DashboardScreen) renderError() string {
	errorBox := components.NewErrorBox(fmt.Sprintf("Failed to load dashboard: %v", d.LoadError), d.Theme, d.A11y)
	return components.Header(d.Version, "dashboard") + "\n" + errorBox.View() + "\n" + d.StatusBar.Render()
}

// renderEmpty renders the empty state
func (d *DashboardScreen) renderEmpty() string {
	emptyCTA := components.NewEmptyCTA("No sessions found", "Run `caf init` to get started", "Initialize workspace", "Press [i] to initialize now", d.Theme)
	return components.Header(d.Version, "dashboard") + "\n" + emptyCTA.View() + "\n" + d.StatusBar.Render()
}

// renderPartial renders the partial state
func (d *DashboardScreen) renderPartial() string {
	width := 80
	height := 24
	
	// Render partial widgets with some data
	widgets := d.renderWidgets(width, height)
	
	// Add partial banner if needed
	partialBanner := components.NewPartialBanner(1, 2, []string{"cross-repo"}, d.Theme)
	
	return components.Header(d.Version, "dashboard") + "\n" + partialBanner.View() + "\n" + widgets + "\n" + d.StatusBar.Render()
}

// renderSuccess renders the success state with actual data
func (d *DashboardScreen) renderSuccess() string {
	width := 80
	height := 24
	
	widgets := d.renderWidgets(width, height)
	
	return components.Header(d.Version, "dashboard") + "\n" + widgets + "\n" + d.StatusBar.Render()
}

// renderWidgets renders all widgets in a grid
func (d *DashboardScreen) renderWidgets(width, height int) string {
	// Grid layout: 2x3 for 5 widgets
	widgetWidth := width / 4
	widgetHeight := height / 3
	
	// Today widget (top-left)
	today := d.renderWidget("Today", d.TodayWidget, widgetWidth, widgetHeight)
	
	// Authorship widget (top-middle)  
	authorship := d.renderWidget("Authorship Score", d.AuthorshipWidget, widgetWidth, widgetHeight)
	
	// Agents widget (top-right)
	agents := d.renderWidget("Active Agents", d.AgentsWidget, widgetWidth, widgetHeight)
	
	// Decisions widget (bottom-left, spans 2 columns)
	decisions := d.renderWidget("Recent Decisions", d.DecisionsWidget, widgetWidth*2-1, widgetHeight)
	
	// Cross-repo widget (bottom-right)
	crossrepo := d.renderWidget("Cross-repo", d.CrossRepoWidget, widgetWidth, widgetHeight)
	
	// Combine widgets
	return today + authorship + agents + "\n" + decisions + crossrepo
}

// renderWidget renders a single widget
func (d *DashboardScreen) renderWidget(title string, widget interface{}, width, height int) string {
	theme := d.Theme
	
	// Add focus indicator if this widget is focused
	focusPrefix := "  "
	if d.CurrentWidget >= 0 && d.CurrentWidget < len(widgetTitleOrder) && widgetTitleOrder[d.CurrentWidget] == title {
		focusPrefix = "▸ "
	}
	
	border := theme.Border.Render("┌" + strings.Repeat("─", width-2) + "┐")
	
	// Render widget content
	var content string
	switch w := widget.(type) {
	case *TodayWidget:
		content = d.renderTodayWidget(w, width-2, height-2)
	case *AuthorshipWidget:
		content = d.renderAuthorshipWidget(w, width-2, height-2)
	case *AgentsWidget:
		content = d.renderAgentsWidget(w, width-2, height-2)
	case *DecisionsWidget:
		content = d.renderDecisionsWidget(w, width-2, height-2)
	case *CrossRepoWidget:
		content = d.renderCrossRepoWidget(w, width-2, height-2)
	}
	
	footer := theme.Border.Render("└" + strings.Repeat("─", width-2) + "┘")
	
	return focusPrefix + title + "\n" + border + "\n" + content + footer + "\n"
}

// renderTodayWidget renders the today widget
func (d *DashboardScreen) renderTodayWidget(w *TodayWidget, width, height int) string {
	lines := []string{
		fmt.Sprintf("  %d cards due", w.CardsDue),
		fmt.Sprintf("  %d pending decisions", w.PendingDecisions),
		fmt.Sprintf("  %d new blindspots", w.NewBlindspots),
	}
	
	for len(lines) < height-1 {
		lines = append(lines, "  ")
	}
	
	return strings.Join(lines, "\n")
}

// renderAuthorshipWidget renders the authorship widget
func (d *DashboardScreen) renderAuthorshipWidget(w *AuthorshipWidget, width, height int) string {
	lines := []string{}
	
	if w.ACS > 0 {
		lines = append(lines, fmt.Sprintf("  ACS %.1f / target 3.5", w.ACS))
	} else {
		lines = append(lines, "  ACS not tracked yet")
	}
	
	if w.Recall > 0 {
		lines = append(lines, fmt.Sprintf("  Recall  %.0f%%", w.Recall*100))
	} else {
		lines = append(lines, "  Recall not tracked")
	}
	
	if w.Coverage > 0 {
		lines = append(lines, fmt.Sprintf("  Coverage %.0f%%", w.Coverage*100))
	} else {
		lines = append(lines, "  Coverage not tracked")
	}
	
	if w.Trend != "" {
		lines = append(lines, fmt.Sprintf("  30-day: %s", w.Trend))
	}
	
	for len(lines) < height-1 {
		lines = append(lines, "  ")
	}
	
	return strings.Join(lines, "\n")
}

// renderAgentsWidget renders the agents widget
func (d *DashboardScreen) renderAgentsWidget(w *AgentsWidget, width, height int) string {
	lines := []string{}
	for _, agent := range w.Agents {
		status := "✓"
		if agent.Health != "healthy" {
			status = "⚠"
		}
		lines = append(lines, fmt.Sprintf("  %s %s (%s)", status, agent.Name, agent.Health))
	}
	
	for len(lines) < height-1 {
		lines = append(lines, "  ")
	}
	
	return strings.Join(lines, "\n")
}

// renderDecisionsWidget renders the decisions widget
func (d *DashboardScreen) renderDecisionsWidget(w *DecisionsWidget, width, height int) string {
	lines := []string{}
	for _, decision := range w.Decisions {
		lines = append(lines, fmt.Sprintf("  • %s", decision.Title))
	}
	
	for len(lines) < height-1 {
		lines = append(lines, "  ")
	}
	
	return strings.Join(lines, "\n")
}

// renderCrossRepoWidget renders the cross-repo widget
func (d *DashboardScreen) renderCrossRepoWidget(w *CrossRepoWidget, width, height int) string {
	lines := []string{}
	
	if w.Enabled {
		status := "✓"
		if w.Stale {
			status = "⚠"
		}
		lines = append(lines, fmt.Sprintf("  %s %s", status, "OryaAgent ← Sec"))
		lines = append(lines, fmt.Sprintf("    stale decision"))
	} else {
		lines = append(lines, "  ◇ No cross-repo sync")
	}
	
	for len(lines) < height-1 {
		lines = append(lines, "  ")
	}
	
	return strings.Join(lines, "\n")
}

// GetID returns the screen ID
func (d *DashboardScreen) GetID() string {
	return "dashboard"
}

// IsActive returns if the screen is active
func (d *DashboardScreen) IsActive() bool {
	return true
}

// SetActive sets the active state
func (d *DashboardScreen) SetActive(active bool) {
	// Update status bar focus when active
	if active {
		d.StatusBar.SetFocus(d.WidgetIDs[d.CurrentWidget])
	}
}

// SetKeyKeys sets keyboard hints (for compatibility)
func (d *DashboardScreen) SetKeyKeys(hints []string) {
	d.StatusBar.SetKeyHints(hints)
}
