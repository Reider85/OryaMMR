// tui_test.go — тесты AppModel: глобальные клавиши, переключение экранов,
// помощь, refresh, view (§18.4 universal map, §18.5 conflict resolution).
package ui

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
)

func TestPlaceholderScreenContract(t *testing.T) {
	p := &PlaceholderScreen{ID: "doctor", Theme: NewTheme()}

	if cmd := p.Init(); cmd != nil {
		t.Error("placeholder Init must be nil")
	}
	screen, cmd := p.Update(nil)
	if screen != Screen(p) || cmd != nil {
		t.Errorf("Update must be identity, got (%v, %v)", screen, cmd)
	}
	view := p.View()
	if !strings.Contains(view, "doctor") || !strings.Contains(view, "Press ? for help") {
		t.Errorf("placeholder view missing content: %q", view)
	}
	if p.GetID() != "doctor" {
		t.Errorf("GetID = %q", p.GetID())
	}
	if !p.IsActive() {
		t.Error("placeholder must be active")
	}
	p.SetActive(false) // no-op, без паники
}

func TestScreenTypeString(t *testing.T) {
	tests := []struct {
		st   ScreenType
		want string
	}{
		{ScreenDashboard, "dashboard"},
		{ScreenDoctor, "doctor"},
		{ScreenSettings, "settings"},
		{ScreenType(99), "unknown"},
	}
	for _, tc := range tests {
		if got := tc.st.String(); got != tc.want {
			t.Errorf("ScreenType(%d).String() = %q, want %q", tc.st, got, tc.want)
		}
	}
}

func TestAppModelInit(t *testing.T) {
	m := NewAppModel(t.TempDir())
	cmd := m.Init() // PrecomputeStyles + placeholder dashboard

	if m.ActiveScreen == nil {
		t.Fatal("Init must set an active screen")
	}
	if m.ActiveScreen.GetID() != "dashboard" {
		t.Errorf("active screen = %q, want dashboard", m.ActiveScreen.GetID())
	}
	if m.CurrentScreen != ScreenDashboard {
		t.Errorf("CurrentScreen = %v", m.CurrentScreen)
	}
	if cmd != nil {
		t.Error("placeholder Init must not spawn commands")
	}
	if m.ScreenReader.GetLatestAnnouncement() == nil {
		t.Error("app start must be announced to screen reader (§19.3)")
	}
}

// keyMsg собирает реалистичное для bubbletea нажатие (KeyRunes + Runes).
func keyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestAppModelGlobalKeys(t *testing.T) {
	t.Run("ctrl+c and esc quit", func(t *testing.T) {
		for _, km := range []tea.KeyMsg{{Type: tea.KeyCtrlC}, {Type: tea.KeyEsc}} {
			m := NewAppModel(t.TempDir())
			_ = m.Init()
			_, cmd := m.Update(km)
			if !m.Quit {
				t.Errorf("%v must set Quit", km)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("%v must produce QuitMsg", km)
			}
		}
	})

	t.Run("? returns HelpMsg", func(t *testing.T) {
		m := NewAppModel(t.TempDir())
		_ = m.Init()
		_, cmd := m.Update(keyMsg('?'))
		if cmd == nil {
			t.Fatal("? must return help command")
		}
		if _, ok := cmd().(HelpMsg); !ok {
			t.Error("? must produce HelpMsg")
		}
	})

	t.Run("r returns RefreshScreenMsg", func(t *testing.T) {
		m := NewAppModel(t.TempDir())
		_ = m.Init()
		_, cmd := m.Update(keyMsg('r'))
		if cmd == nil {
			t.Fatal("r must return refresh command")
		}
		if _, ok := cmd().(RefreshScreenMsg); !ok {
			t.Error("r must produce RefreshScreenMsg")
		}
	})

	t.Run("1-3 switch screens via ScreenSwitchMsg (§18.5)", func(t *testing.T) {
		want := map[rune]ScreenType{'1': ScreenDashboard, '2': ScreenDoctor, '3': ScreenSettings}
		for key, screenType := range want {
			m := NewAppModel(t.TempDir())
			_ = m.Init()
			_, cmd := m.Update(keyMsg(key))
			if cmd == nil {
				t.Fatalf("%c must return switch command", key)
			}
			msg, ok := cmd().(ScreenSwitchMsg)
			if !ok || msg.ScreenType != screenType {
				t.Errorf("%c produced %#v, want ScreenSwitchMsg{%v}", key, msg, screenType)
			}
		}
	})
}

// countingScreen — экран-счётчик для проверки, что switchScreen/refresh
// действительно вызывают Init (PlaceholderScreen.Init возвращает nil и
// tea.Batch(nil) даёт nil — сигнал был бы неотличим от «не вызвано»).
type countingScreen struct {
	id     string
	inits  int
}

func (c *countingScreen) Init() tea.Cmd {
	c.inits++
	return func() tea.Msg { return struct{ Screen string }{c.id} }
}

func (c *countingScreen) Update(msg tea.Msg) (Screen, tea.Cmd) { return c, nil }
func (c *countingScreen) View() string                         { return "view-" + c.id }
func (c *countingScreen) GetID() string                        { return c.id }
func (c *countingScreen) IsActive() bool                       { return true }
func (c *countingScreen) SetActive(bool)                       {}

func TestAppModelScreenSwitchingAndRefresh(t *testing.T) {
	m := NewAppModel(t.TempDir())
	_ = m.Init()
	doc := &countingScreen{id: "doctor"}
	settings := &countingScreen{id: "settings"}
	m.SetScreen(ScreenDoctor, doc)
	m.SetScreen(ScreenSettings, settings)

	// Переключение на зарегистрированный экран: смена + Init нового экрана.
	_, cmd := m.Update(ScreenSwitchMsg{ScreenType: ScreenDoctor})
	if m.CurrentScreen != ScreenDoctor || m.ActiveScreen.GetID() != "doctor" {
		t.Errorf("switch failed: screen=%v active=%q", m.CurrentScreen, m.ActiveScreen.GetID())
	}
	if doc.inits != 1 {
		t.Errorf("screen switch must run Init once, ran %d times", doc.inits)
	}
	if cmd == nil {
		t.Error("Update must return the batched Init command")
	}

	// Незарегистрированный экран — игнорируется (MVP scope, §18.5).
	m.Update(ScreenSwitchMsg{ScreenType: ScreenType(42)})
	if m.CurrentScreen != ScreenDoctor {
		t.Error("unregistered screen must be ignored")
	}

	// Повторное сообщение того же типа — no-op.
	m.Update(ScreenSwitchMsg{ScreenType: ScreenDoctor})
	if doc.inits != 1 {
		t.Errorf("same-screen switch must be a no-op, Init ran %d times", doc.inits)
	}

	// Refresh — переинициализирует активный экран.
	_, refreshCmd := m.Update(RefreshScreenMsg{Force: true})
	if doc.inits != 2 {
		t.Errorf("RefreshScreenMsg must run Init, ran %d times", doc.inits)
	}
	if refreshCmd == nil {
		t.Error("Update must return the batched Init command")
	}

	// HelpMsg возвращает help-команду.
	_, helpCmd := m.Update(HelpMsg{ScreenID: "dashboard"})
	if _, ok := helpCmd().(HelpMsg); !ok {
		t.Error("HelpMsg must loop back to help command")
	}
}

func TestAppModelPerformanceWarningAnnounced(t *testing.T) {
	m := NewAppModel(t.TempDir())
	_ = m.Init()
	// §19.3: объявления уходят в stderr (вне потока TUI-рендера).
	m.A11y.Config.ScreenReader = true
	m.A11y.Config.AnnounceChanges = true

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	_, cmd := m.Update(PerformanceWarningMsg{Level: AnnouncementWarning, Message: "frame render slow"})
	w.Close()
	os.Stderr = oldStderr
	out, _ := io.ReadAll(r)

	if cmd != nil {
		t.Error("performance warning must not spawn commands")
	}
	if !strings.Contains(string(out), "frame render slow") {
		t.Errorf("warning must be announced to stderr, got %q", out)
	}
}

func TestAppModelView(t *testing.T) {
	m := NewAppModel(t.TempDir())
	_ = m.Init()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.Width != 100 || m.Height != 30 {
		t.Fatalf("WindowSizeMsg not applied: %dx%d", m.Width, m.Height)
	}

	view := m.View()
	if !strings.Contains(view, "Screen: dashboard") {
		t.Errorf("view must show status bar, got:\n%s", view)
	}
	if !strings.Contains(view, "Press ? for help") {
		t.Errorf("status bar must hint help key, got:\n%s", view)
	}
	if !strings.Contains(view, "dashboard screen") {
		t.Errorf("view must render active screen, got:\n%s", view)
	}

	// Quit — прощальный экран.
	m.Quit = true
	if got := m.View(); got != "Goodbye!" {
		t.Errorf("quit view = %q", got)
	}
}

func TestAppModelLargeTextModeRenders(t *testing.T) {
	m := NewAppModel(t.TempDir())
	_ = m.Init()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.A11y.Config.LargeText = true

	view := m.View()
	if view == "" {
		t.Fatal("large-text view must not be empty")
	}
	if !strings.Contains(view, "Screen: dashboard") {
		t.Error("large-text mode must keep content")
	}
}

func TestAppModelAccessors(t *testing.T) {
	m := NewAppModel(t.TempDir())
	_ = m.Init()
	doc := &PlaceholderScreen{ID: "doctor", Theme: m.Theme}
	m.SetScreen(ScreenDoctor, doc)

	if m.GetScreen(ScreenDoctor) != Screen(doc) {
		t.Error("GetScreen must return registered screen")
	}
	if m.GetScreen(ScreenSettings) != nil {
		t.Error("GetScreen for unregistered type must be nil")
	}
	if m.GetCurrentScreen() != m.ActiveScreen {
		t.Error("GetCurrentScreen must return ActiveScreen")
	}
	if m.GetScreenType() != m.CurrentScreen {
		t.Error("GetScreenType must return CurrentScreen")
	}

	// SetScreen текущего типа сразу заменяет активный экран.
	m.SetScreen(ScreenDashboard, &PlaceholderScreen{ID: "dashboard-2", Theme: m.Theme})
	if m.ActiveScreen.GetID() != "dashboard-2" {
		t.Errorf("SetScreen(current) must swap ActiveScreen, got %q", m.ActiveScreen.GetID())
	}
}
