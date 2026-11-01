// Package main — точка входа caf: cobra CLI skeleton (PROMPT-MVP.md §4, §14 step 4).
// Язык конвенции: русские комментарии, English имена (PROMPT-MVP.md, Language).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/adapters"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/doctor"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/hooks"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/storage"
)

// version — релизная версия MVP-сборки (тег v0.4.1-mvp.1, PROMPT-MVP §14 step 18).
const version = "0.4.1-mvp.1"

// rootCmd — базовая команда при вызове без подкоманд.
var rootCmd = &cobra.Command{
	Use:     "caf",
	Short:   "CodeAuthorityFirst — AI code authority tracking",
	Long:    `CodeAuthorityFirst (caf) tracks AI coding sessions, provenance, and decision trails across repositories.`,
	Version: version,
}

// initCmd — caf init: создаёт .caf/, ставит universal pre-commit hook, опционально адаптеры.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize CodeAuthorityFirst in a repository",
	Long: `Initialize CodeAuthorityFirst in a repository by:
- Creating the .caf/ directory structure (ARCHITECT.md §2.1)
- Installing the universal pre-commit hook (FR-ADP-003, fail-safe per INV-ARCH-004)
- Optionally detecting and installing adapters (FR-ADP-005)

Example:
  caf init
  caf init --auto-detect
  caf init --adapters opencode,claude-code`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := resolveRepo(flagString(cmd, "repo"))
		if err != nil {
			return err
		}
		
		if flagBool(cmd, "tui") {
			return runInitTUI(repoRoot)
		}
		
		return runInit(repoRoot, flagBool(cmd, "auto-detect"), flagString(cmd, "adapters"))
	},
}

// hookCmd — caf hook <event>: диспетчер pre-session / post-edit / pre-commit.
// Вызывается адаптерами и universal pre-commit hook. Всегда exit 0 при ошибке
// выполнения (fail-safe: warn-and-continue, INV-ARCH-004) — хук не блокирует коммиты.
var hookCmd = &cobra.Command{
	Use:   "hook <event>",
	Short: "Execute a hook event (pre-session, post-edit, pre-commit)",
	Long: `Execute a hook event manually. Usually invoked by AI adapters or the
universal pre-commit hook installed by caf init.

Fail-safe contract (INV-ARCH-004): hook execution errors are reported as
warnings and never block a session or commit.

Events: pre-session, post-edit, pre-commit

Example:
  caf hook pre-session --session 2026-10-24-a1b2c3
  caf hook post-edit --session 2026-10-24-a1b2c3 --files "main.go" --message "added middleware"
  caf hook pre-commit --repo "$(pwd)"
  echo '{"tool_name":"Bash","tool_input":{"command":"git commit -m x"}}' | caf hook pre-commit --from-agent`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		event := args[0]
		repoRoot := flagString(cmd, "repo")
		if repoRoot == "" {
			repoRoot = "."
		}
		sessionID := flagString(cmd, "session")
		files := flagStrings(cmd, "files")
		message := flagString(cmd, "message")
		fromAgent := flagBool(cmd, "from-agent")
		os.Exit(runHookEvent(event, repoRoot, sessionID, files, message, fromAgent))
	},
}

// whoCmd — caf who <file:line>: какая сессия трогала файл (FR-PROV-003, MVP — уровень файла).
var whoCmd = &cobra.Command{
	Use:   "who <file[:line]>",
	Short: "Find which session modified a file",
	Long: `Find which AI session modified a specific file (and, if given, a line).

MVP matches at file level via .caf/sessions/*.yml (files_touched, iterations).
Line-level attribution ships in ALPHA (ARD FR-PROV-003 line map).

Example:
  caf who main.go
  caf who internal/api/policies.go:42`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := resolveRepo(flagString(cmd, "repo"))
		if err != nil {
			return err
		}
		return runWho(args[0], repoRoot)
	},
}

// whenCmd — caf when --pattern: ALPHA scope; в MVP честно сообщает об отложении (§2.1).
var whenCmd = &cobra.Command{
	Use:   "when --pattern <glob>",
	Short: "Find when a code pattern first appeared (ALPHA scope)",
	Long: `Find when a specific code pattern first appeared in the repository.

This command is deferred to ALPHA per PROMPT-MVP.md §2.1 and ROADMAP.md.
MVP acknowledges the request and exits without fabricated data.

Example:
  caf when --pattern "func main"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWhen(flagString(cmd, "pattern"))
	},
}

// contextCmd — caf context [--refresh]: генерация .caf/context.md из сессий (§2.1).
var contextCmd = &cobra.Command{
	Use:   "context [repo]",
	Short: "Generate .caf/context.md from recent sessions",
	Long: `Generate a context.md file from recent sessions and decisions so AI agents
can pick up recent work. Without --refresh, shows the current context status;
with --refresh, regenerates the file.

Example:
  caf context --refresh
  caf context --refresh /path/to/repo`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot := "."
		if len(args) > 0 {
			repoRoot = args[0]
		}
		abs, err := resolveRepo(repoRoot)
		if err != nil {
			return err
		}
		return runContext(abs, flagBool(cmd, "refresh"))
	},
}

// dashboardCmd — caf dashboard: TUI-хаб (FR-UI-001).
var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch the TUI dashboard",
	Long: `Launch the TUI dashboard: today queue, authorship score, active agents,
recent decisions, cross-repo status.

Example:
  caf dashboard
  caf dashboard --tui`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot := flagString(cmd, "repo")
		if repoRoot == "" {
			repoRoot = "."
		}
		abs, err := resolveRepo(repoRoot)
		if err != nil {
			return err
		}
		
		// Check for TUI flag
		if tui, _ := cmd.Flags().GetBool("tui"); tui {
			return runDashboardTUI(abs)
		}
		
		return runDashboard(abs)
	},
}

func init() {
	// Add TUI flag to dashboard command
	dashboardCmd.Flags().BoolP("tui", "t", false, "Launch TUI dashboard")
}

// adaptersCmd — родительская команда для адаптеров (FR-ADP-001/004/005).
var adaptersCmd = &cobra.Command{
	Use:   "adapters",
	Short: "Inspect installed and available adapters",
	Long:  `Inspect the adapter registry: Tier-1 (opencode, Claude Code) and Tier-4 (universal LSP fallback).`,
}

// adaptersListCmd — caf adapters list: capability matrix + статус (UI.md §7, текстовая версия).
var adaptersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List adapters with capability matrix and detection status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagBool(cmd, "tui") {
			repoRoot := "."
			abs, err := resolveRepo(repoRoot)
			if err != nil {
				return err
			}
			return runAdaptersTUI(abs)
		}
		return runAdaptersList()
	},
}

func init() {
	adaptersListCmd.Flags().Bool("tui", false, "Launch TUI adapter manager")
}

// doctorCmd — caf doctor: 5 базовых health checks (FR-OBS-004 partial).
var doctorCmd = &cobra.Command{
	Use:   "doctor [repo]",
	Short: "Run basic health checks (5 checks)",
	Long: `Run health checks: .caf/ validity, config schema, sessions present,
hook installed, LSP responds. Full 11-check version is ALPHA (OBS-004).

Example:
  caf doctor
  caf doctor /path/to/repo`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot := "."
		if len(args) > 0 {
			repoRoot = args[0]
		}
		abs, err := resolveRepo(repoRoot)
		if err != nil {
			return err
		}
		return runDoctor(abs)
	},
}

// autoDetectCmd — caf auto-detect: скан машины на известные агенты (FR-ADP-005, <5s).
var autoDetectCmd = &cobra.Command{
	Use:   "auto-detect",
	Short: "Auto-detect installed AI agents",
	Long: `Scan this machine for known AI agents and report which adapters would be
active. Detection is machine-wide; no repository is required.

Example:
  caf auto-detect`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		runAutoDetect()
		return nil
	},
}

// commandsRegistered — защита от повторной регистрации (main() и тесты).
var commandsRegistered bool

func main() {
	setupCommands()

	if err := rootCmd.Execute(); err != nil {
		// Ошибки использования/RunE — ненулевой exit; hook-команда сюда не попадает (см. Run).
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// setupCommands регистрирует подкоманды и флаги. Выделено из main() ради
// покрытия (§10): тесты вызывают setupCommands() + rootCmd.SetArgs/Execute
// напрямую, не запуская os.Exit.
func setupCommands() {
	if commandsRegistered {
		return
	}
	commandsRegistered = true

	// Один вызов регистрирует все встроенные адаптеры (Tier 1 + Tier 4).
	adapters.Initialize()

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(hookCmd)
	rootCmd.AddCommand(whoCmd)
	rootCmd.AddCommand(whenCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(autoDetectCmd)
	adaptersCmd.AddCommand(adaptersListCmd)
	rootCmd.AddCommand(adaptersCmd)

	initCmd.Flags().StringP("repo", "r", "", "Repository path (default: current directory)")
	initCmd.Flags().Bool("auto-detect", false, "Detect installed agents and report recommended adapter")
	initCmd.Flags().String("adapters", "", "Comma-separated adapter IDs to install (e.g. opencode,claude-code)")
	initCmd.Flags().Bool("tui", false, "Launch interactive TUI onboarding wizard")

	hookCmd.Flags().String("repo", "", "Repository path (default: current directory)")
	hookCmd.Flags().String("session", "", "Session ID")
	hookCmd.Flags().StringSlice("files", nil, "Files changed (comma-separated or repeated)")
	hookCmd.Flags().String("message", "", "Commit message / change summary")
	hookCmd.Flags().Bool("from-agent", false,
		"Read agent hook JSON on stdin (Claude Code PreToolUse) and run pre-commit only for git commit")

	whoCmd.Flags().StringP("repo", "r", "", "Repository path (default: current directory)")

	whenCmd.Flags().String("pattern", "", "Pattern to search for (required)")
	_ = whenCmd.MarkFlagRequired("pattern")

	contextCmd.Flags().Bool("refresh", false, "Regenerate context.md from sessions")
}

// --- helpers: флаги и пути ---

func flagString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func flagBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

func flagStrings(cmd *cobra.Command, name string) []string {
	v, _ := cmd.Flags().GetStringSlice(name)
	return v
}

// resolveRepo приводит путь репозитория к абсолютному.
func resolveRepo(repo string) (string, error) {
	if repo == "" {
		repo = "."
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return "", fmt.Errorf("invalid repository path %q: %w", repo, err)
	}
	return abs, nil
}

// trackableAdapters возвращает отсортированные ID трекаемых адаптеров,
// исключая языковые (LanguageAdapter) — они регистрируются в тот же реестр,
// но не обрабатывают события хуков.
func trackableAdapters() []caf.Adapter {
	var out []caf.Adapter
	for _, id := range caf.ListAdapters() {
		adapter, ok := caf.GetAdapter(id)
		if !ok {
			continue
		}
		if _, isLang := adapter.(caf.LanguageAdapter); isLang {
			continue
		}
		out = append(out, adapter)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// formatTier бейдж тира: T1 (native) / T4 (fallback) (UI.md §7).
func formatTier(tier int) string {
	switch tier {
	case 1:
		return "T1 (native)"
	case 4:
		return "T4 (fallback)"
	default:
		return fmt.Sprintf("T%d", tier)
	}
}

// formatCell подпись capability-ячейки — всегда текст, никогда только цвет (INV-UI-008).
func formatCell(t caf.EventTier) string {
	return string(t)
}

// --- команды ---

// runInit: .caf/ + universal hook + опционально адаптеры (§2.1, §2.2, §2.5).
func runInit(repoRoot string, autoDetect bool, adaptersFlag string) error {
	fmt.Printf("Initializing CodeAuthorityFirst in %s\n", repoRoot)

	if err := storage.InitializeCAF(repoRoot); err != nil {
		return fmt.Errorf("failed to initialize .caf/: %w", err)
	}
	fmt.Println("OK: .caf/ directory structure created")

	if err := hooks.InstallUniversalHook(repoRoot); err != nil {
		// Хук не критичен для init: предупреждаем и продолжаем (fail-safe дух).
		fmt.Printf("WARN: failed to install universal pre-commit hook: %v\n", err)
	} else {
		fmt.Println("OK: universal pre-commit hook installed at .git/hooks/pre-commit")
	}

	if adaptersFlag != "" {
		for _, id := range strings.Split(adaptersFlag, ",") {
			installAdapter(strings.TrimSpace(id))
		}
	}

	if autoDetect {
		detected := reportDetection()
		if rec := recommendedAdapter(detected); rec != "" {
			fmt.Printf("Recommended adapter: %s (install hooks with: caf init --adapters %s)\n", rec, rec)
		} else {
			fmt.Println("No adapters detected; universal LSP fallback stays available via caf dashboard.")
		}
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. caf dashboard  — open the TUI hub")
	fmt.Println("  2. caf doctor     — run health checks")
	fmt.Println("  3. use an AI agent (opencode, Claude Code) — sessions start appearing")
	return nil
}

// installAdapter ставит один адаптер по ID; ошибки — предупреждение, не фатал.
//
// Я убрал пропуск по Detect() (шаг 7): раньше `caf init --adapters opencode`
// ничего не делал, если агент уже был установлен — а настройка хуков
// существующего агента и есть задача адаптера. Install() идемпотентен
// (ARCHITECT.md §3.1), поэтому вызывать его можно сколько угодно раз.
// Проверку Getenv("HOME") убрал тоже: на Windows она пуста и блокировала
// установку, а домашний каталог адаптеры резолвят сами через
// os.UserHomeDir() (USERPROFILE на Windows).
func installAdapter(id string) {
	adapter, ok := caf.GetAdapter(id)
	if !ok {
		fmt.Printf("WARN: unknown adapter %q; known: %s\n", id, knownAdapterIDs())
		return
	}
	if _, isLang := adapter.(caf.LanguageAdapter); isLang {
		fmt.Printf("WARN: %q is a language adapter, not an event adapter; skipped\n", id)
		return
	}
	if err := adapter.Install(); err != nil {
		fmt.Printf("WARN: install %s: %v\n", id, err)
		return
	}
	fmt.Printf("OK: adapter %s installed (hooks wired)\n", id)
	if !adapter.Detect() {
		fmt.Printf("NOTE: %s binary not found in $PATH — hooks activate once you install the agent\n", id)
	}
}

func knownAdapterIDs() string {
	var ids []string
	for _, a := range trackableAdapters() {
		ids = append(ids, a.ID())
	}
	return strings.Join(ids, ", ")
}

// runHookEvent: диспетчер событий. Вызывает *Hook напрямую (не *Command — те
// рекурсивно перезапускали caf через ExecuteHook; баг исправлен на шаге 6,
// см. PreCommitCommand/PreSessionCommand).
// fromAgent=true — pre-commit со stdin-вводом агента (шаг 7, см. hooks.PreCommitFromAgent).
// Возвращает exit-код: 0 всегда, кроме неизвестного события (usage error).
func runHookEvent(event, repoRoot, sessionID string, files []string, message string, fromAgent bool) int {
	ctx := hooks.HookContext{
		RepoRoot:  repoRoot,
		SessionID: sessionID,
		EventType: event,
	}

	var err error
	switch event {
	case "pre-session":
		err = hooks.PreSessionHook(ctx)
	case "post-edit":
		err = hooks.PostEditHook(ctx, files, message)
	case "pre-commit":
		if fromAgent {
			// stdin читается ТОЛЬКО здесь (по флагу): git-путь хука никогда
			// не трогает stdin и не может заблокироваться (INV-ARCH-004).
			err = hooks.PreCommitFromAgent(ctx, os.Stdin)
			break
		}
		if message == "" {
			message = readCommitEditMsg(repoRoot)
		}
		err = hooks.PreCommitHook(ctx, message)
	default:
		fmt.Fprintf(os.Stderr, "unknown hook event %q; expected pre-session, post-edit or pre-commit\n", event)
		return 1
	}

	if err != nil {
		// Fail-safe (INV-ARCH-004): warn-and-continue, никогда не блокируем коммит/сессию.
		fmt.Fprintf(os.Stderr, "WARN: hook %s failed (non-blocking, INV-ARCH-004): %v\n", event, err)
		return 0
	}
	fmt.Printf("OK: hook %s executed\n", event)
	return 0
}

// readCommitEditMsg читает .git/COMMIT_EDITMSG как fallback сообщения для pre-commit
// (git pre-commit не передаёт сообщение хуку аргументом).
func readCommitEditMsg(repoRoot string) string {
	data, err := os.ReadFile(filepath.Join(repoRoot, ".git", "COMMIT_EDITMSG"))
	if err != nil {
		return ""
	}
	return string(data)
}

// runWho: реальный поиск по .caf/sessions (FR-PROV-003, file-level MVP).
func runWho(location, repoRoot string) error {
	file, line := splitFileLine(location)

	if _, err := os.Stat(storage.GetCAFDir(repoRoot)); err != nil {
		return fmt.Errorf("not initialized: %s missing — run `caf init` first", storage.GetCAFDir(repoRoot))
	}

	manager := storage.NewManager(filepath.Join(repoRoot, ".caf", "sessions"))
	ids, err := manager.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}
	if len(ids) == 0 {
		fmt.Println("No sessions recorded yet. Start an AI agent session in this repo to begin tracking.")
		return nil
	}

	matches := 0
	for _, id := range ids {
		session, err := manager.LoadSession(id)
		if err != nil {
			continue
		}
		touched, iters := matchSessionFile(session, file)
		if !touched {
			continue
		}
		matches++
		fmt.Printf("Session %s\n", session.SessionID)
		fmt.Printf("  Agent:        %s\n", session.Agent)
		fmt.Printf("  Model:        %s\n", session.Model)
		fmt.Printf("  Started:      %s\n", session.StartedAt.Format("2006-01-02T15:04:05Z07:00"))
		fmt.Printf("  InitialPrompt: %s\n", firstLine(session.InitialPrompt))
		for _, it := range iters {
			fmt.Printf("  Iteration %s: %s %s", it.ID, it.Action, it.Target)
			if it.Diff != "" {
				fmt.Printf(" — %s", it.Diff)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	if matches == 0 {
		fmt.Printf("No session touched %s.\n", file)
		return nil
	}
	if line != "" {
		fmt.Printf("Note: line-level attribution (line %s) ships in ALPHA (FR-PROV-003); MVP matches by file.\n", line)
	}
	return nil
}

// splitFileLine разбирает "path/to/file.go:42" → ("path/to/file.go", "42").
// Без двоеточия или без номера — line пустая.
func splitFileLine(location string) (file, line string) {
	idx := strings.LastIndex(location, ":")
	if idx <= 0 {
		return location, ""
	}
	return location[:idx], location[idx+1:]
}

// matchSessionFile: файл из FilesTouched или Target итерации (нормализация путей).
func matchSessionFile(s *storage.Session, file string) (bool, []storage.Iteration) {
	norm := filepath.ToSlash(file)
	var iters []storage.Iteration
	for _, ft := range s.FilesTouched {
		if pathMatches(filepath.ToSlash(ft), norm) {
			for _, it := range s.Iterations {
				if strings.Contains(filepath.ToSlash(it.Target), norm) {
					iters = append(iters, it)
				}
			}
			return true, iters
		}
	}
	// Fallback: только по итерациям (файл мог не попасть в files_touched).
	for _, it := range s.Iterations {
		if strings.Contains(filepath.ToSlash(it.Target), norm) {
			iters = append(iters, it)
		}
	}
	return len(iters) > 0, iters
}

// pathMatches: полное равенство или суффикс с границей сегмента.
func pathMatches(recorded, queried string) bool {
	return recorded == queried ||
		strings.HasSuffix(recorded, "/"+queried) ||
		strings.HasSuffix(queried, "/"+recorded)
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}

// runWhen: ALPHA scope — честное сообщение, без выдуманных данных (§2.1, §3).
func runWhen(pattern string) error {
	fmt.Printf("Pattern %q acknowledged.\n", pattern)
	fmt.Println("`caf when` is ALPHA scope (ROADMAP.md, BACKLOG.md) — deferred from MVP.")
	fmt.Println("No history search is performed in MVP; nothing to report.")
	return nil
}

// runContext: --refresh генерирует context.md; без флага — статус файла (§2.1).
func runContext(repoRoot string, refresh bool) error {
	contextFile := filepath.Join(repoRoot, ".caf", "context.md")

	if !refresh {
		info, err := os.Stat(contextFile)
		if err != nil {
			fmt.Println("No .caf/context.md yet. Generate it with: caf context --refresh")
			return nil
		}
		fmt.Printf("context.md exists (%d bytes, modified %s).\n", info.Size(), info.ModTime().Format("2006-01-02T15:04:05"))
		fmt.Println("Regenerate with: caf context --refresh")
		return nil
	}

	gen := storage.NewContextGenerator(
		filepath.Join(repoRoot, ".caf", "sessions"),
		filepath.Join(repoRoot, ".caf", "decisions"),
		20, // max sessions
		10, // max decisions
	)
	if err := gen.GenerateContext(repoRoot); err != nil {
		return fmt.Errorf("failed to generate context: %w", err)
	}
	fmt.Printf("OK: generated %s\n", contextFile)
	return nil
}

// runDashboard: TUI-хаб (bubbletea). Ошибка запуска TUI — фатальна (это не hook).
func runDashboard(repoRoot string) error {
	// Use direct dashboard model to avoid import cycle issue
	return runDashboardDirect(repoRoot)
}

// runAdaptersList: capability matrix + статус детекции (текстовая форма UI.md §7).
func runAdaptersList() error {
	list := trackableAdapters()
	if len(list) == 0 {
		fmt.Println("No adapters registered.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTIER\tpre-session\tpost-edit\tpre-commit\tMCP\tSTATUS")
	for _, a := range list {
		m := a.Capabilities()
		status := adapterStatus(a)
		mcp := "no"
		if m.MCP {
			mcp = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			a.ID(), formatTier(m.Tier),
			formatCell(m.PreSession), formatCell(m.PostEdit), formatCell(m.PreCommit),
			mcp, status)
	}
	return w.Flush()
}

// adapterStatus — статус адаптера в `caf adapters list` (UI.md §7).
//
// Два независимых вопроса (шаг 7): «есть ли агент на машине» — Detect
// (ARCHITECT.md §3.1) и «прописан ли caf в конфиге агента» — опциональный
// caf.Configured. Адаптер без проводки (Tier-4 universal-lsp, шаг 8)
// показывается как раньше: installed / not detected.
func adapterStatus(a caf.Adapter) string {
	wiring, hasWiring := a.(caf.Configured)
	detected := a.Detect()

	if !hasWiring {
		if detected {
			return "installed"
		}
		return "not detected"
	}

	wired := wiring.Configured()
	switch {
	case wired && detected:
		return "installed (hooks wired)"
	case wired:
		return "hooks wired (agent not in $PATH)"
	case detected:
		return "installed (hooks not wired — run caf init --adapters " + a.ID() + ")"
	default:
		return "not detected"
	}
}

// runDoctor: 5 базовых проверок (FR-OBS-004 partial).
func runDoctor(repoRoot string) error {
	report, err := doctor.RunCommand(repoRoot)
	if err != nil {
		return fmt.Errorf("health checks failed to run: %w", err)
	}
	fmt.Print(doctor.FormatReport(report))
	return nil
}

// runAutoDetect: machine-wide детекция; возвращает список установленных ID.
func runAutoDetect() []string {
	fmt.Println("Detecting AI agents on this machine...")
	detected := reportDetection()
	if rec := recommendedAdapter(detected); rec != "" {
		fmt.Printf("\nRecommended adapter: %s\n", rec)
	} else {
		fmt.Println("\nNo Tier-1 agents detected; universal LSP fallback remains available.")
	}
	fmt.Println("\nInstall agent hooks with: caf init --adapters <id>")
	return detected
}

// reportDetection печатает таблицу детекции и возвращает ID установленных адаптеров.
func reportDetection() []string {
	var detected []string
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTIER\tSTATUS")
	for _, a := range trackableAdapters() {
		status := "not detected"
		if a.Detect() {
			status = "installed"
			detected = append(detected, a.ID())
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", a.ID(), formatTier(a.Capabilities().Tier), status)
	}
	_ = w.Flush()
	return detected
}

// recommendedAdapter: приоритет opencode → claude-code → universal-lsp (§2.5).
func recommendedAdapter(detected []string) string {
	for _, want := range []string{"opencode", "claude-code", "universal-lsp"} {
		for _, id := range detected {
			if id == want {
				return id
			}
		}
	}
	return ""
}

// runDashboardTUI: Launch TUI dashboard using direct model
func runDashboardTUI(repoRoot string) error {
	// Create dashboard model
	model := NewDashboardModel(repoRoot, version)
	
	if _, err := runTUI(model, tea.WithAltScreen()); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	
	return nil
}
