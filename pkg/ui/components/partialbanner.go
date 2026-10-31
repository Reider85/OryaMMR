// partialbanner.go — компонент состояния partial-failure (PROMPT-MVP §16.1, §16.3).
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// PartialBanner — баннер состояния partial (§16.3 Partial-failure).
// Показывается ВЕРХУ экрана поверх реальных данных: часть данных загружена,
// часть — нет. Пользователь всегда знает, чего не хватает (никогда не
// деградируем молча — INV-PROMPT-MVP-006). Клавиша "[r] retry" повторяет
// загрузку. Символ "⚠" + текст несут смысл без цвета (INV-PROMPT-MVP-008).
type PartialBanner struct {
	Loaded  int      // сколько виджетов/частей загружено успешно
	Total   int      // сколько ожидалось всего
	Missing []string // что именно не загрузилось (с описанием причины)

	// pre-computed стили (INV-UI-006) — не пересчитываются в View()
	box      lipgloss.Style
	ctaStyle lipgloss.Style
}

// NewPartialBanner создаёт баннер partial-состояния. theme может быть nil —
// берётся дефолт. Счётчики нормализуются: Total < Loaded приводится к Loaded,
// отрицательные значения — к нулю (баннер не должен врать об успешном).
func NewPartialBanner(loaded, total int, missing []string, theme *ui.Theme) *PartialBanner {
	if theme == nil {
		theme = ui.NewTheme()
	}
	if loaded < 0 {
		loaded = 0
	}
	if total < loaded {
		total = loaded
	}
	return &PartialBanner{
		Loaded:   loaded,
		Total:    total,
		Missing:  missing,
		box:      theme.Warning.Copy().Border(lipgloss.NormalBorder(), true).Padding(0, 1),
		ctaStyle: theme.Warning.Copy().Border(lipgloss.NormalBorder(), true).Padding(0, 1).Bold(true),
	}
}

// View рендерит баннер. Формат (§16.3):
//
//	⚠ Partial data: <loaded> of <total> loaded.
//	Missing: <item1>, <item2>     — только если список не пуст
//	[r] retry
//
// Реальные данные рендерятся экраном ПОД баннером — баннер их не заменяет.
func (p *PartialBanner) View() string {
	// Символ + текст: цвет никогда не единственный носитель смысла (INV-PROMPT-MVP-008).
	lines := []string{
		fmt.Sprintf("⚠ Partial data: %d of %d loaded.", p.Loaded, p.Total),
	}
	if len(p.Missing) > 0 {
		lines = append(lines, "Missing: "+strings.Join(p.Missing, ", "))
	}
	lines = append(lines, p.ctaStyle.Render("[r] retry"))
	return p.box.Render(strings.Join(lines, "\n"))
}

// AltText — текстовое описание для screen reader (INV-PROMPT-MVP-008, §19.3).
// Статус идёт перед деталями (UI.md §16.3).
func (p *PartialBanner) AltText() string {
	txt := fmt.Sprintf("PARTIAL: %d of %d loaded.", p.Loaded, p.Total)
	if len(p.Missing) > 0 {
		txt += " Missing: " + strings.Join(p.Missing, ", ") + "."
	}
	return txt + " Press r to retry."
}
