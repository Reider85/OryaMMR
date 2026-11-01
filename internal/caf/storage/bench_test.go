// bench_test.go — бенчмарки file ops на 10k сессий (PROMPT-MVP §6: file ops
// на 10k репо <1s; enforcement >2x → fail, §6:255).
//
// Фикстура (10 000 session YAML) создаётся один раз на процесс (sync.Once,
// параллельный worker-pool) и переиспользуется бенчмарками и budget-тестом;
// TestMain убирает каталог после прогонов.
//
// Сценарии:
//   - list:          ListSessions по каталогу с 10k файлов;
//   - list+load-all: ListSessions + LoadSession для каждой (путь dashboard
//     LoadData, pkg/ui/screens/dashboard.go:146) — строка бюджета §6;
//   - generate:      ContextGenerator.GenerateContext (informational;
//     enforcement отдельно, т.к. читает все сессии).
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

const fixtureSessions = 10_000

var (
	fixtureOnce sync.Once
	fixtureRoot string
	fixtureErr  error
)

// sessionsFixture10k возвращает каталог с 10k session YAML (создаёт один раз
// на процесс).
func sessionsFixture10k(tb testing.TB) string {
	tb.Helper()
	fixtureOnce.Do(func() {
		fixtureRoot, fixtureErr = os.MkdirTemp("", "caf-bench-10k-*")
		if fixtureErr != nil {
			return
		}
		sessionsDir := filepath.Join(fixtureRoot, "sessions")
		if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
			fixtureErr = err
			return
		}
		fixtureErr = writeSessionsParallel(sessionsDir, fixtureSessions)
	})
	if fixtureErr != nil {
		tb.Fatalf("create 10k fixture: %v", fixtureErr)
	}
	return filepath.Join(fixtureRoot, "sessions")
}

// writeSessionsParallel пишет n minimal session YAML worker-пулом (вне таймера
// бенчмарка). Контент валиден для ReadYAML и LoadSession.
func writeSessionsParallel(dir string, n int) error {
	sample := Session{
		Repo:          "bench-repo",
		Agent:         "opencode",
		Model:         "glm-4.6",
		ToolCall:      "opencode run \"bench\"",
		InitialPrompt: "bench fixture session",
		FilesTouched:  []string{"internal/api/policies.go"},
	}
	data, err := yaml.Marshal(&sample)
	if err != nil {
		return fmt.Errorf("marshal sample: %w", err)
	}

	const workers = 64
	jobs := make(chan int)
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		first error
	)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				name := filepath.Join(dir, fmt.Sprintf("2026-10-24-%06d.yml", i))
				if err := os.WriteFile(name, data, 0o644); err != nil {
					mu.Lock()
					if first == nil {
						first = err
					}
					mu.Unlock()
					return
				}
			}
		}()
	}
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return first
}

// TestMain — cleanup разделяемой 10k-фикстуры после всех прогонов.
func TestMain(m *testing.M) {
	code := m.Run()
	if fixtureRoot != "" {
		_ = os.RemoveAll(fixtureRoot)
	}
	os.Exit(code)
}

// BenchmarkFileOps10k/list — только ListSessions (§6 file ops).
func BenchmarkFileOps10k(b *testing.B) {
	dir := sessionsFixture10k(b)
	m := NewManager(dir)

	// Warm-up: прогрев FS-кэша после записи фикстуры (холодный прогон меряет
	// write-back ОС, а не наш код).
	if _, err := m.LoadAllSessions(); err != nil {
		b.Fatalf("warm-up: %v", err)
	}

	b.Run("list", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ids, err := m.ListSessions()
			if err != nil {
				b.Fatalf("ListSessions: %v", err)
			}
			if len(ids) != fixtureSessions {
				b.Fatalf("len(ids) = %d, want %d", len(ids), fixtureSessions)
			}
		}
	})

	b.Run("list-load-all", func(b *testing.B) {
		// Полный путь dashboard LoadData: параллельный bulk-load (§6 file ops).
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sessions, err := m.LoadAllSessions()
			if err != nil {
				b.Fatalf("LoadAllSessions: %v", err)
			}
			if len(sessions) != fixtureSessions {
				b.Fatalf("len(sessions) = %d, want %d", len(sessions), fixtureSessions)
			}
		}
	})
}

// BenchmarkGenerateContext — генерация context.md из 10k сессий (informational;
// enforcement — в TestBudgetGenerateContext, если >2x файла §6 budget).
func BenchmarkGenerateContext(b *testing.B) {
	dir := sessionsFixture10k(b)
	repoRoot := b.TempDir()
	if err := EnsureDir(filepath.Join(repoRoot, ".caf")); err != nil {
		b.Fatal(err)
	}
	g := NewContextGenerator(dir, filepath.Join(repoRoot, "no-decisions"), 50, 10)
	// Warm-up прогревает FS-кэш (см. BenchmarkFileOps10k).
	if err := g.GenerateContext(repoRoot); err != nil {
		b.Fatalf("warm-up: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := g.GenerateContext(repoRoot); err != nil {
			b.Fatalf("GenerateContext: %v", err)
		}
	}
}

// TestBudgetFileOps10k — enforcement §6: bulk-load 10k сессий ≤ 2s
// (2x бюджета file_op 1s; §6:255 >2x → CI fails). Медиана 3 прогонов после
// warm-up: первый прогон сразу после записи 10k файлов меряет write-back
// кэш ОС, а не наш код.
func TestBudgetFileOps10k(t *testing.T) {
	dir := sessionsFixture10k(t)
	m := NewManager(dir)

	// Warm-up: прогреваем FS-кэш и page cache.
	if _, err := m.LoadAllSessions(); err != nil {
		t.Fatalf("warm-up LoadAllSessions: %v", err)
	}

	const iterations = 3
	ds := make([]time.Duration, 0, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		sessions, err := m.LoadAllSessions()
		if err != nil {
			t.Fatalf("LoadAllSessions: %v", err)
		}
		if len(sessions) != fixtureSessions {
			t.Fatalf("len(sessions) = %d, want %d", len(sessions), fixtureSessions)
		}
		ds = append(ds, time.Since(start))
	}

	sorted := append([]time.Duration(nil), ds...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	median := sorted[len(sorted)/2]

	const budget = time.Second
	failThreshold := 2 * budget
	t.Logf("file ops 10k median=%v (budget %v, %.2fx) — samples: %v",
		median, budget, float64(median)/float64(budget), ds)

	if median > failThreshold {
		t.Errorf("PERF BUDGET VIOLATION: file ops 10k median=%v exceeds 2x budget %v (>2x → CI fails, §6)",
			median, failThreshold)
	} else if median > budget {
		t.Logf("⚠ file ops 10k median=%v exceeds budget %v (%.2fx) — within 2x, warning only",
			median, budget, float64(median)/float64(budget))
	}
}
