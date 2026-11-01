// config_test.go — тесты .caf/config.yml: дефолты, round-trip, битый YAML,
// пути, песочница SafeParseYAML (ARCHITECT.md §7, ADR-005).
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigReturnsDefaultsWhenFileMissing(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), ".caf", "config.yml"))

	cfg, err := m.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Version != "0.4.0-triz2" {
		t.Errorf("Version = %q, want 0.4.0-triz2", cfg.Version)
	}
	if cfg.TrivialityThreshold != 50 {
		t.Errorf("TrivialityThreshold = %d, want 50", cfg.TrivialityThreshold)
	}
	if cfg.UI.Theme != "dark" || cfg.UI.Language != "ru" {
		t.Errorf("UI defaults = %+v, want theme=dark language=ru", cfg.UI)
	}
	if !cfg.LSP.Strict {
		t.Error("LSP.Strict must default to true")
	}
	if cfg.Telemetry.Enabled {
		t.Error("Telemetry must be opt-in (SEC-005)")
	}
	if cfg.Cost.LLMTokenCapDaily != 100000 || cfg.Cost.AlertThresholdPct != 80 {
		t.Errorf("Cost defaults = %+v", cfg.Cost)
	}
	if cfg.Decisions.Autocomplete != "aggressive" {
		t.Errorf("Decisions.Autocomplete = %q", cfg.Decisions.Autocomplete)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".caf", "config.yml")
	// Контракт WriteYAML: родительский каталог создаёт вызывающий
	// (`caf init` создаёт .caf/ до записи конфига).
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	m := NewManager(path)

	want := Config{
		Version:             "0.4.1-mvp.1",
		TrivialityThreshold: 42,
		UI: UIConfig{
			Theme:               "high-contrast",
			Language:            "en",
			HeatmapRefreshSeconds: 60,
			ReviewMaxCards:      10,
		},
		LSP:       LSPConfig{Strict: false},
		Telemetry: TelemetryConfig{Enabled: true, Endpoint: "http://localhost:9090"},
		Cost:      CostConfig{LLMTokenCapDaily: 12345, LLMTokenCapMonthly: 54321, AlertThresholdPct: 90},
	}
	if err := m.SaveConfig(want); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	got, err := m.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if got.Version != want.Version || got.TrivialityThreshold != want.TrivialityThreshold {
		t.Errorf("top-level round-trip mismatch: got %+v", got)
	}
	if got.UI != want.UI {
		t.Errorf("UI round-trip: got %+v, want %+v", got.UI, want.UI)
	}
	if got.LSP != want.LSP || got.Telemetry != want.Telemetry || got.Cost != want.Cost {
		t.Errorf("section round-trip mismatch: LSP=%+v Telemetry=%+v Cost=%+v", got.LSP, got.Telemetry, got.Cost)
	}
}

func TestSaveConfigLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".caf", "config.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	m := NewManager(path)

	if err := m.SaveConfig(getDefaultConfig()); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("atomic write must not leave .tmp file behind")
	}
}

func TestLoadConfigCorruptYAMLFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: [unclosed\n  bad indent"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := NewManager(path)
	if _, err := m.LoadConfig(); err == nil {
		t.Fatal("corrupt YAML must return an error")
	} else if !strings.Contains(err.Error(), "failed to load config") {
		t.Errorf("error must be wrapped, got: %v", err)
	}
}

func TestGetConfigPath(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, ".caf", "config.yml")
	if got := GetConfigPath(root); got != want {
		t.Errorf("GetConfigPath = %q, want %q", got, want)
	}
}

func TestReadYAMLMissingFile(t *testing.T) {
	err := ReadYAML(filepath.Join(t.TempDir(), "absent.yml"), &Config{})
	if err == nil {
		t.Fatal("missing file must return an error")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("want wrapped read error, got: %v", err)
	}
}

func TestSafeParseYAMLRejectsNilOutput(t *testing.T) {
	if err := SafeParseYAML([]byte("version: x"), nil); err == nil {
		t.Fatal("nil out must be rejected")
	}
}

func TestSafeParseYAMLRejectsCustomTags(t *testing.T) {
	// YAML-тег «!» может триггерить выполнение кода в некоторых парсерах —
	// SafeParseYAML их отвергает (FR-SEC-001, design-level в MVP).
	err := SafeParseYAML([]byte("!evil payload\nversion: 1\n"), &Config{})
	if err == nil {
		t.Fatal("custom YAML tags must be rejected")
	}
	if !strings.Contains(err.Error(), "custom YAML tags") {
		t.Errorf("want custom-tag error, got: %v", err)
	}
}

func TestWriteYAMLMissingDirFails(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir", "config.yml")
	if err := WriteYAML(missing, getDefaultConfig()); err == nil {
		t.Fatal("writing into missing directory must fail")
	}
}
