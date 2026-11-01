// header.go — компонент заголовка TUI (PROMPT-MVP §4).
package components

import (
	"fmt"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// Header renders the top bar with app name and screen title
// INV-UI-008: "sacred" header — always visible, never scrolled out
func Header(version, screen string) string {
	theme := ui.NewTheme()
	screenTitle := theme.Title.Render(screen)
	headerText := fmt.Sprintf("CodeAuthorityFirst %s — %s", version, screenTitle)
	return theme.Header.Render(headerText)
}

// HeaderWithFocus adds focus indicator to header
// INV-PROMPT-MVP-006: focus always visible (focus ring on header when screen active)
func HeaderWithFocus(version, screen string, isActive bool) string {
	theme := ui.NewTheme()
	screenTitle := theme.Title.Render(screen)
	headerText := fmt.Sprintf("CodeAuthorityFirst %s — %s", version, screenTitle)
	
	if isActive {
		return theme.FocusBorder.Render(headerText)
	}
	return theme.Header.Render(headerText)
}
