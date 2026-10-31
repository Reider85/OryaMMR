// a11y_test.go — тесты accessibility helpers (PROMPT-MVP §19, §18.1).
// Имена тестов намеренно уникальны относительно интеграционных файлов
// (tui_test.go и др. под //go:build integration), чтобы -tags integration
// не приводил к коллизии имён.
package ui

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func newTestA11y() *A11y {
	return NewA11y(NewAccessibilityConfig(), NewTheme(), NewKeys())
}

// --- Alt-text (§19.3) ------------------------------------------------------

func TestA11ySparklineAltText(t *testing.T) {
	rising := SparklineAltText(3.2, 3.8, 30)
	if !strings.Contains(rising, "30-day trend: 3.2 → 3.8 (+0.6, rising)") {
		t.Errorf("rising alt-text mismatch: %q", rising)
	}

	falling := SparklineAltText(3.8, 3.2, 30)
	if !strings.Contains(falling, "falling") {
		t.Errorf("falling direction missing: %q", falling)
	}

	flat := SparklineAltText(3.5, 3.5, 7)
	if !strings.Contains(flat, "(+0.0, flat)") {
		t.Errorf("flat direction missing: %q", flat)
	}

	defDays := SparklineAltText(1, 2, 0)
	if !strings.HasPrefix(defDays, "30-day trend:") {
		t.Errorf("default days should be 30: %q", defDays)
	}
}

func TestA11yProgressAltText(t *testing.T) {
	got := ProgressAltText(12, 12450, 100000, "tokens", "daily cap")
	want := "12% of daily cap (12,450 / 100,000 tokens)"
	if got != want {
		t.Errorf("ProgressAltText() = %q, want %q", got, want)
	}

	// Дефолты: unit=units, label=daily cap
	def := ProgressAltText(50, 1, 2, "", "")
	if !strings.Contains(def, "50% of daily cap (1 / 2 units)") {
		t.Errorf("defaults mismatch: %q", def)
	}

	// Отрицательные числа и >1000 без разделителя ошибок
	if ProgressAltText(100, 999, 999, "x", "cap") != "100% of cap (999 / 999 x)" {
		t.Errorf("small numbers should have no separators: %q", ProgressAltText(100, 999, 999, "x", "cap"))
	}
}

func TestA11yStatusAltText(t *testing.T) {
	withDetail := StatusAltText("opencode", "healthy", "last check 2s ago")
	if withDetail != "opencode: healthy, last check 2s ago" {
		t.Errorf("with detail mismatch: %q", withDetail)
	}
	noDetail := StatusAltText("claude_code", "unknown", "")
	if noDetail != "claude_code: unknown" {
		t.Errorf("without detail mismatch: %q", noDetail)
	}
}

func TestA11yRenderAlt(t *testing.T) {
	a := newTestA11y()

	// Обычный TTY-профиль → визуал
	a.SetOutputProfile(termenv.TrueColor)
	if got := a.RenderAlt("▁▂▃", "30-day trend: rising"); got != "▁▂▃" {
		t.Errorf("visual mode should return visual, got %q", got)
	}

	// Ascii (пайп/screen reader) → alt-text
	a.SetOutputProfile(termenv.Ascii)
	if got := a.RenderAlt("▁▂▃", "30-day trend: rising"); got != "30-day trend: rising" {
		t.Errorf("ascii mode should return alt-text, got %q", got)
	}

	// Ascii без alt → визуал (не пустой экран)
	if got := a.RenderAlt("▁▂▃", ""); got != "▁▂▃" {
		t.Errorf("empty alt should fall back to visual, got %q", got)
	}
}

func TestA11yOutputProfileOverride(t *testing.T) {
	a := newTestA11y()
	a.SetOutputProfile(termenv.ANSI)
	if a.OutputProfile() != termenv.ANSI {
		t.Errorf("override not applied")
	}
	if a.IsAsciiOutput() {
		t.Errorf("ANSI is not ascii")
	}
	a.SetOutputProfile(termenv.Ascii)
	if !a.IsAsciiOutput() {
		t.Errorf("Ascii override should set IsAsciiOutput")
	}
}

// --- Color-blind safety (§19.1) ---------------------------------------------

func TestA11yStatusSymbol(t *testing.T) {
	cases := map[string]string{
		"healthy":   "✓",
		"ok":        "✓",
		"degraded":  "⚠",
		"warning":   "⚠",
		"unhealthy": "✗",
		"error":     "✗",
		"failed":    "✗",
		"passed":    "✓",
		"native":    "✓",
		"partial":   "⚠",
		"mystery":   "?",
		"":          "?",
	}
	for status, want := range cases {
		if got := StatusSymbol(status); got != want {
			t.Errorf("StatusSymbol(%q) = %q, want %q", status, got, want)
		}
	}
	// Регистронезависимость
	if StatusSymbol("HEALTHY") != "✓" {
		t.Errorf("StatusSymbol should be case-insensitive")
	}
}

func TestA11yStatusPrefixHighContrast(t *testing.T) {
	a := newTestA11y()

	// Обычный режим — префиксов нет
	if a.StatusPrefix("healthy") != "" || a.StatusPrefix("error") != "" {
		t.Errorf("default mode should not add prefixes")
	}

	a.Config.HighContrast = true
	if got := a.StatusPrefix("healthy"); got != "+" {
		t.Errorf("HC healthy prefix = %q, want +", got)
	}
	if got := a.StatusPrefix("degraded"); got != "!" {
		t.Errorf("HC warning prefix = %q, want !", got)
	}
	if got := a.StatusPrefix("unhealthy"); got != "X" {
		t.Errorf("HC error prefix = %q, want X", got)
	}
	if got := a.StatusPrefix("mystery"); got != "?" {
		t.Errorf("HC unknown prefix = %q, want ?", got)
	}
}

func TestA11yStatusPair(t *testing.T) {
	a := newTestA11y()
	got := a.StatusPair("healthy", "healthy")
	if got != "✓ healthy" {
		t.Errorf("default pair = %q, want ✓ healthy", got)
	}

	a.Config.HighContrast = true
	hc := a.StatusPair("unhealthy", "unhealthy")
	if !strings.Contains(hc, "X") || !strings.Contains(hc, "✗") || !strings.Contains(hc, "unhealthy") {
		t.Errorf("HC pair must contain prefix+symbol+text, got %q", hc)
	}
}

// --- Focus ring (§18.1, INV-PROMPT-MVP-007) ---------------------------------

func TestA11yFocusTitle(t *testing.T) {
	if got := FocusTitle("Today", true); got != "▸ Today" {
		t.Errorf("focused title = %q", got)
	}
	if got := FocusTitle("Today", false); got != "Today" {
		t.Errorf("unfocused title = %q", got)
	}
}

func TestA11yFocusBorderStyleProfiles(t *testing.T) {
	theme := NewTheme()

	// TrueColor → двойная рамка (видимые Unicode-символы в render)
	tc := RenderFocusBox("widget", true, theme, termenv.TrueColor)
	if !HasFocusMarker(tc) {
		t.Errorf("truecolor focus render must contain focus marker, got %q", tc)
	}
	if !strings.Contains(tc, "═") && !strings.Contains(tc, "╔") {
		t.Errorf("truecolor focus should use DoubleBorder, got %q", tc)
	}

	// ANSI256 → thick border
	a256 := RenderFocusBox("widget", true, theme, termenv.ANSI256)
	if !strings.Contains(a256, "━") && !strings.Contains(a256, "┏") && !HasFocusMarker(a256) {
		t.Errorf("ansi256 focus should use ThickBorder or marker, got %q", a256)
	}

	// Неактивный виджет — без маркера фокуса
	unfocused := RenderFocusBox("widget", false, theme, termenv.TrueColor)
	if HasFocusMarker(unfocused) {
		t.Errorf("unfocused widget must not have focus marker, got %q", unfocused)
	}
}

func TestA11yHasFocusMarker(t *testing.T) {
	if !HasFocusMarker("▸ Dashboard") {
		t.Errorf("marker prefix not detected")
	}
	if !HasFocusMarker("border ═ content") {
		t.Errorf("double border not detected")
	}
	if !HasFocusMarker("thick ━ content") {
		t.Errorf("thick border not detected")
	}
	if !HasFocusMarker("\x1b[7mreverse\x1b[0m") {
		t.Errorf("reverse video not detected")
	}
	if HasFocusMarker("plain widget content") {
		t.Errorf("plain content must not match")
	}
}

func TestA11yFocusBorderStyle(t *testing.T) {
	if FocusBorderStyle(termenv.TrueColor) != lipgloss.DoubleBorder() {
		t.Errorf("truecolor should use DoubleBorder")
	}
	if FocusBorderStyle(termenv.ANSI256) != lipgloss.ThickBorder() {
		t.Errorf("ansi256 should use ThickBorder")
	}
	if FocusBorderStyle(termenv.ANSI) != lipgloss.NormalBorder() {
		t.Errorf("ansi should use NormalBorder")
	}
	if FocusBorderStyle(termenv.Ascii) != lipgloss.NormalBorder() {
		t.Errorf("ascii should use NormalBorder")
	}
}

func TestA11ySendEscapeSequenceStderr(t *testing.T) {
	a := newTestA11y()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	a.SendEscapeSequence("[?1h")
	w.Close()
	os.Stderr = old
	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "\x1b[?1h") {
		t.Errorf("escape sequence not written to stderr: %q", string(out))
	}
}

func TestA11yFocusManagerWrap(t *testing.T) {
	a := newTestA11y()
	fm := NewFocusManager(a)

	if fm.MoveForward() != "" {
		t.Errorf("empty focus manager should return empty string")
	}
	if fm.GetFocusedElement() != "" {
		t.Errorf("empty focus manager should have no focus")
	}
	if len(fm.GetFocusPath()) != 0 {
		t.Errorf("empty focus manager path should be empty")
	}

	fm.AddFocusableElement("today")
	fm.AddFocusableElement("score")
	fm.AddFocusableElement("agents")

	if fm.MoveForward() != "score" {
		t.Errorf("first forward should land on score")
	}
	if fm.MoveForward() != "agents" {
		t.Errorf("second forward should land on agents")
	}
	// wrap
	if fm.MoveForward() != "today" {
		t.Errorf("forward should wrap to today")
	}
	if fm.MoveBackward() != "agents" {
		t.Errorf("backward should wrap to agents")
	}

	if !fm.SetFocus("score") {
		t.Errorf("SetFocus should find score")
	}
	if fm.GetFocusedElement() != "score" {
		t.Errorf("focused element should be score")
	}
	if fm.SetFocus("nonexistent") {
		t.Errorf("SetFocus should fail for unknown id")
	}
	path := fm.GetFocusPath()
	if len(path) != 2 || path[1] != "score" {
		t.Errorf("focus path = %v", path)
	}
}

// --- High-contrast (§19.2) --------------------------------------------------

func TestA11yGetThemeHighContrast(t *testing.T) {
	// Принудительно truecolor: в тестовом окружении lipgloss strip-ит цвета
	// без TTY, и стили рендерились бы одинаково.
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	a := newTestA11y()
	if a.GetTheme() != a.Theme {
		t.Errorf("default should return base theme")
	}
	a.Config.HighContrast = true
	if !a.IsHighContrast() {
		t.Errorf("IsHighContrast should be true")
	}
	hc := a.GetTheme()
	// Проверяем, что high-contrast палитра отличается (белый primary вместо синего)
	if hc.Primary.Render("x") == a.Theme.Primary.Render("x") {
		t.Errorf("high-contrast theme should differ from base theme")
	}
}

func TestA11yContrastRatio(t *testing.T) {
	// WCAG эталон: чёрный на белом = 21.0
	if r := ContrastRatio("#000000", "#FFFFFF"); r < 20.9 || r > 21.1 {
		t.Errorf("black/white ratio = %f, want ~21", r)
	}
	// Симметрия
	if ContrastRatio("#FFFFFF", "#000000") != ContrastRatio("#000000", "#FFFFFF") {
		t.Errorf("ratio must be symmetric")
	}
	// Одинаковый цвет = 1.0
	if r := ContrastRatio("#61AFEF", "#61AFEF"); r < 0.99 || r > 1.01 {
		t.Errorf("same color ratio = %f, want 1.0", r)
	}
	// Короткий hex
	if r := ContrastRatio("#000", "#FFF"); r < 20.9 || r > 21.1 {
		t.Errorf("#000/#FFF ratio = %f, want ~21", r)
	}
	// Некорректный вход не паникует
	if r := ContrastRatio("bogus", "#FFFFFF"); r != 1.0 {
		t.Errorf("invalid fg should return 1.0, got %f", r)
	}
	if r := ContrastRatio("#FFFFFF", "nothex!"); r != 1.0 {
		t.Errorf("invalid bg should return 1.0, got %f", r)
	}
}

func TestA11yValidateColorContrast(t *testing.T) {
	a := newTestA11y()
	// Белый на чёрном — точно AAA (21:1)
	if !a.ValidateColorContrast("#FFFFFF", "#000000") {
		t.Errorf("white/black must pass AAA")
	}
	// Слабый контраст серого на чёрном
	if a.ValidateColorContrast("#333333", "#000000") {
		t.Errorf("#333/#000 (~2.3:1) must fail AAA")
	}
	// Палитра MVP (theme.go §0.4) обязана проходить AA 4.5 к тёмному фону
	for _, hex := range []string{"#61AFEF", "#98C379", "#E5C07B", "#F07178", "#C678DD"} {
		if r := ContrastRatio(hex, "#282C34"); r < 4.5 {
			t.Errorf("palette %s on dark bg = %f, below AA", hex, r)
		}
	}
}

// --- Reduce-motion (§19.4) --------------------------------------------------

func TestA11yMotion(t *testing.T) {
	a := newTestA11y()
	if !a.ShouldAnimate() {
		t.Errorf("default should animate")
	}
	if a.TransitionFrames() != 8 {
		t.Errorf("default transition frames = %d, want 8", a.TransitionFrames())
	}
	if len(a.SpinnerFrames()) <= 1 {
		t.Errorf("animated spinner should have multiple frames")
	}
	if a.SpinnerFrameInterval() != 100*time.Millisecond {
		t.Errorf("spinner interval = %v, want 100ms", a.SpinnerFrameInterval())
	}

	a.Config.ReducedMotion = true
	if a.ShouldAnimate() {
		t.Errorf("reduced motion should not animate")
	}
	if a.TransitionFrames() != 0 {
		t.Errorf("reduced motion transition frames = %d, want 0", a.TransitionFrames())
	}
	frames := a.SpinnerFrames()
	if len(frames) != 1 || frames[0] != "…" {
		t.Errorf("reduced motion spinner = %v, want static …", frames)
	}
}

// --- Screen reader (§19.3, UI.md §16.3) -------------------------------------

func TestA11yAnnouncementTextStatusFirst(t *testing.T) {
	a := newTestA11y()

	// Screen reader выключен → пустая строка
	if a.GetAnnouncementText("FAILED", "check 9") != "" {
		t.Errorf("screen reader disabled should return empty")
	}

	a.Config.ScreenReader = true
	got := a.GetAnnouncementText("FAILED", "check 9")
	if got != "FAILED: check 9" {
		t.Errorf("status-first announcement = %q", got)
	}
}

func TestA11yAnnounceWritesStderr(t *testing.T) {
	a := newTestA11y()
	a.Config.ScreenReader = true

	// Перехватываем stderr
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	a.Announce(AnnouncementWarning, "check 9 timeout")
	w.Close()
	os.Stderr = old

	out, _ := io.ReadAll(r)
	text := string(out)
	if !strings.Contains(text, "[WARNING]") || !strings.Contains(text, "check 9 timeout") {
		t.Errorf("stderr announcement = %q", text)
	}

	// Screen reader off → ничего не пишется
	r2, w2, _ := os.Pipe()
	os.Stderr = w2
	a.Config.ScreenReader = false
	a.Announce(AnnouncementCritical, "should not appear")
	w2.Close()
	os.Stderr = old
	out2, _ := io.ReadAll(r2)
	if len(out2) != 0 {
		t.Errorf("disabled screen reader wrote to stderr: %q", string(out2))
	}
}

func TestA11yAnnouncementLevelString(t *testing.T) {
	if AnnouncementCritical.String() != "CRITICAL" || AnnouncementWarning.String() != "WARNING" ||
		AnnouncementImportant.String() != "IMPORTANT" || AnnouncementNone.String() != "INFO" {
		t.Errorf("announcement level strings mismatch")
	}
}

func TestA11yScreenReaderManager(t *testing.T) {
	a := newTestA11y()
	srm := NewScreenReaderManager(a)

	if srm.GetLatestAnnouncement() != nil {
		t.Errorf("empty manager should have no latest")
	}

	srm.AddAnnouncement("a1", "dashboard loaded", AnnouncementImportant, "Dashboard", false)
	srm.Announce(AnnouncementWarning, "cross-repo stale", "Dashboard", true)

	all := srm.GetAnnouncements()
	if len(all) != 2 {
		t.Fatalf("announcements = %d, want 2", len(all))
	}
	latest := srm.GetLatestAnnouncement()
	if latest == nil || latest.Message != "cross-repo stale" || latest.Context != "Dashboard" {
		t.Errorf("latest = %+v", latest)
	}

	srm.ClearAnnouncements()
	if len(srm.GetAnnouncements()) != 0 {
		t.Errorf("clear failed")
	}
}

// --- Validate / report ------------------------------------------------------

func TestA11yValidateAccessibilityFindings(t *testing.T) {
	a := newTestA11y()
	// Keys.Tab заполнен (NewKeys), screen reader выключен — ошибок нет
	if errs := a.ValidateAccessibility(a.Theme); len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}

	// Screen reader включён, но объявления выключены → ошибка
	a.Config.ScreenReader = true
	a.Config.AnnounceChanges = false
	errs := a.ValidateAccessibility(a.Theme)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "announcements disabled") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected announcements-disabled error, got %v", errs)
	}

	// Пустой Tab → ошибка keyboard navigation
	a.Config.ScreenReader = false
	a.Config.AnnounceChanges = true
	brokenKeys := &Keys{}
	broken := NewA11y(a.Config, NewTheme(), brokenKeys)
	errs = broken.ValidateAccessibility(broken.Theme)
	found = false
	for _, e := range errs {
		if strings.Contains(e, "Tab key not configured") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected tab-key error, got %v", errs)
	}
}

func TestA11yGetAccessibilityReport(t *testing.T) {
	a := newTestA11y()
	a.Config.HighContrast = true
	a.Config.ReducedMotion = true
	report := a.GetAccessibilityReport()
	for _, want := range []string{
		"High contrast mode enabled",
		"Reduced motion enabled",
		"Screen reader support disabled",
		"Keyboard-only operation enabled",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q", want)
		}
	}
}

func TestA11yGetAccessibilityReportWithErrors(t *testing.T) {
	a := newTestA11y()
	a.Config.ScreenReader = true
	a.Config.AnnounceChanges = false
	report := a.GetAccessibilityReport()
	if !strings.Contains(report, "Errors:") || !strings.Contains(report, "✗") {
		t.Errorf("report should include error section, got %q", report)
	}
}

// --- Легаси-совместимость (tui.go, шаги 12-13) ------------------------------

func TestA11yLegacyHelpers(t *testing.T) {
	a := newTestA11y()
	if a.GetFontSize() != "14px" {
		t.Errorf("default font size")
	}
	a.Config.LargeText = true
	if !a.IsLargeText() || a.GetFontSize() != "16px" {
		t.Errorf("large text mode")
	}
	if len(a.GetTabOrder()) == 0 {
		t.Errorf("tab order should not be empty")
	}
	if len(a.GetEscapeSequences()) == 0 {
		t.Errorf("escape sequences should not be empty")
	}
	if a.ValidateInput("ok", 10) != nil {
		t.Errorf("valid input rejected")
	}
	if a.ValidateInput(strings.Repeat("x", 11), 10) == nil {
		t.Errorf("oversized input accepted")
	}
	if a.ValidateInput("bad\x00char", 10) == nil {
		t.Errorf("control char accepted")
	}
	if a.ValidateInput("tabs\tand\nnewlines ok", 100) != nil {
		t.Errorf("tab/newline should be allowed")
	}
	shortcuts := a.GetKeyboardShortcuts()
	if !strings.Contains(shortcuts, "Tab:") || !strings.Contains(shortcuts, "q: Quit") {
		t.Errorf("keyboard shortcuts text mismatch")
	}
}
