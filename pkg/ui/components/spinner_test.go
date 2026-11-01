// spinner_test.go — тесты спиннера асинхронных операций (§20.4, §19.4).
package components

import "testing"

func TestSpinnerLifecycle(t *testing.T) {
	s := NewSpinner()

	if s.IsAnimating() {
		t.Error("fresh spinner must not animate")
	}
	if got := s.Render(); got != "" {
		t.Errorf("Render() before Start = %q, want empty", got)
	}
	if got := s.NextFrame(); got != "" {
		t.Errorf("NextFrame() before Start = %q, want empty", got)
	}

	s.Start()
	if !s.IsAnimating() {
		t.Fatal("Start must begin animation")
	}
	if s.Current != 0 {
		t.Errorf("Start must reset frame index, got %d", s.Current)
	}
	first := s.Render()
	if first == "" {
		t.Fatal("Render() must return a frame while animating")
	}
	// Кадры цикличны: первый Render() уже сдвинул Current на 1 —
	// до возврата к кадру 0 нужно len(Frames)-1 шагов.
	for i := 1; i < len(s.Frames); i++ {
		s.NextFrame()
	}
	if got := s.Render(); got != first {
		t.Errorf("frames must cycle: after one round got %q, want %q", got, first)
	}

	s.Stop()
	if s.IsAnimating() {
		t.Error("Stop must end animation")
	}
	if got := s.Render(); got != "" {
		t.Errorf("Render() after Stop = %q, want empty", got)
	}

	s.Reset()
	if s.IsAnimating() || s.Current != 0 {
		t.Error("Reset must return to initial state")
	}

	// Пустой список кадров — безопасный путь без паники (a11y-режим).
	s.Frames = nil
	s.Start()
	if got := s.Render(); got != "" {
		t.Errorf("Render() with no frames = %q, want empty", got)
	}
}

func TestSpinnerSetAnimating(t *testing.T) {
	s := NewSpinner()
	s.SetAnimating(true)
	if !s.IsAnimating() || s.Current != 0 {
		t.Error("SetAnimating(true) must start animation from frame 0")
	}
	s.NextFrame()
	s.SetAnimating(true) // повторный вызов сбрасывает кадр
	if s.Current != 0 {
		t.Errorf("SetAnimating(true) must reset frame, got %d", s.Current)
	}
	s.SetAnimating(false)
	if s.IsAnimating() {
		t.Error("SetAnimating(false) must stop animation")
	}
}
