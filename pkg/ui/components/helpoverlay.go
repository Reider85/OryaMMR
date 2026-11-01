// helpoverlay.go — компонент help-оверлея (PROMPT-MVP §18.4, §19.5).
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// HelpEntry — одна клавиша с описанием ("[r]", "Refresh").
type HelpEntry struct {
	Key  string
	Desc string
}

// HelpSection — категория клавиш (Navigation / Actions / Screen-specific, §18.4).
type HelpSection struct {
	Title   string
	Entries []HelpEntry
}

// HelpOverlay — модальный help-оверлей экрана. Всегда содержит:
// (1) заголовок с именем экрана,
// (2) все клавиши экрана, сгруппированные по категориям (§18.4),
// (3)глоссарий аббревиатур (ACS/LSP/MCP — §19.5, «no jargon without definition»),
// (4) подсказку закрытия "[?] or [Esc] close" — оверлей никогда не ловит
// пользователя в тупик (INV-UI-010). pre-computed стили (INV-UI-006).
type HelpOverlay struct {
	Screen   string
	Sections []HelpSection

	box     lipgloss.Style
	heading lipgloss.Style
	keyCol  lipgloss.Style
	muted   lipgloss.Style
}

// glossary — глоссарий аббревиатур, обязательный в help (§19.5).
var glossary = []HelpEntry{
	{Key: "ACS", Desc: "Authorship Score — доля кода, написанной с проверкой"},
	{Key: "LSP", Desc: "Language Server Protocol — универсальный фолбэк-адаптер"},
	{Key: "MCP", Desc: "Model Context Protocol — протокол инструментов агента"},
}

// NewHelpOverlay создаёт оверлей для конкретного экрана. theme может быть nil.
func NewHelpOverlay(screen string, sections []HelpSection, theme *ui.Theme) *HelpOverlay {
	if theme == nil {
		theme = ui.NewTheme()
	}
	base := theme.EmptyBox.Copy().Padding(0, 1)
	return &HelpOverlay{
		Screen:   screen,
		Sections: sections,
		box:      base,
		heading:  base.Copy().Bold(true).Foreground(theme.Primary.GetForeground()),
		keyCol:   base.Copy(),
		muted:    base.Copy().Faint(true),
	}
}

// View рендерит оверлей. Формат (§18.4):
//
//	? Help — <screen>
//	<Section>:
//	  <key>  <desc>
//	…
//	Terms:
//	  ACS  <definition>
//	[?] or [Esc] close
func (h *HelpOverlay) View() string {
	lines := []string{h.heading.Render("? Help — " + h.Screen)}

	for _, section := range h.Sections {
		lines = append(lines, "")
		lines = append(lines, h.heading.Render(section.Title+":"))
		for _, e := range section.Entries {
			lines = append(lines, fmt.Sprintf("  %-14s %s", e.Key, e.Desc))
		}
	}

	// Глоссарий: первый уровень иерархии — тот же heading, что и категории.
	lines = append(lines, "")
	lines = append(lines, h.heading.Render("Terms:"))
	for _, g := range glossary {
		lines = append(lines, fmt.Sprintf("  %-14s %s", h.keyCol.Render(g.Key), h.muted.Render(g.Desc)))
	}

	lines = append(lines, "")
	lines = append(lines, h.muted.Render("[?] or [Esc] close"))
	return h.box.Render(strings.Join(lines, "\n"))
}

// AltText — текстовое описание для screen reader (INV-PROMPT-MVP-008, §19.3).
// Статус идёт перед деталями: "HELP: ..." (UI.md §16.3).
func (h *HelpOverlay) AltText() string {
	var b strings.Builder
	b.WriteString("HELP: keyboard shortcuts for " + h.Screen + ". ")
	for _, section := range h.Sections {
		b.WriteString(section.Title + ": ")
		for _, e := range section.Entries {
			b.WriteString(e.Key + " " + e.Desc + ". ")
		}
	}
	b.WriteString("Press ? or Esc to close.")
	return b.String()
}
