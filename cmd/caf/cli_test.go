// cli_test.go — тесты сквозного исполнения cobra-команд через
// setupCommands + rootCmd.SetArgs/Execute (без os.Exit) и перехвата
// интерактивного запуска TUI через seam runTUI (§10 покрытие cmd/caf).
package main

import (
	"errors"
	"io"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/adapters"
)

// installFakeTUI подменяет seam запуска bubbletea: команда строит модель,
// но вечный event loop не стартует.
func installFakeTUI(tb testing.TB) {
	tb.Helper()
	old := runTUI
	runTUI = func(model tea.Model, opts ...tea.ProgramOption) (tea.Model, error) {
		return model, nil
	}
	tb.Cleanup(func() { runTUI = old })
}

// TestRootCommandExecution — каждая подкоманда исполняется через cobra как
// в проде, только вместо TUI-запуска стоит фейк. hookCmd НЕ исполняется:
// его Run вызывает os.Exit (покрыт напрямую в TestRunHookEventExitCodes).
func TestRootCommandExecution(t *testing.T) {
	setupCommands()
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	installFakeTUI(t)
	tempHome(t)
	adapters.Initialize()

	repo := t.TempDir()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"init creates structure", []string{"init", "--repo", repo}, false},
		{"init --tui uses onboarding launcher", []string{"init", "--repo", t.TempDir(), "--tui"}, false},
		{"who on uninitialized repo errors", []string{"who", "--repo", t.TempDir(), "main.go"}, true},
		{"who after init", []string{"who", "--repo", repo, "main.go"}, false},
		{"when acknowledges ALPHA deferral", []string{"when", "--pattern", "func main"}, false},
		{"context status", []string{"context", repo}, false},
		{"context --refresh generates file", []string{"context", "--refresh", repo}, false},
		{"adapters list", []string{"adapters", "list"}, false},
		{"adapters list --tui", []string{"adapters", "list", "--tui"}, false},
		{"doctor", []string{"doctor", repo}, false},
		{"auto-detect", []string{"auto-detect"}, false},
		{"dashboard", []string{"dashboard"}, false},
		{"dashboard --tui", []string{"dashboard", "--tui"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rootCmd.SetArgs(tc.args)
			err := rootCmd.Execute()
			if tc.wantErr && err == nil {
				t.Fatal("want error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Execute(%v): %v", tc.args, err)
			}
		})
	}
}

// TestSetupCommandsIdempotent — повторный вызов не дублирует подкоманды.
func TestSetupCommandsIdempotent(t *testing.T) {
	setupCommands()
	before := len(rootCmd.Commands())
	setupCommands()
	if after := len(rootCmd.Commands()); after != before {
		t.Errorf("commands before=%d after=%d, must be idempotent", before, after)
	}
}

// TestTUIRunnersPropagateErrors — seam возвращает ошибку → раннеры возвращают
// обёрнутые ошибки (никогда не паникуют).
func TestTUIRunnersPropagateErrors(t *testing.T) {
	old := runTUI
	runTUI = func(model tea.Model, opts ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("tty exploded")
	}
	defer func() { runTUI = old }()

	repo := t.TempDir()
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{"dashboard", func() error { return runDashboardDirect(repo) }, "dashboard failed"},
		{"adapters", func() error { return runAdaptersTUI(repo) }, "adapters TUI failed"},
		{"onboarding", func() error { return runInitTUI(repo) }, "onboarding TUI failed"},
		{"dashboard via command", func() error { return runDashboardTUI(repo) }, "failed to run TUI"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			if err == nil {
				t.Fatal("want wrapped error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "tty exploded") {
				t.Errorf("error = %q, want %q wrapping %q", err, tc.want, "tty exploded")
			}
		})
	}
}

// --- ветки installAdapter (WARN/OK/NOTE) ---

// langAdapterStub — тестовый LanguageAdapter в реестре (прод-реестр их не
// содержит с шага 7, но ветка «WARN: language adapter» обязана быть покрыта).
type langAdapterStub struct{ id string }

func (l langAdapterStub) ID() string   { return l.id }
func (l langAdapterStub) Detect() bool { return false }
func (l langAdapterStub) Install() error   { return nil }
func (l langAdapterStub) Uninstall() error { return nil }
func (l langAdapterStub) Capabilities() caf.CapabilityMatrix {
	return caf.CapabilityMatrix{PreSession: caf.TierNone, PostEdit: caf.TierNone, PreCommit: caf.TierNone, Tier: 4}
}
func (l langAdapterStub) ScanImports() []caf.Import             { return nil }
func (l langAdapterStub) BuildSymbolGraph() caf.SymbolGraph     { return caf.SymbolGraph{} }
func (l langAdapterStub) ResolveExternalRepo(p string) caf.RepoRef { return caf.RepoRef{} }

type failingInstallAdapter struct{ id string }

func (f failingInstallAdapter) ID() string     { return f.id }
func (f failingInstallAdapter) Detect() bool   { return false }
func (f failingInstallAdapter) Install() error { return errors.New("boom: cannot write config") }
func (f failingInstallAdapter) Uninstall() error { return nil }
func (f failingInstallAdapter) Capabilities() caf.CapabilityMatrix {
	return caf.CapabilityMatrix{PreSession: caf.TierNative, PostEdit: caf.TierNative, PreCommit: caf.TierNative, MCP: true, Tier: 1}
}

type undetectedAdapter struct{ id string }

func (u undetectedAdapter) ID() string       { return u.id }
func (u undetectedAdapter) Detect() bool     { return false } // бинарь не в $PATH → NOTE
func (u undetectedAdapter) Install() error   { return nil }
func (u undetectedAdapter) Uninstall() error { return nil }
func (u undetectedAdapter) Capabilities() caf.CapabilityMatrix {
	return caf.CapabilityMatrix{PreSession: caf.TierNative, PostEdit: caf.TierNative, PreCommit: caf.TierNative, MCP: true, Tier: 1}
}

// TestInstallAdapterBranches — все ветки installAdapter: language-warn,
// install-error-warn, OK+NOTE (бинарь не найден), unknown-id warn.
func TestInstallAdapterBranches(t *testing.T) {
	tempHome(t)
	adapters.Initialize()

	caf.RegisterAdapter(langAdapterStub{id: "stub-lang"})
	caf.RegisterAdapter(failingInstallAdapter{id: "stub-fail"})
	caf.RegisterAdapter(undetectedAdapter{id: "stub-nodetect"})
	t.Cleanup(func() {
		delete(caf.Registry, "stub-lang")
		delete(caf.Registry, "stub-fail")
		delete(caf.Registry, "stub-nodetect")
	})

	installAdapter("stub-lang")     // WARN: language adapter skipped
	installAdapter("stub-fail")     // WARN: install error (не фатал)
	installAdapter("stub-nodetect") // OK + NOTE: бинарь не в $PATH
	installAdapter("not-registered") // WARN: unknown id
}

// TestExecuteUnknownCommandErrors — неизвестная подкоманда → usage-ошибка.
func TestExecuteUnknownCommandErrors(t *testing.T) {
	setupCommands()
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)

	rootCmd.SetArgs([]string{"definitely-not-a-command"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("unknown command must return an error")
	}
}
