// header_test.go — тесты «священной» шапки (§15.2 rule 8, §15.4).
package components

import (
	"strings"
	"testing"

	"github.com/Reider85/OryaCodeAuthorityFirst/pkg/ui"
)

func TestHeaderContainsVersionAndScreen(t *testing.T) {
	got := Header("1.2.3", "dashboard")
	if !strings.Contains(got, "CodeAuthorityFirst 1.2.3") {
		t.Errorf("header must contain app name and version, got %q", got)
	}
	if !strings.Contains(got, "dashboard") {
		t.Errorf("header must contain current screen, got %q", got)
	}
}

func TestHeaderWithFocusIndicatesActiveScreen(t *testing.T) {
	active := HeaderWithFocus("1.2.3", "doctor", true)
	inactive := HeaderWithFocus("1.2.3", "doctor", false)

	for _, v := range []string{active, inactive} {
		if !strings.Contains(v, "CodeAuthorityFirst") || !strings.Contains(v, "doctor") {
			t.Errorf("both variants must keep header content, got %q", v)
		}
	}
	// Активный экран получает focus-рамку (INV-PROMPT-MVP-007).
	if !ui.HasFocusMarker(active) {
		t.Errorf("active header must carry a focus marker, got %q", active)
	}
	if ui.HasFocusMarker(inactive) {
		t.Errorf("inactive header must not carry a focus marker, got %q", inactive)
	}
}
