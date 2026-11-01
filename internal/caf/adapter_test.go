// adapter_test.go — тесты корневого пакета: реестр адаптеров и валидация
// capability matrix (FR-ADP-001, ADR-009, INV-ARCH-002).
package caf

import "testing"

// stubAdapter — минимальный реализатор Adapter для тестов реестра.
type stubAdapter struct {
	id       string
	detected bool
}

func (s stubAdapter) ID() string             { return s.id }
func (s stubAdapter) Detect() bool           { return s.detected }
func (s stubAdapter) Install() error         { return nil }
func (s stubAdapter) Uninstall() error       { return nil }
func (s stubAdapter) Capabilities() CapabilityMatrix {
	return CapabilityMatrix{
		PreSession: TierNative,
		PostEdit:   TierNative,
		PreCommit:  TierUniversal,
		MCP:        true,
		Tier:       1,
	}
}

// langStub реализует LanguageAdapter (языковые адаптеры регистрируются
// в тот же реестр, но отсеиваются типовым утверждением).
type langStub struct{ id string }

func (l langStub) ID() string                  { return l.id }
func (l langStub) Detect() bool                { return true }
func (l langStub) Install() error              { return nil }
func (l langStub) Uninstall() error            { return nil }
func (l langStub) Capabilities() CapabilityMatrix {
	return CapabilityMatrix{
		PreSession: TierNone,
		PostEdit:   TierNone,
		PreCommit:  TierNone,
		MCP:        false,
		Tier:       4,
	}
}
func (l langStub) ScanImports() []Import       { return []Import{{Path: "fmt", Line: 3}} }
func (l langStub) BuildSymbolGraph() SymbolGraph { return SymbolGraph{Symbols: []Symbol{{Name: "main", Kind: "func"}}} }
func (l langStub) ResolveExternalRepo(pkg string) RepoRef {
	return RepoRef{URL: pkg, Resolved: false}
}

func registerCleanup(tb testing.TB, ids ...string) {
	tb.Helper()
	for _, id := range ids {
		id := id
		tb.Cleanup(func() { delete(Registry, id) })
	}
}

func TestRegisterGetList(t *testing.T) {
	registerCleanup(t, "stub-a")

	RegisterAdapter(stubAdapter{id: "stub-a"})

	got, ok := GetAdapter("stub-a")
	if !ok {
		t.Fatal("GetAdapter after RegisterAdapter must find the adapter")
	}
	if got.ID() != "stub-a" {
		t.Errorf("ID = %q, want stub-a", got.ID())
	}

	found := false
	for _, id := range ListAdapters() {
		if id == "stub-a" {
			found = true
		}
	}
	if !found {
		t.Error("ListAdapters must include registered adapter")
	}
}

func TestGetAdapterMissing(t *testing.T) {
	if got, ok := GetAdapter("definitely-not-registered"); ok || got != nil {
		t.Errorf("GetAdapter(missing) = (%v, %v), want (nil, false)", got, ok)
	}
}

func TestRegisterAdapterOverwrites(t *testing.T) {
	registerCleanup(t, "stub-dup")

	RegisterAdapter(stubAdapter{id: "stub-dup", detected: false})
	RegisterAdapter(stubAdapter{id: "stub-dup", detected: true})

	got, ok := GetAdapter("stub-dup")
	if !ok {
		t.Fatal("adapter must stay registered after re-register")
	}
	if !got.Detect() {
		t.Error("re-registration must overwrite the previous adapter (map semantics)")
	}
}

func TestGetLanguageAdapter(t *testing.T) {
	registerCleanup(t, "stub-lang", "stub-event")

	RegisterAdapter(langStub{id: "stub-lang"})
	RegisterAdapter(stubAdapter{id: "stub-event"})

	lang, ok := GetLanguageAdapter("stub-lang")
	if !ok {
		t.Fatal("langStub must satisfy LanguageAdapter")
	}
	if symbols := lang.BuildSymbolGraph(); len(symbols.Symbols) != 1 {
		t.Errorf("BuildSymbolGraph symbols = %d, want 1", len(symbols.Symbols))
	}

	if _, ok := GetLanguageAdapter("stub-event"); ok {
		t.Error("plain Adapter must NOT satisfy LanguageAdapter")
	}
	if _, ok := GetLanguageAdapter("missing"); ok {
		t.Error("missing adapter must not satisfy LanguageAdapter")
	}
}

func TestValidateCapabilityMatrix(t *testing.T) {
	tests := []struct {
		name    string
		matrix  CapabilityMatrix
		wantErr bool
	}{
		{"valid T1 native", CapabilityMatrix{PreSession: TierNative, PostEdit: TierNative, PreCommit: TierNative, MCP: true, Tier: 1}, false},
		{"valid T4 fallback", CapabilityMatrix{PreSession: TierNone, PostEdit: TierWatcher, PreCommit: TierUniversal, MCP: false, Tier: 4}, false},
		{"valid tier wrapper", CapabilityMatrix{PreSession: TierWrapper, PostEdit: TierWrapper, PreCommit: TierWrapper, MCP: false, Tier: 2}, false},
		{"invalid pre-session tier", CapabilityMatrix{PreSession: "fast", PostEdit: TierNative, PreCommit: TierNative, Tier: 1}, true},
		{"invalid post-edit tier", CapabilityMatrix{PreSession: TierNative, PostEdit: "instant", PreCommit: TierNative, Tier: 1}, true},
		{"invalid pre-commit tier", CapabilityMatrix{PreSession: TierNative, PostEdit: TierNative, PreCommit: "", Tier: 1}, true},
		{"tier number below range", CapabilityMatrix{PreSession: TierNative, PostEdit: TierNative, PreCommit: TierNative, Tier: 0}, true},
		{"tier number above range", CapabilityMatrix{PreSession: TierNative, PostEdit: TierNative, PreCommit: TierNative, Tier: 5}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCapabilityMatrix(tc.matrix)
			if tc.wantErr && err == nil {
				t.Fatal("want error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("want nil error, got %v", err)
			}
		})
	}
}
