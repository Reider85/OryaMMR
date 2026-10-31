// skeleton.go — компонент скелетона загрузки, SWR-паттерн (PROMPT-MVP §4, INV-UI-004).
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// Skeleton — серый плейсхолдер той же формы, что и реальные данные (§16.1).
// Рендер мгновенный и статичный: никакой анимации (INV-PROMPT-MVP-009),
// никакого I/O внутри View() (§6). Показывается сразу при старте загрузки,
// затем DataLoadedMsg мгновенно заменяет его реальными данными (INV-UI-004).
type Skeleton struct {
	Width int // ширина плейсхолдера в символах
	Lines int // число строк-плейсхолдеров

	style lipgloss.Style // pre-computed (INV-UI-006) — не пересчитывается в View()
}

// NewSkeleton создаёт скелетон заданной формы. theme может быть nil —
// тогда берётся палитра по умолчанию. Значения width/lines <= 0 заменяются
// безопасными дефолтами: пустой скелетон = пустой экран, что запрещено (§16.3).
func NewSkeleton(width, lines int, theme *ui.Theme) *Skeleton {
	if width <= 0 {
		width = 20
	}
	if lines <= 0 {
		lines = 3
	}
	if theme == nil {
		theme = ui.NewTheme()
	}
	return &Skeleton{
		Width: width,
		Lines: lines,
		// Muted: плейсхолдер не должен конкурировать с реальными данными (§15.2 п.1).
		style: theme.Muted,
	}
}

// View рендерит Lines строк по Width символов "░". Символ — не цвет —
// несёт смысл плейсхолдера (INV-PROMPT-MVP-008). Строки одинаковой длины:
// после swap'а на реальные данные сетка виджетов не «прыгает» (§17.4).
func (s *Skeleton) View() string {
	row := strings.Repeat("░", s.Width)
	rows := make([]string, s.Lines)
	for i := range rows {
		rows[i] = row
	}
	// Стиль применяем один раз ко всему блоку — не на строку (perf, §6).
	return s.style.Render(strings.Join(rows, "\n"))
}

// AltText — текстовое описание для screen reader (INV-PROMPT-MVP-008, §19.3).
// Статус идёт перед деталями (UI.md §16.3).
func (s *Skeleton) AltText() string {
	return fmt.Sprintf("LOADING: skeleton placeholder, %d lines × %d columns", s.Lines, s.Width)
}
