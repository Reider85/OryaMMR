// bench_test.go — бенчмарк LSP startup против fake-gopls (PROMPT-MVP §6,
// INV-ADR-013: soft budget ~500ms).
//
// §6 бюджет: LSP startup ~500ms. Enforcement (>2x → fail): fail при median
// > 1s (§6:255). Реальный gopls не требуется — re-exec fake-gopls харнес
// из fallback_test.go (installFakeGopls).
package lsp

import (
	"sort"
	"testing"
	"time"
)

// medianDuration возвращает медиану (без упорядочивания входа).
func medianDuration(ds []time.Duration) time.Duration {
	sorted := append([]time.Duration(nil), ds...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)/2]
}

// BenchmarkLSPStartup — NewFallbackManager + Start + Stop на итерацию.
// Start = handshake (initialize) + best-effort начальный синк графа.
func BenchmarkLSPStartup(b *testing.B) {
	installFakeGopls(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := NewFallbackManager(b.TempDir())
		if err := m.Start(); err != nil {
			b.Fatalf("Start: %v", err)
		}
		if err := m.Stop(); err != nil {
			b.Fatalf("Stop: %v", err)
		}
	}
}

// TestBudgetLSPStartup — enforcement бюджета §6: median startup ≤ 1s
// (2x soft-budget 500ms INV-ADR-013). 5 итераций — каждый замер спавнит
// процесс fake-gopls, больше не нужно.
func TestBudgetLSPStartup(t *testing.T) {
	installFakeGopls(t)

	const softBudget = 500 * time.Millisecond
	failThreshold := 2 * softBudget

	const iterations = 5
	ds := make([]time.Duration, 0, iterations)
	for i := 0; i < iterations; i++ {
		m := NewFallbackManager(t.TempDir())
		start := time.Now()
		if err := m.Start(); err != nil {
			t.Fatalf("Start: %v", err)
		}
		ds = append(ds, time.Since(start))
		if err := m.Stop(); err != nil {
			t.Fatalf("Stop: %v", err)
		}
	}

	median := medianDuration(ds)
	ratio := float64(median) / float64(softBudget)
	t.Logf("LSP startup median=%v (soft budget %v, %.2fx) — samples: %v", median, softBudget, ratio, ds)

	if median > failThreshold {
		t.Errorf("PERF BUDGET VIOLATION: LSP startup median=%v exceeds 2x budget %v (>2x → CI fails, §6)",
			median, failThreshold)
	} else if median > softBudget {
		t.Logf("⚠ LSP startup median=%v exceeds soft budget %v (%.2fx) — within 2x, warning only (INV-ADR-013 soft budget)",
			median, softBudget, ratio)
	}
}
