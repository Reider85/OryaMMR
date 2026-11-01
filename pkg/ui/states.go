package ui

import (
	"fmt"
)

// State represents the UI state
// INV-PROMPT-MVP-006: Every screen implements ALL 5 mandatory UI states
type State int

const (
	StateLoading State = iota
	StateSuccess
	StateError
	StateEmpty
	StatePartial
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateLoading:
		return "loading"
	case StateSuccess:
		return "success"
	case StateError:
		return "error"
	case StateEmpty:
		return "empty"
	case StatePartial:
		return "partial"
	default:
		return "unknown"
	}
}

// StateMachine governs transitions for every screen
// INV-PROMPT-MVP-006: Every screen embeds StateMachine and calls Transition() on data events
type StateMachine struct {
	current State
	errMsg  string
	partialMsgs []string  // for StatePartial — what's missing
}

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	return &StateMachine{
		current: StateLoading,
		errMsg:  "",
		partialMsgs: []string{},
	}
}

// Current returns the current state
func (sm *StateMachine) Current() State {
	return sm.current
}

// SetState sets the current state directly (for MVP)
func (sm *StateMachine) SetState(state State) {
	sm.current = state
}

// Error returns the error message if in error state
func (sm *StateMachine) Error() string {
	return sm.errMsg
}

// PartialMessages returns the partial state messages
func (sm *StateMachine) PartialMessages() []string {
	return sm.partialMsgs
}

// Transition transitions to a new state based on a message.
// INV-PROMPT-MVP-006 §16.4: transitions are validated against the transition rules.
// Invalid transitions are silently ignored (current state is kept).
func (sm *StateMachine) Transition(msg interface{}) {
	var target State
	switch m := msg.(type) {
	case LoadingMsg:
		target = StateLoading
	case DataLoadedMsg:
		if m.Empty {
			target = StateEmpty
		} else {
			target = StateSuccess
		}
	case DataErrorMsg:
		target = StateError
	case PartialDataMsg:
		target = StatePartial
	case RetryMsg:
		target = StateLoading
	case RefreshMsg:
		target = StateLoading
	default:
		return // неизвестное сообщение — игнорируем
	}

	// INV-PROMPT-MVP-006 §16.4: валидируем переход перед применением
	if err := ValidateTransition(sm.current, target); err != nil {
		return // невалидный переход — сохраняем текущее состояние
	}

	sm.current = target

	// Обновляем связанные данные состояния
	switch m := msg.(type) {
	case DataErrorMsg:
		sm.errMsg = m.Err.Error()
	case PartialDataMsg:
		sm.partialMsgs = m.Missing
	case LoadingMsg, RetryMsg, RefreshMsg:
		// При входе в Loading очищаем предыдущие ошибки/частичные данные
		sm.errMsg = ""
		sm.partialMsgs = []string{}
	}
}

// IsLoading returns true if the current state is loading
func (sm *StateMachine) IsLoading() bool {
	return sm.current == StateLoading
}

// IsSuccess returns true if the current state is success
func (sm *StateMachine) IsSuccess() bool {
	return sm.current == StateSuccess
}

// IsError returns true if the current state is error
func (sm *StateMachine) IsError() bool {
	return sm.current == StateError
}

// IsEmpty returns true if the current state is empty
func (sm *StateMachine) IsEmpty() bool {
	return sm.current == StateEmpty
}

// IsPartial returns true if the current state is partial
func (sm *StateMachine) IsPartial() bool {
	return sm.current == StatePartial
}

// LoadingMsg represents a loading message
type LoadingMsg struct {
	Reason string
}

// DataLoadedMsg represents a data loaded message
type DataLoadedMsg struct {
	Empty bool
	Data  interface{}
}

// DataErrorMsg represents a data error message
type DataErrorMsg struct {
	Err error
}

// PartialDataMsg represents a partial data message
type PartialDataMsg struct {
	Success interface{}
	Missing []string
}

// RetryMsg represents a retry message
type RetryMsg struct {
	Target string
}

// RefreshMsg represents a refresh message
type RefreshMsg struct {
	Force bool
}

// ErrorMsg represents an error message (каноническое определение для всех экранов)
type ErrorMsg struct {
	Err error
}

// HelpMsg represents a help message (каноническое определение для всех экранов)
type HelpMsg struct {
	ScreenID string
	Message  string
}

// Screen represents a screen with state management
type ScreenBase struct {
	StateMachine *StateMachine
	PerfTracker  *PerfTracker
	Theme        *Theme
	Keys         *Keys
	ScreenReader *ScreenReaderManager
}

// NewScreen creates a new screen
func NewScreen(theme *Theme, keys *Keys, a11y *A11y, perfTracker *PerfTracker) *ScreenBase {
	return &ScreenBase{
		StateMachine: NewStateMachine(),
		PerfTracker:  perfTracker,
		Theme:        theme,
		Keys:         keys,
		ScreenReader: NewScreenReaderManager(a11y),
	}
}

// Transition transitions the screen state
func (s *ScreenBase) Transition(msg interface{}) {
	s.StateMachine.Transition(msg)
}

// GetCurrentState returns the current state
func (s *ScreenBase) GetCurrentState() State {
	return s.StateMachine.Current()
}

// GetStateString returns the current state as string
func (s *ScreenBase) GetStateString() string {
	return s.StateMachine.Current().String()
}

// IsState returns true if the screen is in the specified state
func (s *ScreenBase) IsState(state State) bool {
	return s.StateMachine.Current() == state
}

// GetErrorMessage returns the error message if in error state
func (s *ScreenBase) GetErrorMessage() string {
	return s.StateMachine.Error()
}

// GetPartialMessages returns the partial state messages
func (s *ScreenBase) GetPartialMessages() []string {
	return s.StateMachine.PartialMessages()
}

// StateTransitionError represents an error in state transitions
type StateTransitionError struct {
	FromState State
	ToState   State
	Message   string
}

func (e *StateTransitionError) Error() string {
	return fmt.Sprintf("invalid transition from %s to %s: %s", e.FromState, e.ToState, e.Message)
}

// ValidateTransition validates a state transition per INV-PROMPT-MVP-006 §16.4.
// Valid transitions:
//   Loading  → {Success, Error, Empty, Partial}   (never Loading → Loading)
//   Success  → {Loading, Error, Partial}           (never Success → Empty)
//   Error    → {Loading, Error}                    (Error → Error = retry failure)
//   Empty    → {Loading, Success, Empty}           (Empty → Empty = still no data)
//   Partial  → {Loading, Success, Partial}         (never Partial → Error)
func ValidateTransition(from, to State) error {
	validTransitions := map[State][]State{
		StateLoading:  {StateSuccess, StateError, StateEmpty, StatePartial},
		StateSuccess:  {StateLoading, StateError, StatePartial},
		StateError:    {StateLoading, StateError},
		StateEmpty:    {StateLoading, StateSuccess, StateEmpty},
		StatePartial:  {StateLoading, StateSuccess, StatePartial},
	}

	validToStates, exists := validTransitions[from]
	if !exists {
		return &StateTransitionError{
			FromState: from,
			ToState:   to,
			Message:   "unknown from state",
		}
	}

	for _, validTo := range validToStates {
		if to == validTo {
			return nil
		}
	}

	return &StateTransitionError{
		FromState: from,
		ToState:   to,
		Message:   "invalid transition",
	}
}

// GetStateDescription returns a description of the current state
func (s *ScreenBase) GetStateDescription() string {
	switch s.StateMachine.Current() {
	case StateLoading:
		return "Loading..."
	case StateSuccess:
		return "Success"
	case StateError:
		return fmt.Sprintf("Error: %s", s.StateMachine.Error())
	case StateEmpty:
		return "Empty"
	case StatePartial:
		return "Partial"
	default:
		return "Unknown"
	}
}

// GetStateColor returns the color for the current state
func (s *ScreenBase) GetStateColor() string {
	switch s.StateMachine.Current() {
	case StateLoading:
		return "muted"
	case StateSuccess:
		return "secondary"
	case StateError:
		return "error"
	case StateEmpty:
		return "muted"
	case StatePartial:
		return "warning"
	default:
		return "muted"
	}
}

// GetStateIcon returns the icon for the current state
func (s *ScreenBase) GetStateIcon() string {
	switch s.StateMachine.Current() {
	case StateLoading:
		return "⠋"
	case StateSuccess:
		return "✓"
	case StateError:
		return "✗"
	case StateEmpty:
		return "○"
	case StatePartial:
		return "⚠"
	default:
		return "?"
	}
}