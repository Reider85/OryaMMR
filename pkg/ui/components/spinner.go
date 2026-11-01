// spinner.go — компонент спиннера асинхронных операций (PROMPT-MVP §4, §20.4).
package components

import (
	"time"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// Spinner represents a loading spinner with accessibility support
type Spinner struct {
	Frames    []string
	Interval  time.Duration
	Current   int
	Animating bool
}

// NewSpinner creates a new spinner with a11y frames
func NewSpinner() *Spinner {
	a11y := ui.NewA11y(ui.NewAccessibilityConfig(), ui.NewTheme(), ui.NewKeys())
	return &Spinner{
		Frames:    a11y.SpinnerFrames(),
		Interval:  a11y.SpinnerFrameInterval(),
		Animating: false,
	}
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	s.Animating = true
	s.Current = 0
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	s.Animating = false
}

// NextFrame returns the next frame of the spinner
func (s *Spinner) NextFrame() string {
	if !s.Animating || len(s.Frames) == 0 {
		return ""
	}
	
	frame := s.Frames[s.Current]
	s.Current = (s.Current + 1) % len(s.Frames)
	return frame
}

// Render returns the current frame of the spinner
func (s *Spinner) Render() string {
	if !s.Animating {
		return ""
	}
	return s.NextFrame()
}

// Reset resets the spinner to initial state
func (s *Spinner) Reset() {
	s.Current = 0
	s.Animating = false
}

// IsAnimating returns whether the spinner is currently animating
func (s *Spinner) IsAnimating() bool {
	return s.Animating
}

// SetAnimating sets the animation state
func (s *Spinner) SetAnimating(animating bool) {
	s.Animating = animating
	if animating {
		s.Current = 0
	}
}
