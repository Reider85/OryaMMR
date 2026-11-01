// perf_test.go — тесты бюджетов производительности и PerfTracker (§6, INV-UI-001..003).
package ui

import (
	"strings"
	"testing"
	"time"
)

func TestPerformanceBudgetsMatchSection6(t *testing.T) {
	b := NewPerformanceBudget()
	if b.FirstPaint != 100*time.Millisecond {
		t.Errorf("FirstPaint = %v, want 100ms (INV-UI-001)", b.FirstPaint)
	}
	if b.FrameRender != 16*time.Millisecond {
		t.Errorf("FrameRender = %v, want 16ms (INV-UI-002)", b.FrameRender)
	}
	if b.Keystroke != 50*time.Millisecond {
		t.Errorf("Keystroke = %v, want 50ms (INV-UI-003)", b.Keystroke)
	}
}

func TestPerfTrackerStartEndAndBudget(t *testing.T) {
	pt := NewPerfTracker(NewPerformanceBudget())

	pt.Start("frame_render")
	d := pt.End("frame_render")
	if d < 0 {
		t.Errorf("duration must be non-negative, got %v", d)
	}
	if got, ok := pt.GetDuration("frame_render"); !ok || got != d {
		t.Errorf("GetDuration = (%v, %v), want (%v, true)", got, ok, d)
	}
	if _, ok := pt.GetDuration("never-measured"); ok {
		t.Error("GetDuration for unknown op must report not-found")
	}

	within, _, budget := pt.CheckBudget("frame_render")
	if !within {
		t.Errorf("immediate frame_render must be within %s budget", budget)
	}
	if w, dur, b := pt.CheckBudget("frame_render"); w && (dur > b) {
		t.Errorf("withinBudget inconsistent: %v > %v", dur, b)
	}

	if within, _, _ := pt.CheckBudget("unknown-op"); within {
		t.Error("unknown operation must not report within-budget")
	}
	if within, _, _ := pt.CheckBudget("no-measure"); within {
		t.Error("unmeasured operation must not report within-budget")
	}
}

func TestPerfTrackerOverBudgetBranches(t *testing.T) {
	pt := NewPerfTracker(NewPerformanceBudget())
	// Тест в том же пакете — измерения вносятся напрямую для ветки «превышено».
	pt.measures["first_paint"] = 500 * time.Millisecond
	pt.measures["frame_render"] = 1 * time.Millisecond

	within, dur, budget := pt.CheckBudget("first_paint")
	if within {
		t.Errorf("500ms must exceed %s first-paint budget", budget)
	}
	if dur != 500*time.Millisecond {
		t.Errorf("duration = %v, want 500ms", dur)
	}

	report := pt.GetReport()
	if !strings.Contains(report, "✗ First paint") {
		t.Errorf("report must flag exceeded first paint, got:\n%s", report)
	}
	if !strings.Contains(report, "✓ Frame render") {
		t.Errorf("report must mark within-budget frame render, got:\n%s", report)
	}
}

func TestMeasurePerformanceCmd(t *testing.T) {
	msg := MeasurePerformance("first_paint")()
	perf, ok := msg.(PerfMsg)
	if !ok {
		t.Fatalf("MeasurePerformance must produce PerfMsg, got %T", msg)
	}
	if perf.Name != "first_paint" {
		t.Errorf("Name = %q", perf.Name)
	}
	if perf.Timestamp.IsZero() {
		t.Error("Timestamp must be set")
	}
}

func TestPerformanceWarning(t *testing.T) {
	critical := NewPerformanceWarning("first_paint", 40*time.Millisecond, 10*time.Millisecond)
	if !critical.IsCritical() {
		t.Error("4x budget must be critical (>2x, INV-UI-001 enforcement)")
	}
	if got := critical.String(); !strings.HasPrefix(got, "CRITICAL:") || !strings.Contains(got, "4.0x") {
		t.Errorf("critical String = %q", got)
	}

	warn := NewPerformanceWarning("keystroke", 60*time.Millisecond, 50*time.Millisecond)
	if warn.IsCritical() {
		t.Error("1.2x budget must not be critical")
	}
	if got := warn.String(); !strings.HasPrefix(got, "WARNING:") {
		t.Errorf("warning String = %q", got)
	}
}

func TestEnforceBudget(t *testing.T) {
	warning, ok := EnforceBudget("frame_render", time.Millisecond, 16*time.Millisecond)
	if !ok || warning != nil {
		t.Errorf("within budget = (%v, %v), want (nil, true)", warning, ok)
	}
	warning, ok = EnforceBudget("frame_render", time.Second, 16*time.Millisecond)
	if ok || warning == nil {
		t.Fatalf("over budget = (%v, %v), want (warning, false)", warning, ok)
	}
	if warning.Ratio <= 2.0 {
		t.Errorf("1s/16ms ratio = %v, want > 2", warning.Ratio)
	}
}

func TestBenchmarkHelperSelectsBudgetByName(t *testing.T) {
	fast := func() time.Duration { return time.Millisecond }

	if _, warning, within := Benchmark("first_paint", fast); !within || warning != nil {
		t.Errorf("fast first_paint = (warning=%v, within=%v), want (nil, true)", warning, within)
	}
	if _, warning, within := Benchmark("first_paint", func() time.Duration { return 500 * time.Millisecond }); within || warning == nil {
		t.Errorf("slow first_paint = (warning=%v, within=%v), want (warning, false)", warning, within)
	}
	// Неизвестное имя падает на file_op-бюджет (1s).
	if _, warning, within := Benchmark("custom-op", func() time.Duration { return 2 * time.Second }); within || warning == nil {
		t.Errorf("slow custom-op = (warning=%v, within=%v), want over-budget", warning, within)
	}
}

func TestOptimizeForPerformance(t *testing.T) {
	pt := NewPerfTracker(NewPerformanceBudget())
	// Ничего не измерено → все три базовых бюджета «превышены» (not-found = false).
	suggestions := OptimizeForPerformance(pt)
	if len(suggestions) < 3 {
		t.Errorf("fresh tracker must suggest optimizations, got %v", suggestions)
	}
	if len(suggestions) < 7 {
		t.Errorf("want ≥7 suggestions (2 first-paint + 3 frame + 2 keystroke), got %d", len(suggestions))
	}

	// Всё в бюджете → оптимизации не нужны.
	pt2 := NewPerfTracker(NewPerformanceBudget())
	pt2.measures["first_paint"] = time.Millisecond
	pt2.measures["frame_render"] = time.Millisecond
	pt2.measures["keystroke"] = time.Millisecond
	if got := OptimizeForPerformance(pt2); len(got) != 0 {
		t.Errorf("within-budget tracker must have no suggestions, got %v", got)
	}
}
