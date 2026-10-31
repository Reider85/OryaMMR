// emptycta.go — компонент empty-with-CTA (PROMPT-MVP §4, §16.1).
package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// EmptyCTA — рамка состояния empty (§16.3 Empty-with-CTA). Всегда содержит 4 части:
// (1) что пусто ("No sessions yet"),
// (2) почему ("caf hasn't been invoked in this repo"),
// (3) что сделать — CTA в цвете Primary (§2.6 Empty-state CTA),
// (4) клавиатурную подсказку ("[i] run caf init now").
// Пустой фрейм без объяснения запрещён (INV-UI-010, INV-PROMPT-MVP-006).
// Символ "○" + текст несут смысл без цвета (INV-PROMPT-MVP-008).
type EmptyCTA struct {
	What   string // что пусто (обязательно)
	Why    string // почему пусто (обязательно)
	Action string // что сделать — CTA (обязательно)
	Hint   string // клавиатурная подсказка, напр. "[i] run caf init now"

	// pre-computed стили (INV-UI-006) — не пересчитываются в View()
	box       lipgloss.Style
	ctaStyle  lipgloss.Style
	hintStyle lipgloss.Style
}

// NewEmptyCTA создаёт EmptyCTA. theme может быть nil — берётся дефолт.
// Все 4 строки обязательны (§16.3): пустые строки заменяются безопасными
// заглушками, чтобы экран никогда не рендерился без объяснения.
func NewEmptyCTA(what, why, action, hint string, theme *ui.Theme) *EmptyCTA {
	if theme == nil {
		theme = ui.NewTheme()
	}
	if what == "" {
		what = "Nothing here yet"
	}
	if why == "" {
		why = "No data available"
	}
	if action == "" {
		action = "Run `caf init` to start tracking"
	}
	if hint == "" {
		hint = "[i] run caf init now"
	}
	return &EmptyCTA{
		What:   what,
		Why:    why,
		Action: action,
		Hint:   hint,
		box:    theme.EmptyBox.Copy().Padding(0, 1),
		// CTA в Primary (§2.6): следующий шаг должен бросаться в глаза.
		ctaStyle:  theme.EmptyBox.Copy().Padding(0, 1).Foreground(theme.Primary.GetForeground()).Bold(true),
		hintStyle: theme.EmptyBox.Copy().Padding(0, 1).Faint(true),
	}
}

// View рендерит бокс empty-состояния. Формат (§16.3):
//
//	○ <what>
//	<why>
//	<action>          — CTA, Primary + Bold
//	<hint>            — клавиатурная подсказка, Muted
func (e *EmptyCTA) View() string {
	// Символ + текст: цвет никогда не единственный носитель смысла (INV-PROMPT-MVP-008).
	return e.box.Render(strings.Join([]string{
		"○ " + e.What,
		e.Why,
		e.ctaStyle.Render(e.Action),
		e.hintStyle.Render(e.Hint),
	}, "\n"))
}

// AltText — текстовое описание для screen reader (INV-PROMPT-MVP-008, §19.3).
// Статус идёт перед деталями (UI.md §16.3).
func (e *EmptyCTA) AltText() string {
	return "EMPTY: " + e.What + ". " + e.Why + ". Next: " + e.Action + ". " + e.Hint
}
