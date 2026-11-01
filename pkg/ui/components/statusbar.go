// statusbar.go — компонент нижней строки статуса (PROMPT-MVP §4, §15.2 п.7).
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// StatusBar represents the status bar with screen info and key hints
type StatusBar struct {
	ScreenName    string
	Focus         string
	KeyHints      []string
	RefreshState  string
	LastRefresh   time.Time
	Theme         *ui.Theme
}

// NewStatusBar creates a new status bar
func NewStatusBar(screenName string) *StatusBar {
	return &StatusBar{
		ScreenName:   screenName,
		Theme:        ui.NewTheme(),
		RefreshState: "", // idle
	}
}

// SetRefreshState sets the refresh indicator state
func (sb *StatusBar) SetRefreshState(state string) {
	sb.RefreshState = state
	if state != "" {
		sb.LastRefresh = time.Now()
	}
}

// Render renders the status bar
func (sb *StatusBar) Render() string {
	theme := sb.Theme
	
	// Left side: screen name and focus
	left := fmt.Sprintf("%s focus: %s", sb.ScreenName, sb.Focus)
	
	// Right side: key hints and refresh indicator
	rightParts := []string{}
	for _, hint := range sb.KeyHints {
		rightParts = append(rightParts, hint)
	}
	
	if sb.RefreshState != "" {
		rightParts = append(rightParts, sb.RefreshState)
	}
	
	right := strings.Join(rightParts, "  ")
	
	// Combine with padding
	padding := 100 - len(left) - len(right)
	if padding < 0 {
		padding = 0
	}
	content := fmt.Sprintf("%s%s%s", left, strings.Repeat(" ", padding), right)
	
	return theme.StatusBar.Render(content)
}

// SetKeyHints sets keyboard hints for the current screen
func (sb *StatusBar) SetKeyHints(hints []string) {
	sb.KeyHints = hints
}

// SetFocus sets the current focus element
func (sb *StatusBar) SetFocus(focus string) {
	sb.Focus = focus
}

// SetRefreshIndicator sets the refresh indicator
func (sb *StatusBar) SetRefreshIndicator(format string, args ...interface{}) {
	if format == "" {
		sb.RefreshState = ""
		return
	}
	sb.RefreshState = fmt.Sprintf(format, args...)
}

// SetRefreshCompleted sets the refresh completed indicator
func (sb *StatusBar) SetRefreshCompleted() {
	if sb.LastRefresh.IsZero() {
		sb.RefreshState = ""
		return
	}
	sb.RefreshState = fmt.Sprintf("✓ refreshed at %s", sb.LastRefresh.Format("15:04:05"))
}
