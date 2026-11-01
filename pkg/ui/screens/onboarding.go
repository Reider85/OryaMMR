// onboarding.go — TUI-экран онбординг-визарда (PROMPT-MVP §4, §2.7).
package screens

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui/components"
)

// OnboardingScreen represents the onboarding wizard screen
type OnboardingScreen struct {
	ui.StateMachine
	
	Version      string
	RepoRoot     string
	Theme        *ui.Theme
	Keys         *ui.Keys
	A11y         *ui.A11y
	StatusBar    *components.StatusBar
	
	// Wizard state
	CurrentStep  int
	TotalSteps   int
	
	// Step 1: Welcome
	WelcomeAccepted bool
	
	// Step 2: Detection
	DetectedAdapters []string
	DetectionError   error
	DetectionSpinner *components.Spinner
	
	// Step 3: Installation
	InstallingAdapters map[string]bool
	InstallErrors      map[string]error
	InstallSuccess     []string
	
	// Step 4: Summary
	SummaryComplete   bool
	SummaryMessage   string
	
	// Loading states
	Loading          bool
	LoadError        error

	// Help overlay (§18.4: ? показывает help на каждом экране)
	ShowHelp bool
}

// Step represents a wizard step
type Step struct {
	ID      string
	Title   string
	Content string
	Actions []StepAction
}

// StepAction represents an action for a wizard step
type StepAction struct {
	Key     string
	Label   string
	Action  func() tea.Cmd
	Default bool
}

// NewOnboardingScreen creates a new onboarding screen
func NewOnboardingScreen(repoRoot string, version string) *OnboardingScreen {
	theme := ui.NewTheme()
	keys := ui.NewKeys()
	a11y := ui.NewA11y(ui.NewAccessibilityConfig(), theme, keys)
	
	screen := &OnboardingScreen{
		StateMachine: *ui.NewStateMachine(),
		Version:      version,
		RepoRoot:     repoRoot,
		Theme:        theme,
		Keys:         keys,
		A11y:         a11y,
		CurrentStep:  0,
		TotalSteps:   4,
		Loading:      true,
		
		// Initialize spinner for detection
		DetectionSpinner: components.NewSpinner(),
		
		// Initialize installation tracking
		InstallingAdapters: make(map[string]bool),
		InstallErrors:      make(map[string]error),
		InstallSuccess:     make([]string, 0),
	}
	
	// Initialize status bar
	screen.StatusBar = components.NewStatusBar("onboarding")
	screen.StatusBar.SetKeyHints([]string{
		"[Enter] next", "[Esc] back", "[q] abort", "[?] help",
	})
	
	return screen
}

// Init initializes the onboarding screen
func (o *OnboardingScreen) Init() tea.Cmd {
	return o.StartStep(1)
}

// StartStep starts a specific wizard step
func (o *OnboardingScreen) StartStep(step int) tea.Cmd {
	o.CurrentStep = step
	o.Loading = true

	switch step {
	case 1: // Welcome
		// Welcome не требует загрузки — сразу success (§16.5).
		o.StateMachine.SetState(ui.StateSuccess)
		return nil
	case 2: // Detection
		o.StateMachine.SetState(ui.StateLoading)
		o.DetectionSpinner.Start()
		return o.DetectAdapters()
	case 3: // Installation
		o.StateMachine.SetState(ui.StateLoading)
		return o.InstallAdapters()
	case 4: // Summary
		o.StateMachine.SetState(ui.StateLoading)
		return o.GenerateSummary()
	default:
		return nil
	}
}

// DetectAdapters runs adapter detection
func (o *OnboardingScreen) DetectAdapters() tea.Cmd {
	return func() tea.Msg {
		// Simulate detection
		time.Sleep(2 * time.Second)
		
		var detected []string
		adapters := caf.ListAdapters()
		
		for _, id := range adapters {
			// Simulate detection success for some adapters
			if id == "opencode" || id == "claude-code" || id == "universal-lsp" {
				detected = append(detected, id)
			}
		}
		
		return DetectionComplete{Detected: detected}
	}
}

// InstallAdapters installs detected adapters
func (o *OnboardingScreen) InstallAdapters() tea.Cmd {
	return func() tea.Msg {
		// Simulate installation of all detected adapters
		time.Sleep(3 * time.Second)
		
		var success []string
		var errors map[string]error = make(map[string]error)
		
		// Simulate installation results
		for _, adapterID := range o.DetectedAdapters {
			if adapterID == "opencode" || adapterID == "claude-code" {
				success = append(success, adapterID)
			} else {
				errors[adapterID] = fmt.Errorf("installation failed")
			}
		}
		
		return InstallationComplete{
			Success: success,
			Errors:  errors,
		}
	}
}

// GenerateSummary generates the final summary
func (o *OnboardingScreen) GenerateSummary() tea.Cmd {
	return func() tea.Msg {
		// Generate summary based on installation results
		var message string
		if len(o.InstallSuccess) > 0 {
			message = fmt.Sprintf("Setup complete. %d adapters installed.", len(o.InstallSuccess))
		} else {
			message = "Setup complete with some warnings."
		}
		
		return SummaryComplete{Message: message}
	}
}

// DetectionComplete is the message for detection completion
type DetectionComplete struct {
	Detected []string
}

// InstallationComplete is the message for installation completion
type InstallationComplete struct {
	Success []string
	Errors  map[string]error
}

// SummaryComplete is the message for summary generation
type SummaryComplete struct {
	Message string
}

// Update handles messages for the onboarding screen
func (o *OnboardingScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return o.handleKey(msg)
	case DetectionComplete:
		o.handleDetectionComplete(msg)
	case InstallationComplete:
		o.handleInstallationComplete(msg)
	case SummaryComplete:
		o.handleSummaryComplete(msg)
	case tea.WindowSizeMsg:
		// Handle resize if needed
	}
	
	// Состояние устанавливают сами хендлеры (Success/Empty/Partial/Error);
	// безусловный SetState(StateSuccess) здесь затирал бы Empty/Partial (INV-PROMPT-MVP-006).
	return o, nil
}

// handleKey handles keyboard input
func (o *OnboardingScreen) handleKey(msg tea.KeyMsg) (ui.Screen, tea.Cmd) {
	// Help-open режим: ?/Esc закрывают оверлей, q/Ctrl+C выходят, остальные
	// клавиши не действуют, пока открыт help (нет магии под оверлеем).
	if o.ShowHelp {
		switch {
		case runeOf(msg) == '?', msg.Type == tea.KeyEsc:
			o.ShowHelp = false
			return o, nil
		case runeOf(msg) == 'q', msg.Type == tea.KeyCtrlC:
			return o, tea.Quit
		}
		return o, nil
	}

	// Печатные клавиши — через runeOf: bubbletea кодирует их как KeyRunes (§18.4).
	switch r := runeOf(msg); r {
	case 'q':
		return o, tea.Quit
	case '?':
		o.ShowHelp = true
		return o, nil
	case 'r':
		// ErrorBox обещает "[r] retry" (INV-UI-010) — retry текущего шага.
		return o, o.StartStep(o.CurrentStep)
	case 'i', 'c':
		// [i]/[c] подсказаны на шаге детекции и в empty-CTA (§2.7):
		// выбор агента для установки, дальше — симуляция установки (шаг 3).
		if o.CurrentStep != 2 && o.StateMachine.Current() != ui.StateEmpty {
			return o, nil
		}
		id := "opencode"
		if r == 'c' {
			id = "claude-code"
		}
		return o, o.startInstallWith(id)
	}

	switch msg.Type {
	case tea.KeyEnter:
		if o.CurrentStep < o.TotalSteps {
			return o, o.StartStep(o.CurrentStep + 1)
		}
	case tea.KeyEsc:
		if o.CurrentStep > 1 {
			return o, o.StartStep(o.CurrentStep - 1)
		} else {
			// Abort on first step
			return o, tea.Quit
		}
	case tea.KeyCtrlC:
		return o, tea.Quit
	}
	
	return o, nil
}

// startInstallWith — [i]/[c] из empty-CTA или шага детекции: фиксирует выбор
// агента и запускает установку (шаг 3). Empty → Loading разрешён (§16.4).
func (o *OnboardingScreen) startInstallWith(id string) tea.Cmd {
	o.DetectedAdapters = []string{id}
	o.Loading = false
	return o.StartStep(3)
}

// handleDetectionComplete handles detection completion
func (o *OnboardingScreen) handleDetectionComplete(msg DetectionComplete) {
	o.DetectedAdapters = msg.Detected
	o.DetectionError = nil
	o.Loading = false
	o.DetectionSpinner.Stop()
	
	// 0 найденных агентов → empty-with-CTA (§2.7, INV-PROMPT-MVP-006):
	// пользователь получает варианты восстановления, а не пустой экран.
	if len(msg.Detected) == 0 {
		o.StateMachine.SetState(ui.StateEmpty)
		return
	}
	o.StateMachine.SetState(ui.StateSuccess)
}

// handleInstallationComplete handles installation completion
func (o *OnboardingScreen) handleInstallationComplete(msg InstallationComplete) {
	o.InstallSuccess = msg.Success
	o.InstallErrors = msg.Errors
	o.Loading = false
	
	// Обновляем трекинг установок (msg.Success уже записан выше — без повторного append).
	for _, id := range msg.Success {
		o.InstallingAdapters[id] = false
	}
	
	// Частичный успех: часть адаптеров встала, часть нет → partial-failure (INV-PROMPT-MVP-006).
	if len(msg.Errors) > 0 {
		o.StateMachine.SetState(ui.StatePartial)
		return
	}
	o.StateMachine.SetState(ui.StateSuccess)
}

// handleSummaryComplete handles summary generation
func (o *OnboardingScreen) handleSummaryComplete(msg SummaryComplete) {
	o.SummaryMessage = msg.Message
	o.SummaryComplete = true
	o.Loading = false
	
	// Transition to success state
	o.StateMachine.SetState(ui.StateSuccess)
}

// View returns the onboarding view
func (o *OnboardingScreen) View() string {
	// Update status bar
	o.StatusBar.SetRefreshCompleted()
	
	// Help overlay (§18.4): ? на каждом экране показывает все клавиши.
	// Header и StatusBar — «священные» (§15.2), оверлей заменяет только контент.
	if o.ShowHelp {
		overlay := components.NewHelpOverlay("onboarding", o.helpSections(), o.Theme)
		return components.Header(o.Version, "onboarding") + "\n" + overlay.View() + "\n" + o.StatusBar.Render()
	}
	
	// 5-state routing (INV-PROMPT-MVP-006): error/empty/partial перекрывают
	// пошаговый рендер; loading/success идут по CurrentStep (шаги сами
	// показывают спиннер детекции — §16.5).
	switch o.StateMachine.Current() {
	case ui.StateError:
		return o.renderErrorState()
	case ui.StateEmpty:
		return o.renderEmptyState()
	case ui.StatePartial:
		return o.renderPartialState()
	}
	
	// Render based on current step and state
	switch o.CurrentStep {
	case 1:
		return o.renderWelcome()
	case 2:
		return o.renderDetection()
	case 3:
		return o.renderInstallation()
	case 4:
		return o.renderSummary()
	default:
		return o.renderWelcome()
	}
}

// renderErrorState: ErrorBox с recovery-path (§16.3, INV-UI-010).
func (o *OnboardingScreen) renderErrorState() string {
	err := o.LoadError
	if err == nil {
		err = o.DetectionError
	}
	if err == nil {
		err = fmt.Errorf("onboarding step failed")
	}
	errorBox := components.NewErrorBox(fmt.Sprintf("Onboarding failed: %v", err), o.Theme, o.A11y)
	return components.Header(o.Version, "onboarding") + "\n" + errorBox.View() + "\n" + o.StatusBar.Render()
}

// renderEmptyState: пустая детекция → EmptyCTA с вариантами восстановления (§2.7).
func (o *OnboardingScreen) renderEmptyState() string {
	emptyCTA := components.NewEmptyCTA(
		"No agents detected",
		"Automatic detection found no supported AI agents",
		"Install an agent or continue with universal LSP fallback",
		"[i] install opencode  [c] install Claude Code  [Enter] continue  [Esc] back",
		o.Theme,
	)
	return components.Header(o.Version, "onboarding") + "\n" + emptyCTA.View() + "\n" + o.StatusBar.Render()
}

// renderPartialState: часть адаптеров встала, часть нет → PartialBanner + шаг (§16.3).
func (o *OnboardingScreen) renderPartialState() string {
	missing := make([]string, 0, len(o.InstallErrors))
	for id := range o.InstallErrors {
		missing = append(missing, id)
	}
	sort.Strings(missing) // детерминированный порядок при итерации по map
	banner := components.NewPartialBanner(len(o.InstallSuccess), len(o.InstallSuccess)+len(missing), missing, o.Theme)
	return components.Header(o.Version, "onboarding") + "\n" + banner.View() + "\n" + o.installationBody() + "\n" + o.StatusBar.Render()
}

// renderWelcome renders the welcome step
func (o *OnboardingScreen) renderWelcome() string {
	
	welcome := fmt.Sprintf("▸ Step %d/%d: Welcome\n", 1, o.TotalSteps)
	welcome += strings.Repeat("─", 30) + "\n"
	welcome += "Welcome to CodeAuthorityFirst!\n\n"
	welcome += "This wizard will help you set up your\n"
	welcome += "development environment with AI agents\n"
	welcome += "and code quality tools.\n\n"
	welcome += "Press [Enter] to continue, [q] to abort.\n"
	
	return components.Header(o.Version, "onboarding") + "\n" + welcome + "\n" + o.StatusBar.Render()
}

// renderDetection renders the detection step
func (o *OnboardingScreen) renderDetection() string {
	
	detection := fmt.Sprintf("▸ Step %d/%d: Detection\n", 2, o.TotalSteps)
	detection += strings.Repeat("─", 30) + "\n"
	detection += "Detecting available agents...\n\n"
	
	if o.Loading {
		detection += fmt.Sprintf("%s Detecting agents...\n", o.DetectionSpinner.Render())
	} else {
		detection += "Detection results:\n"
		for _, adapterID := range o.DetectedAdapters {
			detection += fmt.Sprintf(" ✓ %s detected\n", adapterID)
		}
		if len(o.DetectedAdapters) == 0 {
			detection += " No agents detected\n"
		}
	}
	
	detection += "\nNo agents detected? You can:\n"
	detection += "  [i] install opencode\n"
	detection += "  [c] install Claude Code\n"
	detection += "  [Enter] LSP-fallback only\n"
	detection += "  [Esc] back\n"
	
	return components.Header(o.Version, "onboarding") + "\n" + detection + "\n" + o.StatusBar.Render()
}

// renderInstallation renders the installation step
func (o *OnboardingScreen) renderInstallation() string {
	return components.Header(o.Version, "onboarding") + "\n" + o.installationBody() + "\n" + o.StatusBar.Render()
}

// installationBody — содержимое шага 3 без header/statusbar
// (используется и в renderPartialState, чтобы не дублировать обвязку).
func (o *OnboardingScreen) installationBody() string {
	installation := fmt.Sprintf("▸ Step %d/%d: Installation\n", 3, o.TotalSteps)
	installation += strings.Repeat("─", 30) + "\n"
	installation += "Installing detected agents...\n\n"
	
	if o.Loading {
		installation += "Installing adapters...\n"
		for _, adapterID := range o.DetectedAdapters {
			if installing, exists := o.InstallingAdapters[adapterID]; exists && installing {
				installation += fmt.Sprintf(" %s %s installing...\n", components.NewSpinner().Render(), adapterID)
			}
		}
	} else {
		installation += "Installation results:\n"
		for _, id := range o.InstallSuccess {
			installation += fmt.Sprintf(" ✓ %s installed\n", id)
		}
		// Итерация по map — сортируем ключи, иначе порядок строк не детерминирован.
		errIDs := make([]string, 0, len(o.InstallErrors))
		for id := range o.InstallErrors {
			errIDs = append(errIDs, id)
		}
		sort.Strings(errIDs)
		for _, id := range errIDs {
			installation += fmt.Sprintf(" ✗ %s failed: %v\n", id, o.InstallErrors[id])
		}
	}
	
	installation += "\nPress [Enter] to continue, [Esc] back\n"
	
	return installation
}

// renderSummary renders the summary step
func (o *OnboardingScreen) renderSummary() string {
	
	summary := fmt.Sprintf("▸ Step %d/%d: Summary\n", 4, o.TotalSteps)
	summary += strings.Repeat("─", 30) + "\n"
	summary += o.SummaryMessage + "\n\n"
	
	if len(o.InstallSuccess) > 0 {
		summary += "Installed adapters:\n"
		for _, id := range o.InstallSuccess {
			summary += fmt.Sprintf("  • %s\n", id)
		}
	}
	
	summary += "\nNext steps:\n"
	summary += "  • Run `caf dashboard` to view your workspace\n"
	summary += "  • Use `caf doctor` to check health\n"
	summary += "  • See `caf help` for more commands\n\n"
	summary += "Press [Enter] to complete setup\n"
	
	return components.Header(o.Version, "onboarding") + "\n" + summary + "\n" + o.StatusBar.Render()
}

// helpSections — все живые клавиши визарда для help-оверлея (§18.4).
// Перечисляются только реально обрабатываемые клавиши — help не врёт (INV-UI-009).
func (o *OnboardingScreen) helpSections() []components.HelpSection {
	return []components.HelpSection{
		{Title: "Navigation", Entries: []components.HelpEntry{
			{Key: "Enter", Desc: "next step"},
			{Key: "Esc", Desc: "back (quit on welcome)"},
		}},
		{Title: "Actions", Entries: []components.HelpEntry{
			{Key: "r", Desc: "retry current step"},
			{Key: "q", Desc: "quit"},
			{Key: "?", Desc: "toggle this help"},
		}},
		{Title: "Screen-specific", Entries: []components.HelpEntry{
			{Key: "i", Desc: "install opencode (detection step)"},
			{Key: "c", Desc: "install Claude Code (detection step)"},
		}},
	}
}

// GetID returns the screen ID
func (o *OnboardingScreen) GetID() string {
	return "onboarding"
}

// IsActive returns if the screen is active
func (o *OnboardingScreen) IsActive() bool {
	return true
}

// SetActive sets the active state
func (o *OnboardingScreen) SetActive(active bool) {
	// Update status bar focus when active
	if active {
		o.StatusBar.SetFocus("onboarding")
	}
}

// SetKeyKeys sets keyboard hints (for compatibility)
func (o *OnboardingScreen) SetKeyKeys(hints []string) {
	o.StatusBar.SetKeyHints(hints)
}
