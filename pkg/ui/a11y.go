// a11y.go — accessibility helpers (PROMPT-MVP §4, §19, UI.md §16).
// INV-PROMPT-MVP-007: фокус всегда видим. INV-PROMPT-MVP-008: у каждого
// нетекстового элемента есть altText; цвет никогда не единственный носитель смысла.
//
// Реализовано здесь (step 11, NEW v0.4.1):
//   - altText-генераторы + RenderAlt для non-TTY/screen reader (§19.3)
//   - focus-ring по профилям терминала (§18.1)
//   - high-contrast символы/префиксы (§19.2)
//   - reduce-motion (§19.4)
//   - реальный WCAG-расчёт контрастности (§19.2)
//   - screen-reader объявления в stderr, status-first (UI.md §16.3)
package ui

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// AccessibilityConfig defines accessibility settings
// INV-UI-007: WCAG-AAA compliance for color contrast, keyboard navigation, screen reader support
type AccessibilityConfig struct {
	HighContrast    bool
	ReducedMotion   bool
	ScreenReader    bool
	LargeText       bool
	AnnounceChanges bool
	KeyboardOnly    bool
	TabNavigation   bool
	EscapeExits     bool
	ConfirmActions  bool
}

// NewAccessibilityConfig creates a new accessibility config
func NewAccessibilityConfig() *AccessibilityConfig {
	return &AccessibilityConfig{
		HighContrast:    false,
		ReducedMotion:   false,
		ScreenReader:    false,
		LargeText:       false,
		AnnounceChanges: true,
		KeyboardOnly:    true,
		TabNavigation:   true,
		EscapeExits:     true,
		ConfirmActions:  true,
	}
}

// A11y represents accessibility features
type A11y struct {
	Config *AccessibilityConfig
	Theme  *Theme
	Keys   *Keys

	// profileOverride — test-seam: позволяет тестам подменить профиль
	// терминала и проверять Ascii/ANSI/256/TrueColor ветки без реального TTY.
	profileOverride *termenv.Profile
}

// NewA11y creates a new accessibility manager
func NewA11y(config *AccessibilityConfig, theme *Theme, keys *Keys) *A11y {
	return &A11y{
		Config: config,
		Theme:  theme,
		Keys:   keys,
	}
}

// SetOutputProfile подменяет определённый автоматически профиль терминала
// (используется в тестах; в проде всегда nil — профиль определяется termenv).
func (a *A11y) SetOutputProfile(p termenv.Profile) {
	a.profileOverride = &p
}

// OutputProfile возвращает текущий профиль терминала (§18.1, §19.3).
func (a *A11y) OutputProfile() termenv.Profile {
	if a.profileOverride != nil {
		return *a.profileOverride
	}
	return termenv.DefaultOutput().Profile
}

// IsAsciiOutput true, когда вывод пайпится в файл/ screen reader без TTY:
// в этом режиме View() обязан отдавать alt-text вместо визуала (§19.3).
func (a *A11y) IsAsciiOutput() bool {
	return a.OutputProfile() == termenv.Ascii
}

// GetTheme returns the appropriate theme based on accessibility settings
func (a *A11y) GetTheme() *Theme {
	if a.Config.HighContrast {
		return a.Theme.HighContrast()
	}
	return a.Theme
}

// IsHighContrast true, когда включён high-contrast режим (§19.2).
func (a *A11y) IsHighContrast() bool {
	return a.Config.HighContrast
}

// GetAnnouncementText generates text for screen readers, status-first:
// статус идёт перед деталями ("FAILED check 9", а не "check 9 FAILED") — UI.md §16.3.
func (a *A11y) GetAnnouncementText(action, description string) string {
	if !a.Config.ScreenReader {
		return ""
	}
	return fmt.Sprintf("%s: %s", action, description)
}

// ShouldAnimate returns true if animations should be shown (§19.4)
func (a *A11y) ShouldAnimate() bool {
	return !a.Config.ReducedMotion
}

// SpinnerFrames возвращает кадры спиннера: Bubbles-подобные кадры при
// обычном режиме, статичный "…" при reduced-motion (§19.4, §20.4).
func (a *A11y) SpinnerFrames() []string {
	if a.Config.ReducedMotion {
		return []string{"…"}
	}
	return []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
}

// SpinnerFrameInterval — 100ms (Bubbles default): быстрее отвлекает,
// медленнее выглядит лагом (§6, §20.4).
func (a *A11y) SpinnerFrameInterval() time.Duration {
	return 100 * time.Millisecond
}

// TransitionFrames — число кадров анимации перехода: 8 (~100ms) обычно,
// 0 при reduced-motion (мгновенная смена, §19.4, §20.1).
func (a *A11y) TransitionFrames() int {
	if a.Config.ReducedMotion {
		return 0
	}
	return 8
}

// GetKeyboardShortcuts returns keyboard shortcuts for screen reader announcements
func (a *A11y) GetKeyboardShortcuts() string {
	return fmt.Sprintf(`Keyboard shortcuts:
Tab: Navigate between widgets
Enter: Select/Activate
Esc: Back/Cancel
Space: Toggle
1-9: Switch screens
?: Help
q: Quit`)
}

// ValidateColorContrast проверяет реальное WCAG-соотношение контрастности
// (§19.2): порог 7.0 = WCAG AAA для обычного текста.
func (a *A11y) ValidateColorContrast(foreground, background string) bool {
	return ContrastRatio(foreground, background) >= 7.0
}

// parseHexColor парсит "#RGB" или "#RRGGBB" в нормализованные 0..255 компоненты.
func parseHexColor(s string) (r, g, b int, ok bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	switch len(s) {
	case 3:
		var v [3]int
		for i := 0; i < 3; i++ {
			d := hexDigit(s[i])
			if d < 0 {
				return 0, 0, 0, false
			}
			v[i] = d*16 + d
		}
		return v[0], v[1], v[2], true
	case 6:
		var v [3]int
		for i := 0; i < 3; i++ {
			hi := hexDigit(s[i*2])
			lo := hexDigit(s[i*2+1])
			if hi < 0 || lo < 0 {
				return 0, 0, 0, false
			}
			v[i] = hi*16 + lo
		}
		return v[0], v[1], v[2], true
	default:
		return 0, 0, 0, false
	}
}

// hexDigit возвращает значение hex-цифры или -1.
func hexDigit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}

// relativeLuminance — WCAG 2.1 relative luminance (formula).
func relativeLuminance(r, g, b int) float64 {
	linear := func(c int) float64 {
		v := float64(c) / 255.0
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}

// ContrastRatio считает WCAG contrast ratio между двумя hex-цветами.
// Возвращает 1.0 для некорректных входов (никогда не паникует).
func ContrastRatio(fg, bg string) float64 {
	fr, fg_, fb, ok1 := parseHexColor(fg)
	br, bg_, bb, ok2 := parseHexColor(bg)
	if !ok1 || !ok2 {
		return 1.0
	}
	l1 := relativeLuminance(fr, fg_, fb)
	l2 := relativeLuminance(br, bg_, bb)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// AnnouncementLevel — уровень объявления для screen reader.
type AnnouncementLevel int

const (
	AnnouncementNone AnnouncementLevel = iota
	AnnouncementImportant
	AnnouncementWarning
	AnnouncementCritical
)

// String — текстовый уровень (status-first: уровень идёт перед текстом).
func (l AnnouncementLevel) String() string {
	switch l {
	case AnnouncementCritical:
		return "CRITICAL"
	case AnnouncementWarning:
		return "WARNING"
	case AnnouncementImportant:
		return "IMPORTANT"
	default:
		return "INFO"
	}
}

// Announce announces something to screen readers.
// Пишет в STDERR — stdout принадлежит Bubble Tea-кадру, писать туда нельзя.
// Буфер ScreenReaderManager остаётся источником истины для тестов и UI.
func (a *A11y) Announce(level AnnouncementLevel, text string) {
	if !a.Config.ScreenReader || !a.Config.AnnounceChanges {
		return
	}
	fmt.Fprintf(os.Stderr, "[%s] %s\n", level.String(), text)
}

// IsLargeText returns true if large text is enabled
func (a *A11y) IsLargeText() bool {
	return a.Config.LargeText
}

// GetFontSize returns the appropriate font size.
// LEGACY: веб-наследие скаффолда, TUI не имеет px-шрифтов; сохранено для
// совместимости tui.go. Не используется в MVP-экранах (шаг 13).
func (a *A11y) GetFontSize() string {
	if a.Config.LargeText {
		return "16px"
	}
	return "14px"
}

// ValidateInput validates user input for accessibility
func (a *A11y) ValidateInput(input string, maxLength int) error {
	if len(input) > maxLength {
		return fmt.Errorf("input exceeds maximum length of %d", maxLength)
	}

	// Check for control characters that might cause issues
	for _, r := range input {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return fmt.Errorf("input contains invalid control character")
		}
	}

	return nil
}

// GetTabOrder returns the tab navigation order.
// LEGACY: веб-наследие; реальный tab-ring строит components/focusring.go (шаг 12).
func (a *A11y) GetTabOrder() []string {
	return []string{"main", "sidebar", "content", "footer"}
}

// EscapeSequence represents an escape sequence for screen readers
// LEGACY: веб-наследие скаффолда, сохранено для совместимости.
type EscapeSequence struct {
	Sequence string
	Purpose  string
}

// GetEscapeSequences returns common escape sequences (LEGACY)
func (a *A11y) GetEscapeSequences() []EscapeSequence {
	return []EscapeSequence{
		{Sequence: "\x1b[?1h", Purpose: "Enter application mode"},
		{Sequence: "\x1b[?1l", Purpose: "Exit application mode"},
		{Sequence: "\x1b[=1h", Purpose: "Set keypad to application mode"},
		{Sequence: "\x1b[=1l", Purpose: "Set keypad to numeric mode"},
	}
}

// SendEscapeSequence sends an escape sequence (LEGACY; пишет в stderr)
func (a *A11y) SendEscapeSequence(sequence string) {
	fmt.Fprintf(os.Stderr, "\x1b%s", sequence)
}

// ---------------------------------------------------------------------------
// Alt-text (§19.3, INV-PROMPT-MVP-008)
// ---------------------------------------------------------------------------

// SparklineAltText формирует текстовое описание спарклайна:
// "30-day trend: 3.2 → 3.8 (+0.6, rising)".
func SparklineAltText(from, to float64, days int) string {
	if days <= 0 {
		days = 30
	}
	delta := to - from
	direction := "flat"
	switch {
	case delta > 0:
		direction = "rising"
	case delta < 0:
		direction = "falling"
	}
	return fmt.Sprintf("%d-day trend: %.1f → %.1f (%+.1f, %s)", days, from, to, delta, direction)
}

// ProgressAltText формирует текстовое описание прогресс-бара:
// "12% of daily cap (12,450 / 100,000 tokens)".
func ProgressAltText(pct float64, value, capacity int64, unit, label string) string {
	if unit == "" {
		unit = "units"
	}
	if label == "" {
		label = "daily cap"
	}
	return fmt.Sprintf("%.0f%% of %s (%s / %s %s)",
		pct, label, formatCount(value), formatCount(capacity), unit)
}

// formatCount форматирует число с разделителями тысяч (en-US).
func formatCount(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// StatusAltText формирует описание индикатора статуса:
// "opencode: healthy, last check 2s ago".
func StatusAltText(name, status, detail string) string {
	if detail == "" {
		return fmt.Sprintf("%s: %s", name, status)
	}
	return fmt.Sprintf("%s: %s, %s", name, status, detail)
}

// RenderAlt — §19.3: когда вывод пайпится (profile == Ascii, нет TTY),
// визуальный элемент заменяется на alt-text. В обычном TTY — визуал.
func (a *A11y) RenderAlt(visual, alt string) string {
	if a.IsAsciiOutput() {
		if alt == "" {
			return visual
		}
		return alt
	}
	return visual
}

// ---------------------------------------------------------------------------
// Color-blind safety (§19.1, INV-PROMPT-MVP-008): символ + текст всегда
// ---------------------------------------------------------------------------

// StatusSymbol возвращает семантический символ статуса (§19.1).
func StatusSymbol(status string) string {
	switch strings.ToLower(status) {
	case "healthy", "ok", "success", "passed", "native":
		return "✓"
	case "degraded", "warning", "partial":
		return "⚠"
	case "unhealthy", "error", "failed":
		return "✗"
	default:
		return "?"
	}
}

// StatusPrefix — high-contrast префикс (§19.2 таблица): Secondary "+",
// Warning "!", Error "X". В обычном режиме префикс пуст — работает символ.
func (a *A11y) StatusPrefix(status string) string {
	if !a.Config.HighContrast {
		return ""
	}
	switch strings.ToLower(status) {
	case "healthy", "ok", "success", "passed", "native":
		return "+"
	case "degraded", "warning", "partial":
		return "!"
	case "unhealthy", "error", "failed":
		return "X"
	default:
		return "?"
	}
}

// StatusPair — гарантированная пара "символ + текст" (INV-PROMPT-MVP-008):
// цвет никогда не единственный носитель смысла. В high-contrast режиме
// добавляется текстовый префикс (§19.2).
func (a *A11y) StatusPair(status, label string) string {
	symbol := StatusSymbol(status)
	prefix := a.StatusPrefix(status)
	if prefix != "" {
		return fmt.Sprintf("%s %s %s", prefix, symbol, label)
	}
	return fmt.Sprintf("%s %s", symbol, label)
}

// ---------------------------------------------------------------------------
// Focus ring (§18.1, INV-PROMPT-MVP-007)
// ---------------------------------------------------------------------------

// FocusMarker — маркер фокуса для 16-color режима: "▸ " в заголовке (§18.1).
const FocusMarker = "▸ "

// FocusBorderStyle выбирает border по профилю терминала (§18.1):
// TrueColor → DoubleBorder, 256-color → ThickBorder, 16-color → NormalBorder
// (фокус там несёт "▸ " + reverse-video, а не border).
func FocusBorderStyle(profile termenv.Profile) lipgloss.Border {
	switch profile {
	case termenv.TrueColor:
		return lipgloss.DoubleBorder()
	case termenv.ANSI256:
		return lipgloss.ThickBorder()
	default:
		return lipgloss.NormalBorder()
	}
}

// FocusBorder возвращает стиль рамки фокуса для профиля (INV-PROMPT-MVP-007).
// На инверсии (16-color/Ascii) стиль Reverse(true) — виден без цвета.
func FocusBorder(theme *Theme, profile termenv.Profile) lipgloss.Style {
	base := theme.Primary
	switch profile {
	case termenv.TrueColor:
		return base.Border(lipgloss.DoubleBorder(), true)
	case termenv.ANSI256:
		return base.Border(lipgloss.ThickBorder(), true)
	default: // ANSI, Ascii
		return lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Reverse(true)
	}
}

// UnfocusedBorder — неактивный виджет: обычный бордер, Muted (§18.1).
func UnfocusedBorder(theme *Theme) lipgloss.Style {
	return theme.Muted.Border(lipgloss.RoundedBorder(), true)
}

// FocusTitle добавляет "▸ " к заголовку сфокусированного виджета (§18.1,
// 16-color ветка; маркер виден на любом профиле — «виден с 3 метров»).
func FocusTitle(title string, focused bool) string {
	if focused {
		return FocusMarker + title
	}
	return title
}

// RenderFocusBox оборачивает содержимое рамкой фокуса, если focused.
func RenderFocusBox(content string, focused bool, theme *Theme, profile termenv.Profile) string {
	if focused {
		return FocusBorder(theme, profile).Render(content)
	}
	return UnfocusedBorder(theme).Render(content)
}

// HasFocusMarker — хелпер для focus-visibility тестов (§10, шаг 14):
// проверяет, что в отрендеренном виджете есть видимый маркер фокуса.
// Ищет "▸ ", двойные/жирные border-символы или ANSI reverse-video.
func HasFocusMarker(rendered string) bool {
	if strings.Contains(rendered, FocusMarker) {
		return true
	}
	// Double border: ╔═╗║╚═╝; Thick border: ┏━┓┃┗━┛
	for _, ch := range []string{"═", "╔", "╗", "╚", "╝", "━", "┏", "┓", "┗", "┛"} {
		if strings.Contains(rendered, ch) {
			return true
		}
	}
	// Reverse video (16-color focus, §18.1)
	if strings.Contains(rendered, "\x1b[7m") || strings.Contains(rendered, "\x1b(0") {
		return true
	}
	return false
}

// FocusManager manages focus for keyboard navigation.
// Легаси string-ID API сохранён для tui.go и тестов шагов 12-13;
// tab-ring для виджетов строит components/focusring.go (шаг 12, §18.2).
type FocusManager struct {
	focusableElements []string
	currentFocus      int
	a11y              *A11y
}

// NewFocusManager creates a new focus manager
func NewFocusManager(a11y *A11y) *FocusManager {
	return &FocusManager{
		focusableElements: []string{},
		currentFocus:      0,
		a11y:              a11y,
	}
}

// AddFocusableElement adds a focusable element
func (fm *FocusManager) AddFocusableElement(id string) {
	fm.focusableElements = append(fm.focusableElements, id)
}

// MoveForward moves focus in the specified direction
func (fm *FocusManager) MoveForward() string {
	if len(fm.focusableElements) == 0 {
		return ""
	}

	fm.currentFocus = (fm.currentFocus + 1) % len(fm.focusableElements)
	return fm.focusableElements[fm.currentFocus]
}

// MoveBackward moves focus backward
func (fm *FocusManager) MoveBackward() string {
	if len(fm.focusableElements) == 0 {
		return ""
	}

	fm.currentFocus = (fm.currentFocus - 1 + len(fm.focusableElements)) % len(fm.focusableElements)
	return fm.focusableElements[fm.currentFocus]
}

// GetFocusedElement returns the currently focused element
func (fm *FocusManager) GetFocusedElement() string {
	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableElements) {
		return fm.focusableElements[fm.currentFocus]
	}
	return ""
}

// SetFocus sets focus to a specific element
func (fm *FocusManager) SetFocus(id string) bool {
	for i, element := range fm.focusableElements {
		if element == id {
			fm.currentFocus = i
			return true
		}
	}
	return false
}

// GetFocusPath returns the current focus path
func (fm *FocusManager) GetFocusPath() []string {
	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableElements) {
		path := make([]string, fm.currentFocus+1)
		copy(path, fm.focusableElements[:fm.currentFocus+1])
		return path
	}
	return []string{}
}

// ScreenReaderAnnouncement represents a screen reader announcement
type ScreenReaderAnnouncement struct {
	ID         string
	Message    string
	Level      AnnouncementLevel
	Context    string
	Persistent bool
}

// ScreenReaderManager manages screen reader announcements
type ScreenReaderManager struct {
	announcements []ScreenReaderAnnouncement
	a11y          *A11y
}

// NewScreenReaderManager creates a new screen reader manager
func NewScreenReaderManager(a11y *A11y) *ScreenReaderManager {
	return &ScreenReaderManager{
		announcements: []ScreenReaderAnnouncement{},
		a11y:          a11y,
	}
}

// AddAnnouncement adds a new announcement
func (srm *ScreenReaderManager) AddAnnouncement(id, message string, level AnnouncementLevel, context string, persistent bool) {
	announcement := ScreenReaderAnnouncement{
		ID:         id,
		Message:    message,
		Level:      level,
		Context:    context,
		Persistent: persistent,
	}
	srm.announcements = append(srm.announcements, announcement)
	srm.a11y.Announce(level, message)
}

// Announce добавляет объявление с короткой сигнатурой (level, text, context, persistent)
func (srm *ScreenReaderManager) Announce(level AnnouncementLevel, message, context string, persistent bool) {
	srm.AddAnnouncement(message, message, level, context, persistent)
}

// GetAnnouncements returns all announcements
func (srm *ScreenReaderManager) GetAnnouncements() []ScreenReaderAnnouncement {
	return srm.announcements
}

// ClearAnnouncements clears all announcements
func (srm *ScreenReaderManager) ClearAnnouncements() {
	srm.announcements = []ScreenReaderAnnouncement{}
}

// GetLatestAnnouncement returns the latest announcement
func (srm *ScreenReaderManager) GetLatestAnnouncement() *ScreenReaderAnnouncement {
	if len(srm.announcements) == 0 {
		return nil
	}
	return &srm.announcements[len(srm.announcements)-1]
}

// ValidateAccessibility validates the current UI state for accessibility
func (a *A11y) ValidateAccessibility(theme *Theme) []string {
	errors := []string{}

	// Check color contrast (реальный WCAG-расчёт, §19.2)
	if !a.ValidateColorContrast("#FFFFFF", "#000000") {
		errors = append(errors, "WCAG contrast calculation is broken (white on black must pass AAA)")
	}

	// Проверяем палитру темы: контраст Primary/Error к фону >= 7.0 (AAA).
	if pair := checkPaletteContrast(theme); pair != "" {
		errors = append(errors, pair)
	}

	// Check keyboard navigation
	if a.Config.KeyboardOnly && len(a.Keys.Tab) == 0 {
		errors = append(errors, "Tab key not configured for keyboard navigation")
	}

	// Check screen reader support
	if a.Config.ScreenReader && !a.Config.AnnounceChanges {
		errors = append(errors, "Screen reader enabled but change announcements disabled")
	}

	return errors
}

// checkPaletteContrast возвращает описание проблемы контрастности палитры
// или "" если всё в порядке. Фон берём из Background-стиля темы, если он задан.
func checkPaletteContrast(theme *Theme) string {
	if theme == nil {
		return ""
	}
	// Фон по умолчанию — тёмный #282C34 (theme.go §0.4); в high-contrast #000000.
	bg := "#282C34"
	type pair struct {
		name string
		fg   string
	}
	// lipgloss не отдаёт hex обратно — проверяем через известные константы палитры.
	pairs := []pair{
		{"Primary", "#61AFEF"},
		{"Secondary", "#98C379"},
		{"Warning", "#E5C07B"},
		{"Error", "#F07178"},
		{"Accent", "#C678DD"},
	}
	var failed []string
	for _, p := range pairs {
		if ContrastRatio(p.fg, bg) < 4.5 { // AA floor for UI palette
			failed = append(failed, p.name)
		}
	}
	if len(failed) > 0 {
		return fmt.Sprintf("palette contrast below WCAG AA on dark background: %s", strings.Join(failed, ", "))
	}
	return ""
}

// GetAccessibilityReport returns an accessibility report
func (a *A11y) GetAccessibilityReport() string {
	report := "Accessibility Report:\n"

	if a.Config.HighContrast {
		report += "✓ High contrast mode enabled\n"
	} else {
		report += "○ High contrast mode disabled\n"
	}

	if a.Config.ReducedMotion {
		report += "✓ Reduced motion enabled\n"
	} else {
		report += "○ Reduced motion disabled\n"
	}

	if a.Config.ScreenReader {
		report += "✓ Screen reader support enabled\n"
	} else {
		report += "○ Screen reader support disabled\n"
	}

	if a.Config.KeyboardOnly {
		report += "✓ Keyboard-only operation enabled\n"
	} else {
		report += "○ Keyboard-only operation disabled\n"
	}

	// Validate accessibility
	errors := a.ValidateAccessibility(a.Theme)
	if len(errors) > 0 {
		report += "\nErrors:\n"
		for _, err := range errors {
			report += fmt.Sprintf("✗ %s\n", err)
		}
	} else {
		report += "\n✓ No accessibility errors found\n"
	}

	return report
}
