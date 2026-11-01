// keys_test.go — тесты универсальной клавиатурной карты (UI.md §0.6, INV-UI-009).
package ui

import (
	"strings"
	"testing"
)

func TestKeyClassification(t *testing.T) {
	k := NewKeys()

	nav := []string{k.Tab, k.ShiftTab, k.Up, k.Down, k.Left, k.Right, k.PageUp, k.PageDown, k.Home, k.End}
	for _, key := range nav {
		if !k.IsNavigationKey(key) {
			t.Errorf("%q must be a navigation key", key)
		}
	}

	actions := []string{k.Enter, k.Space, k.Refresh, k.Quit, k.Help, k.Back}
	for _, key := range actions {
		if !k.IsActionKey(key) {
			t.Errorf("%q must be an action key", key)
		}
	}

	globals := []string{k.Dashboard, k.Doctor, k.Metrics, k.Cost, k.Goto1, k.Goto2, k.Goto3, k.Goto4, k.Goto5, k.Goto6, k.Goto7, k.Goto8, k.Goto9}
	for _, key := range globals {
		if !k.IsGlobalShortcut(key) {
			t.Errorf("%q must be a global shortcut", key)
		}
	}

	if k.IsNavigationKey("x") || k.IsActionKey("x") || k.IsGlobalShortcut("x") {
		t.Error("unknown key must not classify as navigation/action/global")
	}
}

func TestParseKeyAndHandleKeyPress(t *testing.T) {
	if got := ParseKey("r"); got.Key != "r" {
		t.Errorf("ParseKey = %+v, want Key=r", got)
	}
	// HandleKeyPress — осознанный no-op: диспетчеризация живёт в экранах (§18).
	if cmd := HandleKeyPress("r", nil); cmd != nil {
		t.Error("HandleKeyPress must return nil (no-op)")
	}
}

func TestKeyConflictErrorMessage(t *testing.T) {
	err := &KeyConflictError{Key: "1", Current: "dashboard", Requested: "screen1"}
	got := err.Error()
	for _, want := range []string{"Key conflict", "'1'", "dashboard", "screen1"} {
		if !strings.Contains(got, want) {
			t.Errorf("conflict message missing %q: %s", want, got)
		}
	}
}

func TestValidateKeyMapDefaultIsConflictFree(t *testing.T) {
	// Дефолтная карта: все глобальные шорткаты уникальны после дедупликации
	// карты (см. реализацию — globalShortcuts это map, конфликт невозможен).
	if err := ValidateKeyMap(NewKeys()); err != nil {
		t.Errorf("default key map must validate, got %v", err)
	}
}

func TestGetHelpTextCoversUniversalMap(t *testing.T) {
	help := GetHelpText(NewKeys())
	for _, want := range []string{"Navigation:", "Actions:", "Shortcuts:", "Search:", "Tab/Shift+Tab", "Refresh", "Quit", "Help", "1-9"} {
		if !strings.Contains(help, want) {
			t.Errorf("help text missing %q (§18.4: help shows ALL keybindings)", want)
		}
	}
}
