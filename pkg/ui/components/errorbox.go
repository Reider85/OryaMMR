// errorbox.go — компонент состояния error (PROMPT-MVP §4, §16.1).
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// ErrorBox — рамка состояния error (§16.3 Error). Всегда содержит:
// (1) сообщение простым языком (без стек-трейсов и сырых Go-ошибок),
// (2) CTA "[r] retry",
// (3) подсказку "[Esc] dismiss",
// опционально строку Fix для восстановимых ошибок (например, прав доступа).
// Никогда не показывает ошибку без пути восстановления (INV-UI-010).
// Символ "✗" всегда присутствует рядом с цветом (INV-PROMPT-MVP-008).
type ErrorBox struct {
	Message  string // сообщение простым языком (обязательно)
	Fix      string // предложенный способ исправления (опционально)
	RetryKey string // клавиша повтора, по умолчанию "r"

	// pre-computed стили (INV-UI-006) — не пересчитываются в View().
	// box несёт рамку+фон; fix/cta — только акцент внутри бокса (без padding,
	// иначе отступ удвоится).
	box     lipgloss.Style
	fixLine lipgloss.Style
	ctaLine lipgloss.Style
}

// NewErrorBox создаёт ErrorBox. theme/a11y могут быть nil — берутся дефолты.
// RetryKey всегда "r" (универсальная клавиша refresh, UI.md §0.6; §16.3 требует
// "[r] retry" как обязательный CTA состояния error).
func NewErrorBox(message string, theme *ui.Theme, a11y *ui.A11y) *ErrorBox {
	if theme == nil {
		theme = ui.NewTheme()
	}
	if a11y == nil {
		a11y = ui.NewA11y(ui.NewAccessibilityConfig(), theme, ui.NewKeys())
	}
	// High-contrast режим берёт усиленную палитру (§19.2).
	t := a11y.GetTheme()
	base := t.ErrorBox.Copy().Padding(0, 1)
	return &ErrorBox{
		Message:  message,
		RetryKey: "r",
		box:      base,
		fixLine:  base.Copy().Faint(true),
		ctaLine:  base.Copy().Bold(true),
	}
}

// SetFix задаёт предложенный способ исправления восстановимой ошибки (§16.3).
func (e *ErrorBox) SetFix(fix string) {
	e.Fix = fix
}

// View рендерит бокс ошибки. Формат (§16.3):
//
//	✗ Error
//	<message>
//	Fix: <fix>              — только если задан
//	[r] retry  [Esc] dismiss
//
// Рендер статичный, без I/O (§6); transition-анимация — забота экрана (шаг 13).
func (e *ErrorBox) View() string {
	// Символ + текст: цвет никогда не единственный носитель смысла (INV-PROMPT-MVP-008).
	lines := []string{"✗ Error", e.Message}
	if e.Fix != "" {
		lines = append(lines, e.fixLine.Render("Fix: "+e.Fix))
	}
	// CTA: жирная строка — клавиши восстановления заметны с первого взгляда (§18.4).
	lines = append(lines, e.ctaLine.Render(fmt.Sprintf("[%s] retry   [Esc] dismiss", e.RetryKey)))
	return e.box.Render(strings.Join(lines, "\n"))
}

// AltText — текстовое описание для screen reader (INV-PROMPT-MVP-008, §19.3).
// Статус идёт перед деталями (UI.md §16.3): "ERROR: ...", а не "... — ERROR".
func (e *ErrorBox) AltText() string {
	txt := fmt.Sprintf("ERROR: %s. Press %s to retry, Esc to dismiss.", e.Message, e.RetryKey)
	if e.Fix != "" {
		txt += " Suggested fix: " + e.Fix
	}
	return txt
}
