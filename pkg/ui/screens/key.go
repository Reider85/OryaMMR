// key.go — диспетчер печатных клавиш для TUI-экранов (PROMPT-MVP §18.4).
package screens

import "github.com/charmbracelet/bubbletea"

// runeOf возвращает одиночный символ нажатой клавиши либо 0.
//
// В bubbletea v0.24 печатная клавиша приходит как KeyMsg{Type: KeyRunes,
// Runes: [...}}, а НЕ как Type = значение символа. Поэтому switch по msg.Type
// со строковыми-рунными case ('r', 'q', '?', ...) никогда не срабатывал —
// клавиши refresh/quit/help были мёртвым кодом (INV-UI-009, INV-PROMPT-MVP-007
// касается видимости фокуса, §18.4 — полноты клавиатуры). runeOf — единая
// точка, через которую экраны получают символ корректно.
func runeOf(msg tea.KeyMsg) rune {
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && !msg.Alt {
		return msg.Runes[0]
	}
	return 0
}
