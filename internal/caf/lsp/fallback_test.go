package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/storage"
)

// TestMain — re-exec паттерн: дочерний процесс test-бинарника с
// CAF_FAKE_GOPLS=1 играет роль LSP-сервера (тот же stdio-фрейминг, что и
// в универсальном адаптере; дубль минимален, потому что пакеты тестируются
// изолированно).
func TestMain(m *testing.M) {
	if os.Getenv("CAF_FAKE_GOPLS") == "1" {
		runFakeGopls()
		return
	}
	os.Exit(m.Run())
}

func runFakeGopls() {
	in := bufio.NewReader(os.Stdin)
	for {
		body, err := readLSPFrame(in)
		if err != nil {
			return
		}
		var req struct {
			ID     *int   `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(body, &req) != nil || req.ID == nil {
			continue
		}
		var result interface{} = map[string]interface{}{}
		if req.Method == "initialize" {
			result = map[string]interface{}{"capabilities": map[string]interface{}{}}
		}
		if req.Method == "workspace/symbol" {
			result = []map[string]interface{}{
				{
					"name": "FakeFn", "kind": 12,
					"location": map[string]interface{}{
						"uri": "file:///fake/x.go",
						"range": map[string]interface{}{
							"start": map[string]int{"line": 0, "character": 0},
							"end":   map[string]int{"line": 0, "character": 1},
						},
					},
				},
			}
		}
		resp, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": *req.ID, "result": result})
		fmt.Fprintf(os.Stdout, "Content-Length: %d\r\n\r\n%s", len(resp), resp)
	}
}

func readLSPFrame(r *bufio.Reader) ([]byte, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if name, value, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, err
			}
			length = n
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

// installFakeGopls кладёт обёртку test-бинарника под именем gopls в PATH.
// Принимает testing.TB — работает и в тестах, и в бенчмарках (шаг 15).
func installFakeGopls(tb testing.TB) {
	tb.Helper()
	dir := tb.TempDir()
	if runtime.GOOS == "windows" {
		script := "@echo off\r\nset CAF_FAKE_GOPLS=1\r\n\"" + os.Args[0] + "\"\r\n"
		if err := os.WriteFile(filepath.Join(dir, "gopls.bat"), []byte(script), 0644); err != nil {
			tb.Fatal(err)
		}
	} else {
		script := "#!/bin/sh\nexport CAF_FAKE_GOPLS=1\nexec \"" + os.Args[0] + "\"\n"
		if err := os.WriteFile(filepath.Join(dir, "gopls"), []byte(script), 0755); err != nil {
			tb.Fatal(err)
		}
	}
	tb.Setenv("PATH", dir)
}

// TestLazyLoadWithoutGopls — пустой $PATH: LazyLoad/Start возвращают ошибку
// с hint про установку, без паники и без запуска процесса (fail-safe дух).
func TestLazyLoadWithoutGopls(t *testing.T) {
	t.Setenv("PATH", "")
	m := NewFallbackManager(t.TempDir())

	err := m.LazyLoad()
	if err == nil {
		t.Fatal("LazyLoad without gopls must fail")
	}
	if !strings.Contains(err.Error(), "gopls") {
		t.Errorf("error must name gopls, got: %v", err)
	}
	if m.GetAdapter().Started() {
		t.Error("no process must be running after failed LazyLoad")
	}
	if m.IsAvailable() {
		t.Error("IsAvailable() = true without gopls")
	}
}

// TestSaveLoadSymbolGraphRoundTrip — Load читает кэш, Save пересохраняет
// в fresh-репозиторий (создаёт parent-директории сам).
func TestSaveLoadSymbolGraphRoundTrip(t *testing.T) {
	repo := t.TempDir()
	m := NewFallbackManager(repo)

	// Граф в памяти — через LoadSymbolGraph из предзаписанного кэша.
	want := []caf.Symbol{
		{Name: "Handler", Kind: "function", Location: "api.go:10:1"},
		{Name: "Server", Kind: "struct", Location: "api.go:20:1"},
	}
	graphFile := filepath.Join(repo, ".caf", "mental-model", "graph.yml")
	if err := storage.EnsureDir(filepath.Dir(graphFile)); err != nil {
		t.Fatal(err)
	}
	if err := storage.WriteYAML(graphFile, caf.SymbolGraph{Symbols: want}); err != nil {
		t.Fatal(err)
	}

	if err := m.LoadSymbolGraph(); err != nil {
		t.Fatalf("LoadSymbolGraph: %v", err)
	}
	got := m.GetSymbols()
	if len(got) != len(want) {
		t.Fatalf("loaded %d symbols, want %d", len(got), len(want))
	}
	if got[0].Name != want[0].Name || got[0].Location != want[0].Location {
		t.Errorf("symbol[0] = %+v, want %+v", got[0], want[0])
	}

	// Пересохранение в fresh-репозиторий без .caf/ — SaveSymbolGraph создаёт
	// parent-директории сам (баг прежней версии: WriteYAML падал без mkdir).
	repo2 := t.TempDir()
	m2 := NewFallbackManager(repo2)
	m2.symbols = want
	if err := m2.SaveSymbolGraph(); err != nil {
		t.Fatalf("SaveSymbolGraph into fresh repo: %v", err)
	}
	var roundtrip caf.SymbolGraph
	if err := storage.ReadYAML(filepath.Join(repo2, ".caf", "mental-model", "graph.yml"), &roundtrip); err != nil {
		t.Fatalf("re-read saved graph: %v", err)
	}
	if len(roundtrip.Symbols) != len(want) {
		t.Errorf("round-trip got %d symbols, want %d", len(roundtrip.Symbols), len(want))
	}
}

// TestGetManagerForRepoRequiresCAFDir — без .caf/ менеджер не создаётся.
func TestGetManagerForRepoRequiresCAFDir(t *testing.T) {
	if _, err := GetManagerForRepo(t.TempDir()); err == nil {
		t.Fatal("GetManagerForRepo without .caf/ must fail")
	}
}

// TestGetManagerForRepoUsesCache — при наличии кэша менеджер возвращается
// без старта LSP (lazy-load INV-ARCH-003): gopls нет в $PATH, но ошибка нет.
func TestGetManagerForRepoUsesCache(t *testing.T) {
	t.Setenv("PATH", "")
	repo := t.TempDir()
	if err := storage.InitializeCAF(repo); err != nil {
		t.Fatal(err)
	}
	graphFile := filepath.Join(repo, ".caf", "mental-model", "graph.yml")
	if err := storage.WriteYAML(graphFile, caf.SymbolGraph{
		Symbols: []caf.Symbol{{Name: "Cached", Kind: "function", Location: "x.go:1:1"}},
	}); err != nil {
		t.Fatal(err)
	}

	m, err := GetManagerForRepo(repo)
	if err != nil {
		t.Fatalf("GetManagerForRepo with cache: %v", err)
	}
	if m.StartupMeasured() {
		t.Error("manager must not start LSP when cache is fresh (lazy-load)")
	}
	symbols := m.GetSymbols()
	if len(symbols) != 1 || symbols[0].Name != "Cached" {
		t.Errorf("symbols = %+v, want cached one", symbols)
	}
}

// TestGetManagerForRepoDegradesWithoutGopls — кэша нет, gopls нет: менеджер
// возвращается (не падает), LSP не поднят, doctor покажет warning.
func TestGetManagerForRepoDegradesWithoutGopls(t *testing.T) {
	t.Setenv("PATH", "")
	repo := t.TempDir()
	if err := storage.InitializeCAF(repo); err != nil {
		t.Fatal(err)
	}

	m, err := GetManagerForRepo(repo)
	if err != nil {
		t.Fatalf("GetManagerForRepo must degrade, not fail: %v", err)
	}
	if m.GetAdapter().Started() {
		t.Error("no LSP process without gopls")
	}
	if len(m.GetSymbols()) != 0 {
		t.Error("no symbols without cache or LSP")
	}
}

// TestCrashDegradation — после maxCrashes фатальных сбоев Start отказывается
// перезапускать сервер (ADR-013 mitigation), IsAvailable=false, Degraded=true.
func TestCrashDegradation(t *testing.T) {
	t.Setenv("PATH", "")
	m := NewFallbackManager(t.TempDir())
	m.crashes = maxCrashes

	err := m.Start()
	if err == nil {
		t.Fatal("Start after maxCrashes must fail")
	}
	if !strings.Contains(err.Error(), "crashes") {
		t.Errorf("error must mention crashes, got: %v", err)
	}
	if m.IsAvailable() {
		t.Error("IsAvailable() = true after degradation")
	}
	if !m.Degraded() {
		t.Error("Degraded() = false after maxCrashes")
	}
	if m.Crashes() != maxCrashes {
		t.Errorf("Crashes() = %d, want %d", m.Crashes(), maxCrashes)
	}
}

// TestSyncSymbolsWithoutStart — SyncSymbols до Start: ошибка, не паника.
func TestSyncSymbolsWithoutStart(t *testing.T) {
	m := NewFallbackManager(t.TempDir())
	if err := m.SyncSymbols(); err == nil {
		t.Fatal("SyncSymbols before Start must fail")
	}
}

// TestStopIdempotent — двойной Stop без процесса: nil, nil.
func TestStopIdempotent(t *testing.T) {
	m := NewFallbackManager(t.TempDir())
	if err := m.Stop(); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := m.Stop(); err != nil {
		t.Fatalf("second Stop must be nil: %v", err)
	}
}

// TestGetSymbolGraphEmpty — до синка граф пустой, но валидной формы.
func TestGetSymbolGraphEmpty(t *testing.T) {
	m := NewFallbackManager(t.TempDir())
	g := m.GetSymbolGraph()
	if len(g.Symbols) != 0 {
		t.Errorf("fresh manager graph must be empty, got %d symbols", len(g.Symbols))
	}
}

// TestStartSyncStopWithFakeGopls — полный цикл против fake-gopls: Start
// поднимает процесс и синкает граф, SaveSymbolGraph пишет кэш, повторный
// Start — no-op, Stop идемпотентен.
func TestStartSyncStopWithFakeGopls(t *testing.T) {
	installFakeGopls(t)
	repo := t.TempDir()
	m := NewFallbackManager(repo)

	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !m.GetAdapter().Started() {
		t.Fatal("adapter not started after Start")
	}
	if !m.StartupMeasured() {
		t.Error("StartupMeasured() = false after Start")
	}

	// Start уже синкнул начальный граф (best-effort) — но независимо от этого
	// явный SyncSymbols обязан вернуть символы fake-сервера.
	if err := m.SyncSymbols(); err != nil {
		t.Fatalf("SyncSymbols: %v", err)
	}
	symbols := m.GetSymbols()
	if len(symbols) != 1 || symbols[0].Name != "FakeFn" {
		t.Errorf("symbols = %+v, want [FakeFn]", symbols)
	}

	// Кэш записан в .caf/mental-model/graph.yml.
	if _, err := os.Stat(filepath.Join(repo, ".caf", "mental-model", "graph.yml")); err != nil {
		t.Errorf("graph cache not written: %v", err)
	}

	// Повторный Start — no-op без ошибки и без замены процесса.
	if err := m.Start(); err != nil {
		t.Errorf("second Start must be no-op: %v", err)
	}

	if err := m.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if m.GetAdapter().Started() {
		t.Error("adapter still started after Stop")
	}
	if err := m.Stop(); err != nil {
		t.Errorf("second Stop must be nil: %v", err)
	}
	// После Stop SyncSymbols — ошибка (сервер мёртв).
	if err := m.SyncSymbols(); err == nil {
		t.Error("SyncSymbols after Stop must fail")
	}
}

// TestGetSymbolsStaleResync — устаревший кэш (> syncInterval) при живом
// сервере триггерит пересинк; при мёртвом — отдаёт stale, не паникует.
func TestGetSymbolsStaleResync(t *testing.T) {
	installFakeGopls(t)
	m := NewFallbackManager(t.TempDir())
	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop()

	// Делаем кэш устаревшим.
	m.lastSync = time.Now().Add(-time.Hour)
	got := m.GetSymbols()
	if len(got) != 1 {
		t.Errorf("stale GetSymbols must resync from live server, got %d symbols", len(got))
	}

	// Мёртвый сервер + устаревший кэш: stale лучше, чем пустота/паника.
	_ = m.Stop()
	m.lastSync = time.Now().Add(-time.Hour)
	got = m.GetSymbols()
	if len(got) != 1 {
		t.Errorf("stale GetSymbols with dead server must keep old data, got %d", len(got))
	}
}

// TestInitializeConvenience — Initialize() = New + Start (удобный путь).
func TestInitializeConvenience(t *testing.T) {
	installFakeGopls(t)
	m, err := Initialize(t.TempDir())
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	defer m.Stop()
	if !m.GetAdapter().Started() {
		t.Error("Initialize must start the LSP server")
	}
}

// TestGetManagerForRepoStartsWithoutCache — .caf/ есть, кэша нет: менеджер
// запускает LSP (fake) и синкает.
func TestGetManagerForRepoStartsWithoutCache(t *testing.T) {
	installFakeGopls(t)
	repo := t.TempDir()
	if err := storage.InitializeCAF(repo); err != nil {
		t.Fatal(err)
	}

	m, err := GetManagerForRepo(repo)
	if err != nil {
		t.Fatalf("GetManagerForRepo: %v", err)
	}
	defer m.Stop()
	if !m.StartupMeasured() {
		t.Error("manager must start LSP when no cache exists")
	}
	if len(m.GetSymbols()) != 1 {
		t.Errorf("expected fake symbol after start, got %d", len(m.GetSymbols()))
	}
}
