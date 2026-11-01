// main_test.go — тесты CLI-хелперов caf: флаги, разбор путей, матчинг сессий,
// статусы адаптеров, runWho/runContext (FR-PROV-003, FR-ADP-005, §2.1).
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/adapters"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/storage"
)

func TestFormatTier(t *testing.T) {
	tests := []struct {
		tier int
		want string
	}{
		{1, "T1 (native)"},
		{4, "T4 (fallback)"},
		{2, "T2"},
		{3, "T3"},
		{0, "T0"},
	}
	for _, tc := range tests {
		if got := formatTier(tc.tier); got != tc.want {
			t.Errorf("formatTier(%d) = %q, want %q", tc.tier, got, tc.want)
		}
	}
}

func TestFormatCell(t *testing.T) {
	// INV-UI-008: ячейка — всегда текст, никогда только цвет.
	if got := formatCell(caf.TierNative); got != "native" {
		t.Errorf("formatCell(native) = %q", got)
	}
	if got := formatCell(caf.TierWatcher); got != "watcher" {
		t.Errorf("formatCell(watcher) = %q", got)
	}
	if got := formatCell(caf.TierNone); got != "none" {
		t.Errorf("formatCell(none) = %q", got)
	}
}

func TestSplitFileLine(t *testing.T) {
	tests := []struct {
		in       string
		wantFile string
		wantLine string
	}{
		{"internal/api/policies.go:42", "internal/api/policies.go", "42"},
		{"README.md", "README.md", ""},
		{":42", ":42", ""},
		{"", "", ""},
	}
	for _, tc := range tests {
		file, line := splitFileLine(tc.in)
		if file != tc.wantFile || line != tc.wantLine {
			t.Errorf("splitFileLine(%q) = (%q, %q), want (%q, %q)", tc.in, file, line, tc.wantFile, tc.wantLine)
		}
	}
}

func TestPathMatches(t *testing.T) {
	tests := []struct {
		recorded string
		queried  string
		want     bool
	}{
		{"internal/api/policies.go", "internal/api/policies.go", true},
		{"internal/api/policies.go", "policies.go", true},        // суффикс
		{"policies.go", "internal/api/policies.go", true},        // обратный суффикс
		{"internal/api/policies.go", "handlers.go", false},       // другой файл
		{"internal/api/policies.go", "api/policies.go.bak", false}, // граница сегмента
	}
	for _, tc := range tests {
		if got := pathMatches(tc.recorded, tc.queried); got != tc.want {
			t.Errorf("pathMatches(%q, %q) = %v, want %v", tc.recorded, tc.queried, got, tc.want)
		}
	}
}

func TestMatchSessionFile(t *testing.T) {
	session := &storage.Session{
		SessionID:   "2026-10-24-a1b2c3",
		FilesTouched: []string{"internal/api/policies.go"},
		Iterations: []storage.Iteration{
			{ID: "i1", Action: "read", Target: "internal/api/policies.go"},
			{ID: "i2", Action: "edit", Target: "internal/api/policies.go", Diff: "added middleware"},
			{ID: "i3", Action: "read", Target: "README.md"},
		},
	}

	t.Run("matches by file suffix and returns iterations", func(t *testing.T) {
		ok, iters := matchSessionFile(session, "policies.go")
		if !ok {
			t.Fatal("policies.go must match session via files_touched")
		}
		if len(iters) != 2 {
			t.Errorf("iterations = %d, want 2 (i1, i2)", len(iters))
		}
	})

	t.Run("falls back to iteration targets", func(t *testing.T) {
		s := &storage.Session{
			SessionID:   "2026-10-24-x", // файл не в files_touched
			Iterations:  []storage.Iteration{{ID: "i9", Action: "edit", Target: "cmd/caf/main.go"}},
		}
		ok, iters := matchSessionFile(s, "main.go")
		if !ok || len(iters) != 1 {
			t.Errorf("fallback match = (%v, %d iters), want (true, 1)", ok, len(iters))
		}
	})

	t.Run("no match returns false", func(t *testing.T) {
		ok, iters := matchSessionFile(session, "nonexistent.go")
		if ok || len(iters) != 0 {
			t.Errorf("no-match = (%v, %d), want (false, 0)", ok, len(iters))
		}
	})
}

func TestFirstLine(t *testing.T) {
	if got := firstLine("first\nsecond"); got != "first" {
		t.Errorf("firstLine(\\n) = %q", got)
	}
	if got := firstLine("first\rsecond"); got != "first" {
		t.Errorf("firstLine(\\r) = %q", got)
	}
	if got := firstLine("single"); got != "single" {
		t.Errorf("firstLine(no newline) = %q", got)
	}
	if got := firstLine(""); got != "" {
		t.Errorf("firstLine(empty) = %q", got)
	}
}

func TestRecommendedAdapter(t *testing.T) {
	tests := []struct {
		name     string
		detected []string
		want     string
	}{
		{"priority opencode over claude-code", []string{"claude-code", "opencode"}, "opencode"},
		{"claude-code over universal", []string{"universal-lsp", "claude-code"}, "claude-code"},
		{"only universal fallback", []string{"universal-lsp"}, "universal-lsp"},
		{"nothing detected", nil, ""},
		{"unknown ids only", []string{"cursor", "cline"}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := recommendedAdapter(tc.detected); got != tc.want {
				t.Errorf("recommendedAdapter(%v) = %q, want %q", tc.detected, got, tc.want)
			}
		})
	}
}

// --- статусы адаптеров (UI.md §7) ---

type fakeAdapter struct {
	id       string
	detected bool
}

func (f fakeAdapter) ID() string       { return f.id }
func (f fakeAdapter) Detect() bool     { return f.detected }
func (f fakeAdapter) Install() error   { return nil }
func (f fakeAdapter) Uninstall() error { return nil }
func (f fakeAdapter) Capabilities() caf.CapabilityMatrix {
	return caf.CapabilityMatrix{
		PreSession: caf.TierNative,
		PostEdit:   caf.TierNative,
		PreCommit:  caf.TierNative,
		MCP:        true,
		Tier:       1,
	}
}

// wiredAdapter реализует опциональный caf.Configured (ADR-009: ядро — ровно
// 5 методов; проводка хуков — отдельное расширение).
type wiredAdapter struct {
	fakeAdapter
	wired bool
}

func (w wiredAdapter) Configured() bool { return w.wired }

func TestAdapterStatus(t *testing.T) {
	tests := []struct {
		name   string
		adapter caf.Adapter
		want   string
	}{
		{"plain detected", fakeAdapter{id: "x", detected: true}, "installed"},
		{"plain not detected", fakeAdapter{id: "x", detected: false}, "not detected"},
		{"wired and detected", wiredAdapter{fakeAdapter{id: "x", detected: true}, true}, "installed (hooks wired)"},
		{"wired but agent missing", wiredAdapter{fakeAdapter{id: "x", detected: false}, true}, "hooks wired (agent not in $PATH)"},
		{"detected but not wired", wiredAdapter{fakeAdapter{id: "x", detected: true}, false}, "installed (hooks not wired — run caf init --adapters x)"},
		{"neither", wiredAdapter{fakeAdapter{id: "x", detected: false}, false}, "not detected"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := adapterStatus(tc.adapter); got != tc.want {
				t.Errorf("adapterStatus = %q, want %q", got, tc.want)
			}
		})
	}
}

// --- флаги и пути ---

func TestFlagHelpers(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("repo", "", "")
	cmd.Flags().Bool("auto-detect", false, "")
	cmd.Flags().StringSlice("files", nil, "")

	if got := flagString(cmd, "repo"); got != "" {
		t.Errorf("flagString default = %q, want empty", got)
	}
	_ = cmd.Flags().Set("repo", "/tmp/x")
	if got := flagString(cmd, "repo"); got != "/tmp/x" {
		t.Errorf("flagString = %q, want /tmp/x", got)
	}
	if got := flagString(cmd, "missing"); got != "" {
		t.Errorf("missing flag must return zero value, got %q", got)
	}

	if flagBool(cmd, "auto-detect") {
		t.Error("bool default must be false")
	}
	_ = cmd.Flags().Set("auto-detect", "true")
	if !flagBool(cmd, "auto-detect") {
		t.Error("flagBool must read set value")
	}
	if flagBool(cmd, "missing") {
		t.Error("missing bool flag must return false")
	}

	_ = cmd.Flags().Set("files", "a.go,b.go")
	got := flagStrings(cmd, "files")
	if len(got) != 2 || got[0] != "a.go" || got[1] != "b.go" {
		t.Errorf("flagStrings = %v, want [a.go b.go]", got)
	}
	if missing := flagStrings(cmd, "missing"); len(missing) != 0 {
		t.Errorf("missing slice flag must return empty value, got %v", missing)
	}
}

func TestResolveRepo(t *testing.T) {
	abs, err := resolveRepo("")
	if err != nil {
		t.Fatalf("resolveRepo(\"\"): %v", err)
	}
	if !filepath.IsAbs(abs) {
		t.Errorf("empty repo must resolve to cwd absolute path, got %q", abs)
	}

	rel, err := resolveRepo(filepath.Join(t.TempDir(), "sub"))
	if err != nil {
		t.Fatalf("resolveRepo(sub): %v", err)
	}
	if !filepath.IsAbs(rel) {
		t.Errorf("resolveRepo must return absolute path, got %q", rel)
	}
}

// --- команды (FR-PROV-003, §2.1) ---

func TestRunWhoNotInitialized(t *testing.T) {
	err := runWho("internal/api/policies.go:42", t.TempDir())
	if err == nil {
		t.Fatal("runWho on uninitialized repo must return an error")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("error must hint at caf init, got: %v", err)
	}
}

func TestRunWhoWithSessions(t *testing.T) {
	repoRoot := t.TempDir()
	if err := storage.InitializeCAF(repoRoot); err != nil {
		t.Fatalf("InitializeCAF: %v", err)
	}

	manager := storage.NewManager(filepath.Join(repoRoot, ".caf", "sessions"))
	err := manager.SaveSession(&storage.Session{
		SessionID:    "2026-10-24-a1b2c3",
		Repo:         "test-repo",
		StartedAt:    time.Date(2026, 10, 24, 11, 2, 14, 0, time.UTC),
		EndedAt:      time.Date(2026, 10, 24, 11, 47, 33, 0, time.UTC),
		Agent:        "opencode",
		InitialPrompt: "add rate limiter",
		FilesTouched: []string{"internal/api/policies.go"},
		Iterations: []storage.Iteration{
			{ID: "i1", Action: "read", Target: "internal/api/policies.go"},
			{ID: "i2", Action: "edit", Target: "internal/api/policies.go", Diff: "added middleware RateLimit"},
		},
	})
	if err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	// Найдено или нет — runWho не падает: печатает результат и возвращает nil.
	if err := runWho("internal/api/policies.go:42", repoRoot); err != nil {
		t.Errorf("runWho: %v", err)
	}
	if err := runWho("unrelated/file.go:1", repoRoot); err != nil {
		t.Errorf("runWho (no match): %v", err)
	}
}

func TestRunWhoWithNoSessionsYet(t *testing.T) {
	repoRoot := t.TempDir()
	if err := storage.InitializeCAF(repoRoot); err != nil {
		t.Fatalf("InitializeCAF: %v", err)
	}
	// Инициализирован, но сессий нет — честное сообщение, nil.
	if err := runWho("main.go", repoRoot); err != nil {
		t.Errorf("runWho on empty sessions: %v", err)
	}
}

func TestRunContextRefreshCreatesFile(t *testing.T) {
	repoRoot := t.TempDir()
	if err := storage.InitializeCAF(repoRoot); err != nil {
		t.Fatalf("InitializeCAF: %v", err)
	}

	manager := storage.NewManager(filepath.Join(repoRoot, ".caf", "sessions"))
	if err := manager.SaveSession(&storage.Session{
		SessionID:   "2026-10-24-a1b2c3",
		Repo:        "test-repo",
		StartedAt:   time.Date(2026, 10, 24, 11, 2, 14, 0, time.UTC),
		Agent:       "opencode",
		InitialPrompt: "add rate limiter",
	}); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	if err := runContext(repoRoot, false); err != nil {
		t.Errorf("runContext(status) must not fail: %v", err)
	}
	if err := runContext(repoRoot, true); err != nil {
		t.Fatalf("runContext(--refresh): %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".caf", "context.md")); err != nil {
		t.Errorf("context.md must exist after --refresh: %v", err)
	}
}

func TestRunWhenDefersToAlpha(t *testing.T) {
	// `caf when` — ALPHA scope: честное сообщение, ноль выдуманных данных (§3).
	if err := runWhen("internal/**/*.go"); err != nil {
		t.Errorf("runWhen must not fail in MVP, got: %v", err)
	}
}

func TestTrackableAdaptersExcludeLanguageAdapters(t *testing.T) {
	adapters.Initialize() // регистрирует встроенные адаптеры (Tier 1 + Tier 4)

	list := trackableAdapters()
	if len(list) == 0 {
		t.Fatal("expected builtin adapters after Initialize")
	}
	seen := map[string]bool{}
	for _, a := range list {
		seen[a.ID()] = true
		if _, isLang := a.(caf.LanguageAdapter); isLang {
			t.Errorf("%q must not be trackable (language adapter)", a.ID())
		}
	}
	for _, want := range []string{"opencode", "claude-code", "universal-lsp"} {
		if !seen[want] {
			t.Errorf("trackableAdapters missing %q", want)
		}
	}
}
