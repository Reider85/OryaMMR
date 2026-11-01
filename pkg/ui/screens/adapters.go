// adapters.go — TUI-менеджер адаптеров (PROMPT-MVP §4, §2.8).
package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui/components"
)

// AdaptersScreen represents the adapters management screen
type AdaptersScreen struct {
	ui.StateMachine
	
	Version      string
	RepoRoot     string
	Theme        *ui.Theme
	Keys         *ui.Keys
	A11y         *ui.A11y
	StatusBar    *components.StatusBar
	
	// Data
	Adapters     []caf.Adapter
	Installed    []string
	Available    []string
	
	// UI state
	SelectedAdapter int
	ShowDetail     bool
	DetailAdapter  *caf.Adapter
	InstallError   error
	DetectError    error
	
	// Loading
	Loading       bool
	LoadError     error
}

// AdapterHealth represents the health status of an adapter
type AdapterHealth struct {
	ID       string
	Status   string // "healthy", "degraded", "unhealthy", "unknown"
	Message  string
	LastCheck time.Time
}

// NewAdaptersScreen creates a new adapters screen
func NewAdaptersScreen(repoRoot string, version string) *AdaptersScreen {
	theme := ui.NewTheme()
	keys := ui.NewKeys()
	a11y := ui.NewA11y(ui.NewAccessibilityConfig(), theme, keys)
	
	screen := &AdaptersScreen{
		StateMachine: *ui.NewStateMachine(),
		Version:      version,
		RepoRoot:     repoRoot,
		Theme:        theme,
		Keys:         keys,
		A11y:         a11y,
		Loading:      true,
		SelectedAdapter: 0,
		ShowDetail:     false,
	}
	
	// Initialize status bar
	screen.StatusBar = components.NewStatusBar("adapters")
	screen.StatusBar.SetKeyHints([]string{
		"[i] install", "[c] claude", "[a] auto-detect", "[Enter] detail", "[r] refresh", "[?] help", "[q] quit",
	})
	
	return screen
}

// Init initializes the adapters screen
func (a *AdaptersScreen) Init() tea.Cmd {
	return a.LoadData()
}

// LoadData loads adapters data
func (a *AdaptersScreen) LoadData() tea.Cmd {
	return func() tea.Msg {
	// Get all available adapters
	adapters := caf.ListAdapters()
	var adapterList []caf.Adapter
	var installedList []string
	var availableList []string
	
	for _, id := range adapters {
		adapterObj, ok := caf.GetAdapter(id)
		if ok {
			adapterList = append(adapterList, adapterObj)
			
			// Check if adapter is installed
			installed := a.isAdapterInstalled(id)
			if installed {
				installedList = append(installedList, id)
			} else {
				availableList = append(availableList, id)
			}
		}
	}
		
		return AdaptersDataLoaded{
			Adapters:  adapterList,
			Installed: installedList,
			Available: availableList,
			LoadTime:  time.Now(),
		}
	}
}

// isAdapterInstalled checks if an adapter is installed
func (a *AdaptersScreen) isAdapterInstalled(id string) bool {
	// Check if there's a config for this adapter
	configPath := filepath.Join(a.RepoRoot, ".caf", "adapters", id+".yml")
	if _, err := os.Stat(configPath); err == nil {
		return true
	}
	return false
}

// AdaptersDataLoaded is the message for data loading
type AdaptersDataLoaded struct {
	Adapters  []caf.Adapter
	Installed []string
	Available []string
	LoadTime  time.Time
}

// Update handles messages for the adapters screen
func (a *AdaptersScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKey(msg)
	case AdaptersDataLoaded:
		a.handleDataLoaded(msg)
	case tea.WindowSizeMsg:
		// Handle resize if needed
	}
	
	// Update state machine
	a.StateMachine.Transition(nil)
	
	return a, nil
}

// handleKey handles keyboard input
func (a *AdaptersScreen) handleKey(msg tea.KeyMsg) (ui.Screen, tea.Cmd) {
	// Печатные клавиши — через runeOf: bubbletea кодирует их как KeyRunes (§18.4).
	switch runeOf(msg) {
	case 'i':
		return a, a.installAdapter("opencode")
	case 'c':
		return a, a.installAdapter("claude-code")
	case 'a':
		return a, a.autoDetect()
	case 'r':
		a.Loading = true
		a.LoadError = nil
		// §16.4: Success → Loading на refresh (скелет вместо устаревших данных).
		a.StateMachine.SetState(ui.StateLoading)
		return a, a.LoadData()
	case '?':
		// Show help overlay
		return a, nil
	}

	switch msg.Type {
	case tea.KeyUp:
		if a.SelectedAdapter > 0 {
			a.SelectedAdapter--
		}
	case tea.KeyDown:
		if a.SelectedAdapter < len(a.Adapters)-1 {
			a.SelectedAdapter++
		}
	case tea.KeyEnter:
		// Guard: Enter при пустом списке не должен паниковать (скелет/empty state).
		if a.SelectedAdapter >= 0 && a.SelectedAdapter < len(a.Adapters) {
			a.ShowDetail = true
			a.DetailAdapter = &a.Adapters[a.SelectedAdapter]
			a.StatusBar.SetFocus("detail")
		}
	case tea.KeyEsc:
		if a.ShowDetail {
			a.ShowDetail = false
			a.DetailAdapter = nil
			a.StatusBar.SetFocus("adapters")
		} else {
			return a, tea.Quit
		}
	case tea.KeyCtrlC:
		return a, tea.Quit
	}
	
	return a, nil
}

// handleDataLoaded handles successful data loading
func (a *AdaptersScreen) handleDataLoaded(msg AdaptersDataLoaded) {
	a.Adapters = msg.Adapters
	a.Installed = msg.Installed
	a.Available = msg.Available
	a.Loading = false
	a.LoadError = nil
	
	// Нет зарегистрированных адаптеров → empty-with-CTA (§2.8, INV-PROMPT-MVP-006).
	if len(msg.Adapters) == 0 {
		a.StateMachine.SetState(ui.StateEmpty)
		return
	}
	// Transition to success state
	a.StateMachine.SetState(ui.StateSuccess)
}

// installAdapter installs an adapter
func (a *AdaptersScreen) installAdapter(id string) tea.Cmd {
	return func() tea.Msg {
		// Simulate installation
		time.Sleep(1 * time.Second)
		
		// Check if installation succeeded
		if a.isAdapterInstalled(id) {
			return InstallSuccess{AdapterID: id}
		} else {
			return InstallError{AdapterID: id, Err: fmt.Errorf("failed to install %s", id)}
		}
	}
}

// autoDetect runs auto-detection
func (a *AdaptersScreen) autoDetect() tea.Cmd {
	return func() tea.Msg {
		// Simulate detection
		time.Sleep(2 * time.Second)
		
		var detected []string
		for _, adapter := range a.Adapters {
			// Simulate detection success for some adapters
			if adapter.ID() == "opencode" || adapter.ID() == "claude-code" {
				detected = append(detected, adapter.ID())
			}
		}
		
		return AutoDetectComplete{Detected: detected}
	}
}

// InstallSuccess is the message for successful installation
type InstallSuccess struct {
	AdapterID string
}

// InstallError is the message for failed installation
type InstallError struct {
	AdapterID string
	Err       error
}

// AutoDetectComplete is the message for auto-detection completion
type AutoDetectComplete struct {
	Detected []string
}

// View returns the adapters view
func (a *AdaptersScreen) View() string {
	// Update status bar
	a.StatusBar.SetRefreshCompleted()
	
	// Render based on current state
	switch a.StateMachine.Current() {
	case ui.StateLoading:
		return a.renderSkeleton()
	case ui.StateError:
		return a.renderError()
	case ui.StateEmpty:
		return a.renderEmpty()
	case ui.StatePartial:
		return a.renderPartial()
	case ui.StateSuccess:
		if a.ShowDetail {
			return a.renderDetail()
		}
		return a.renderSuccess()
	default:
		return a.renderSkeleton()
	}
}

// renderSkeleton renders the loading skeleton
func (a *AdaptersScreen) renderSkeleton() string {
	// Render skeleton table
	skeleton := "Adapters — capability matrix\n"
	skeleton += "ID              TIER   pre-session  post-edit  pre-commit  MCP   HEALTH\n"
	skeleton += strings.Repeat("─", 70) + "\n"
	skeleton += strings.Repeat("┌─────────────────────────────────────────────────────────────────────┐\n", 5)
	skeleton += strings.Repeat("│                                                               │\n", 5)
	skeleton += strings.Repeat("└─────────────────────────────────────────────────────────────────────┘\n", 5)
	
	return components.Header(a.Version, "adapters") + "\n" + skeleton + "\n" + a.StatusBar.Render()
}

// renderError renders the error state
func (a *AdaptersScreen) renderError() string {
	errorBox := components.NewErrorBox(fmt.Sprintf("Failed to load adapters: %v", a.LoadError), a.Theme, a.A11y)
	return components.Header(a.Version, "adapters") + "\n" + errorBox.View() + "\n" + a.StatusBar.Render()
}

// renderEmpty renders the empty state
func (a *AdaptersScreen) renderEmpty() string {
	emptyCTA := components.NewEmptyCTA("No adapters available", "Use [i]/[c]/[a] to install adapters", "Run adapter installation", "Press [i] to install now", a.Theme)
	return components.Header(a.Version, "adapters") + "\n" + emptyCTA.View() + "\n" + a.StatusBar.Render()
}

// renderPartial renders the partial state
func (a *AdaptersScreen) renderPartial() string {
	partialBanner := components.NewPartialBanner(1, 2, []string{"universal-lsp"}, a.Theme)
	return components.Header(a.Version, "adapters") + "\n" + partialBanner.View() + "\n" + a.tableBody() + "\n" + a.StatusBar.Render()
}

// renderSuccess renders the success state with actual data
func (a *AdaptersScreen) renderSuccess() string {
	return components.Header(a.Version, "adapters") + "\n" + a.tableBody() + "\n" + a.StatusBar.Render()
}

// tableBody — таблица capability matrix без header/statusbar
// (используется и в renderPartial, чтобы не дублировать «священную» обвязку §15.2).
func (a *AdaptersScreen) tableBody() string {
	
	// Render table header
	header := "Adapters — capability matrix\n"
	header += "ID              TIER   pre-session  post-edit  pre-commit  MCP   HEALTH\n"
	header += strings.Repeat("─", 70) + "\n"
	
	// Render adapter rows
	rows := ""
	for i, adapter := range a.Adapters {
		// Add focus indicator
		prefix := "  "
		if i == a.SelectedAdapter {
			prefix = "▸ "
		}
		
	// Get adapter info
	id := adapter.ID()
	tier := a.getTierLabel(&adapter)
	preSession := a.getCapabilityLabel(&adapter, "pre-session")
	postEdit := a.getCapabilityLabel(&adapter, "post-edit")
	preCommit := a.getCapabilityLabel(&adapter, "pre-commit")
	mcp := a.getCapabilityLabel(&adapter, "mcp")
	health := a.getHealthStatus(&adapter)
		
		row := fmt.Sprintf("%s%-14s %-6s %-11s %-10s %-10s %-5s %s", 
			prefix, id, tier, preSession, postEdit, preCommit, mcp, health)
		rows += row + "\n"
	}
	
	return header + rows
}

// renderDetail renders the detail view for a selected adapter
func (a *AdaptersScreen) renderDetail() string {
	if a.DetailAdapter == nil {
		return a.renderSuccess()
	}
	
	adapter := a.DetailAdapter
	
	detail := fmt.Sprintf("Adapter Details: %s\n", (*adapter).ID())
	detail += strings.Repeat("─", 40) + "\n"
	detail += fmt.Sprintf("Tier: %s\n", a.getTierLabel(adapter))
	detail += fmt.Sprintf("Capabilities:\n")
	detail += fmt.Sprintf("  pre-session: %s\n", a.getCapabilityLabel(adapter, "pre-session"))
	detail += fmt.Sprintf("  post-edit: %s\n", a.getCapabilityLabel(adapter, "post-edit"))
	detail += fmt.Sprintf("  pre-commit: %s\n", a.getCapabilityLabel(adapter, "pre-commit"))
	detail += fmt.Sprintf("  MCP support: %s\n", a.getCapabilityLabel(adapter, "mcp"))
	detail += fmt.Sprintf("Health: %s\n", a.getHealthStatus(adapter))
	detail += "\nPress [Esc] to return to list\n"
	
	return components.Header(a.Version, "adapters") + "\n" + detail + "\n" + a.StatusBar.Render()
}

// getTierLabel returns the tier label for an adapter
func (a *AdaptersScreen) getTierLabel(adapter *caf.Adapter) string {
	// Simulate tier detection
	switch (*adapter).ID() {
	case "opencode", "claude-code":
		return "T1"
	case "universal-lsp":
		return "T4"
	default:
		return "T2"
	}
}

// getCapabilityLabel returns the capability label for an adapter
func (a *AdaptersScreen) getCapabilityLabel(adapter *caf.Adapter, capability string) string {
	// Simulate capability detection
	switch (*adapter).ID() {
	case "opencode", "claude-code":
		return "native"
	case "universal-lsp":
		switch capability {
		case "pre-session":
			return "—"
		case "post-edit":
			return "watcher"
		case "pre-commit":
			return "universal"
		case "mcp":
			return "no"
		}
	}
	return "—"
}

// getHealthStatus returns the health status for an adapter
func (a *AdaptersScreen) getHealthStatus(adapter *caf.Adapter) string {
	// Simulate health check
	if (*adapter).ID() == "claude-code" {
		return "⚠ degraded (hooks not wired)"
	}
	return "✓ healthy"
}

// GetID returns the screen ID
func (a *AdaptersScreen) GetID() string {
	return "adapters"
}

// IsActive returns if the screen is active
func (a *AdaptersScreen) IsActive() bool {
	return true
}

// SetActive sets the active state
func (a *AdaptersScreen) SetActive(active bool) {
	// Update status bar focus when active
	if active {
		a.StatusBar.SetFocus("adapters")
	}
}

// SetKeyKeys sets keyboard hints (for compatibility)
func (a *AdaptersScreen) SetKeyKeys(hints []string) {
	a.StatusBar.SetKeyHints(hints)
}


