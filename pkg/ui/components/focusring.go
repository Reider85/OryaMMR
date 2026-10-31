// focusring.go — видимый индикатор фокуса (PROMPT-MVP §4, INV-PROMPT-MVP-007).
package components

import (
	"github.com/muesli/termenv"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// Focusable — виджет, участвующий в таб-кольце (§18.2).
// Каждый мульти-виджетный экран оборачивает свои виджеты в FocusRing.
type Focusable interface {
	ID() string
	SetFocus(bool)
	IsFocused() bool
	View() string
}

// FocusRing — таб-кольцо виджетов (§18.2). Tab вызывает Next(), Shift+Tab —
// Prev(); кольцо циклически заворачивается в обе стороны. Фокус всегда видим
// (INV-PROMPT-MVP-007): текущий виджет рендерится с маркером "▸ " и рамкой
// фокуса (§18.1), остальные — с приглушённой рамкой.
type FocusRing struct {
	widgets []Focusable
	current int
}

// NewFocusRing создаёт кольцо из переданных виджетов. Первый виджет получает
// фокус сразу — экран никогда не стартует без видимого фокуса (INV-PROMPT-MVP-007).
func NewFocusRing(widgets ...Focusable) *FocusRing {
	fr := &FocusRing{widgets: widgets}
	if len(fr.widgets) > 0 {
		fr.widgets[fr.current].SetFocus(true)
	}
	return fr
}

// Next переводит фокус на следующий виджет с wrap-around (§18.2).
// Пустое кольцо → nil (без паники, §7 fail-safe дух).
func (fr *FocusRing) Next() Focusable {
	if len(fr.widgets) == 0 {
		return nil
	}
	fr.widgets[fr.current].SetFocus(false)
	fr.current = (fr.current + 1) % len(fr.widgets)
	fr.widgets[fr.current].SetFocus(true)
	return fr.widgets[fr.current]
}

// Prev переводит фокус на предыдущий виджет с wrap-around (§18.2).
func (fr *FocusRing) Prev() Focusable {
	if len(fr.widgets) == 0 {
		return nil
	}
	fr.widgets[fr.current].SetFocus(false)
	fr.current = (fr.current - 1 + len(fr.widgets)) % len(fr.widgets)
	fr.widgets[fr.current].SetFocus(true)
	return fr.widgets[fr.current]
}

// Current возвращает сфокусированный виджет (или nil для пустого кольца).
func (fr *FocusRing) Current() Focusable {
	if len(fr.widgets) == 0 {
		return nil
	}
	return fr.widgets[fr.current]
}

// Len возвращает число виджетов в кольце.
func (fr *FocusRing) Len() int {
	return len(fr.widgets)
}

// Index возвращает индекс текущего фокуса (0 для пустого кольца).
func (fr *FocusRing) Index() int {
	return fr.current
}

// RenderFocus оборачивает содержимое виджета видимой рамкой фокуса (§18.1):
// focused → BorderDouble/Thick/Normal+Reverse по профилю терминала,
// не focused → RoundedBorder + Muted. Визуал делегируется хелперам шага 11
// (ui.FocusBorder / ui.UnfocusedBorder) — без дублирования.
func RenderFocus(content string, focused bool, theme *ui.Theme, profile termenv.Profile) string {
	if theme == nil {
		theme = ui.NewTheme()
	}
	return ui.RenderFocusBox(content, focused, theme, profile)
}

// RenderFocusTitle добавляет маркер "▸ " к заголовку сфокусированного виджета
// (§18.1, 16-color ветка; маркер виден на любом профиле — «виден с 3 метров»,
// INV-PROMPT-MVP-007).
func RenderFocusTitle(title string, focused bool) string {
	return ui.FocusTitle(title, focused)
}

// HasFocusMarker — хелпер focus-visibility тестов (§10): проверяет, что в
// отрендеренном виджете есть видимый маркер фокуса (INV-PROMPT-MVP-007).
func HasFocusMarker(rendered string) bool {
	return ui.HasFocusMarker(rendered)
}
