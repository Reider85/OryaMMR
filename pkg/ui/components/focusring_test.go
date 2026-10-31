package components

import (
	"testing"

	"github.com/muesli/termenv"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

// testWidget — минимальный Focusable для тестов кольца (§18.2).
type testWidget struct {
	id      string
	focused bool
}

func (w *testWidget) ID() string      { return w.id }
func (w *testWidget) SetFocus(f bool) { w.focused = f }
func (w *testWidget) IsFocused() bool { return w.focused }
func (w *testWidget) View() string    { return "content of " + w.id }

func widgets(ids ...string) []*testWidget {
	out := make([]*testWidget, len(ids))
	for i, id := range ids {
		out[i] = &testWidget{id: id}
	}
	return out
}

// ─── NewFocusRing: фокус на первом виджете сразу (INV-PROMPT-MVP-007) ───────

func TestNewFocusRing_FirstWidgetFocused(t *testing.T) {
	ws := widgets("a", "b", "c")
	fr := NewFocusRing(toFocusable(ws)...)

	if got := fr.Current().ID(); got != "a" {
		t.Errorf("Current().ID() = %q, want \"a\"", got)
	}
	if !ws[0].IsFocused() {
		t.Error("first widget must be focused on construction (экран никогда не стартует без видимого фокуса)")
	}
}

// ─── Next/Prev: циклический обход в обе стороны (§18.2) ─────────────────────

func TestFocusRing_Next_CyclesWithWrap(t *testing.T) {
	ws := widgets("a", "b", "c")
	fr := NewFocusRing(toFocusable(ws)...)

	for i, want := range []string{"b", "c", "a", "b"} {
		got := fr.Next().ID()
		if got != want {
			t.Errorf("Next() #%d = %q, want %q", i+1, got, want)
		}
	}
}

func TestFocusRing_Prev_CyclesWithWrap(t *testing.T) {
	ws := widgets("a", "b", "c")
	fr := NewFocusRing(toFocusable(ws)...)

	// с "a" назад — wrap к "c"
	if got := fr.Prev().ID(); got != "c" {
		t.Errorf("Prev() from a = %q, want \"c\" (wrap-around)", got)
	}
	if got := fr.Prev().ID(); got != "b" {
		t.Errorf("Prev() #2 = %q, want \"b\"", got)
	}
}

func TestFocusRing_OnlyOneWidgetFocused_AfterNavigation(t *testing.T) {
	ws := widgets("a", "b", "c")
	fr := NewFocusRing(toFocusable(ws)...)

	fr.Next() // a → b
	fr.Next() // b → c
	fr.Prev() // c → b

	focused := 0
	for _, w := range ws {
		if w.IsFocused() {
			focused++
		}
	}
	if focused != 1 {
		t.Errorf("expected exactly 1 focused widget after navigation, got %d (INV-PROMPT-MVP-007)", focused)
	}
	if fr.Current().ID() != "b" {
		t.Errorf("Current().ID() = %q, want \"b\"", fr.Current().ID())
	}
}

// ─── Граничные случаи: пустое кольцо и один виджет ───────────────────────────

func TestFocusRing_EmptyRing_NilNoPanic(t *testing.T) {
	fr := NewFocusRing()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("empty ring panicked: %v (fail-safe дух §7)", r)
		}
	}()
	if fr.Next() != nil || fr.Prev() != nil || fr.Current() != nil {
		t.Error("empty ring must return nil from Next/Prev/Current")
	}
	if fr.Len() != 0 || fr.Index() != 0 {
		t.Errorf("empty ring Len/Index = %d/%d, want 0/0", fr.Len(), fr.Index())
	}
}

func TestFocusRing_SingleWidget_StaysFocused(t *testing.T) {
	ws := widgets("only")
	fr := NewFocusRing(toFocusable(ws)...)

	if fr.Next().ID() != "only" {
		t.Error("Next() on single-widget ring must stay on the same widget")
	}
	if !ws[0].IsFocused() {
		t.Error("single widget must remain focused")
	}
}

// ─── Видимый фокус: маркер + рамка (§18.1, INV-PROMPT-MVP-007) ─────────────

func TestRenderFocus_Focused_HasVisibleMarker(t *testing.T) {
	// Проверяем на ANSI256/TrueColor: маркер — border-символы Thick/Double.
	for _, profile := range []termenv.Profile{termenv.TrueColor, termenv.ANSI256} {
		got := RenderFocus("widget body", true, ui.NewTheme(), profile)
		if !HasFocusMarker(got) {
			t.Errorf("focused render (profile %v) has no visible focus marker (INV-PROMPT-MVP-007):\n%q", profile, got)
		}
	}
}

func TestRenderFocus_Unfocused_NoFocusMarker(t *testing.T) {
	got := RenderFocus("widget body", false, ui.NewTheme(), termenv.TrueColor)
	if HasFocusMarker(got) {
		t.Errorf("unfocused render must not carry a focus marker:\n%q", got)
	}
}

func TestRenderFocus_NilTheme_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RenderFocus with nil theme panicked: %v", r)
		}
	}()
	RenderFocus("body", true, nil, termenv.ANSI256)
}

func TestRenderFocusTitle_Focused_AddsMarker(t *testing.T) {
	if got := RenderFocusTitle("Today", true); !HasFocusMarker(got) {
		t.Errorf("focused title %q must contain ▸ marker (INV-PROMPT-MVP-007)", got)
	}
	if got := RenderFocusTitle("Today", false); HasFocusMarker(got) {
		t.Errorf("unfocused title %q must not contain focus marker", got)
	}
}

// toFocusable адаптирует []*testWidget к сигнатуре NewFocusRing(...Focusable).
func toFocusable(ws []*testWidget) []Focusable {
	out := make([]Focusable, len(ws))
	for i, w := range ws {
		out[i] = w
	}
	return out
}
