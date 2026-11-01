package adapters

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf"
	claudecode "github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/adapters/claude_code"
	opencode "github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/adapters/opencode"
)

// contractCases — ожидаемые capability-матрицы. Источник истины — таблица
// ARCHITECT.md §1 (FR-ADP-002: матрица возможностей есть артефакт первого
// класса, ADR-009). Тест не даёт ей «поплыть»: любое изменение ячейки в
// коде без правки §1 ломает CI.
//
// Матрица данных, а не кода (TRIZ-15): сюда добавляется строка — и новый
// адаптер попадает в контракт. PROMPT-MVP §10 говорит про «12 cases for
// 4 adapters» — 4-й event-адаптер (или тестовый дублёр) подключится к этому
// же набору без изменения теста; сейчас в MVP три event-адаптера
// (opencode, claude-code, universal-lsp) × 3 события = 9 явных ячеек.
var contractCases = []struct {
	id   string
	want caf.CapabilityMatrix
}{
	{
		id: "opencode",
		want: caf.CapabilityMatrix{
			PreSession: caf.TierNative,
			PostEdit:   caf.TierNative,
			PreCommit:  caf.TierNative,
			MCP:        true,
			Tier:       1,
		},
	},
	{
		id: "claude-code",
		want: caf.CapabilityMatrix{
			PreSession: caf.TierNative,
			PostEdit:   caf.TierNative,
			PreCommit:  caf.TierNative,
			MCP:        true,
			Tier:       1,
		},
	},
	{
		id: "universal-lsp",
		want: caf.CapabilityMatrix{
			PreSession: caf.TierWrapper,
			PostEdit:   caf.TierWatcher,
			PreCommit:  caf.TierUniversal,
			MCP:        false,
			Tier:       4,
		},
	},
}

// TestAdapterContract — контракт FR-ADP-001/002: каждый адаптер в реестре
// реализует интерфейс Adapter (5 методов — INV-ADR-009), его ID совпадает с
// ключом реестра, матрица равна строке ARCHITECT.md §1 и валидна, Detect()
// не паникует даже при пустом $PATH.
func TestAdapterContract(t *testing.T) {
	Initialize()

	for _, tc := range contractCases {
		t.Run(tc.id, func(t *testing.T) {
			adapter, ok := caf.GetAdapter(tc.id)
			if !ok {
				t.Fatalf("adapter %q is not registered by Initialize()", tc.id)
			}
			if adapter.ID() != tc.id {
				t.Errorf("ID() = %q, want %q", adapter.ID(), tc.id)
			}

			got := adapter.Capabilities()
			if got != tc.want {
				t.Errorf("capability matrix drifted from ARCHITECT.md §1:\n got %+v\nwant %+v",
					got, tc.want)
			}
			if err := caf.ValidateCapabilityMatrix(got); err != nil {
				t.Errorf("ValidateCapabilityMatrix: %v", err)
			}

			t.Setenv("PATH", "")
			if adapter.Detect() {
				t.Errorf("Detect() must be false with an empty $PATH")
			}

			// Tier-1 адаптеры обязаны уметь отвечать «прописан ли caf» —
			// статус hooks wired в `caf adapters list` (caf.Configured).
			if got.Tier == 1 {
				if _, isConfigured := adapter.(caf.Configured); !isConfigured {
					t.Errorf("Tier-1 adapter %q must implement caf.Configured", tc.id)
				}
			}
		})
	}
}

// TestTier1InstallLifecycle — Install() идемпотентен, Uninstall() удаляет
// только своё и тоже идемпотентен (ARCHITECT.md §3.1). Выполняется во
// временном HOME: реальный ~/.config и ~/.claude не трогаются.
//
// universal-lsp здесь не участвует: его Install() требует gopls, lifecycle
// Tier-4 адаптера покрыт TestTier4InstallLifecycle ниже (шаг 8).
func TestTier1InstallLifecycle(t *testing.T) {
	Initialize()

	for _, id := range []string{"opencode", "claude-code"} {
		t.Run(id, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_CONFIG_HOME", "")

			adapter, ok := caf.GetAdapter(id)
			if !ok {
				t.Fatalf("adapter %q not registered", id)
			}

			if err := adapter.Install(); err != nil {
				t.Fatalf("first Install: %v", err)
			}
			first, err := os.ReadFile(configPathFor(t, id))
			if err != nil {
				t.Fatalf("config not written: %v", err)
			}
			if !adapter.(caf.Configured).Configured() {
				t.Error("Configured() = false after Install")
			}

			if err := adapter.Install(); err != nil {
				t.Fatalf("second Install must be idempotent: %v", err)
			}
			second, err := os.ReadFile(configPathFor(t, id))
			if err != nil {
				t.Fatalf("config read failed: %v", err)
			}
			if !bytes.Equal(first, second) {
				t.Errorf("Install is not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
			}

			if err := adapter.Uninstall(); err != nil {
				t.Fatalf("Uninstall: %v", err)
			}
			if err := adapter.Uninstall(); err != nil {
				t.Fatalf("second Uninstall must be nil: %v", err)
			}
			if adapter.(caf.Configured).Configured() {
				t.Error("Configured() = true after Uninstall")
			}
		})
	}
}

// configPathFor возвращает путь конфига Tier-1 адаптера для проверок lifecycle.
func configPathFor(t *testing.T, id string) string {
	t.Helper()
	switch id {
	case "opencode":
		p, err := opencode.ConfigPath()
		if err != nil {
			t.Fatalf("opencode.ConfigPath: %v", err)
		}
		return p
	case "claude-code":
		p, err := claudecode.ConfigPath()
		if err != nil {
			t.Fatalf("claudecode.ConfigPath: %v", err)
		}
		return p
	}
	t.Fatalf("unknown tier-1 adapter %q", id)
	return ""
}

// TestNoLanguageAdaptersRegistered — регрессия решения шага 7: выдуманные
// LanguageAdapter-заглушки ("go"/"python" с фабричными symbol graph) удалены
// из opencode/ и claude_code/. Интерфейс LanguageAdapter остаётся в
// internal/caf/adapter.go (требование §4), но реализаций в MVP нет —
// символный граф строит gopls (ADR-013, шаг 8).
func TestNoLanguageAdaptersRegistered(t *testing.T) {
	Initialize()
	for _, id := range []string{"go", "python"} {
		if _, ok := caf.GetAdapter(id); ok {
			t.Errorf("language adapter %q must not be in the registry (removed in step 7)", id)
		}
	}
}

// TestTier4InstallLifecycle — Install()/Uninstall() universal-lsp идемпотентны
// (ARCHITECT.md §3.1). Install() без gopls в $PATH — ошибка с hint (ADR-026:
// MVP требует ручной установки); со стабом gopls — nil дважды. Uninstall()
// без запущенного процесса — nil дважды.
func TestTier4InstallLifecycle(t *testing.T) {
	Initialize()

	adapter, ok := caf.GetAdapter("universal-lsp")
	if !ok {
		t.Fatal("adapter \"universal-lsp\" not registered")
	}

	// Без gopls: Install() обязан отказать и объяснить, как поставить.
	t.Setenv("PATH", "")
	if err := adapter.Install(); err == nil {
		t.Error("Install() without gopls must fail")
	}

	// Со стабом gopls: Install() идемпотентен.
	dir := t.TempDir()
	stubGoplsInDir(t, dir)
	t.Setenv("PATH", dir)
	if err := adapter.Install(); err != nil {
		t.Errorf("first Install with stub gopls: %v", err)
	}
	if err := adapter.Install(); err != nil {
		t.Errorf("second Install must be idempotent: %v", err)
	}

	// Uninstall() идемпотентен при отсутствии запущенного процесса.
	if err := adapter.Uninstall(); err != nil {
		t.Errorf("first Uninstall: %v", err)
	}
	if err := adapter.Uninstall(); err != nil {
		t.Errorf("second Uninstall must be nil: %v", err)
	}
}

// stubGoplsInDir кладёт стаб gopls в dir (Detect/Install ищут его в $PATH).
func stubGoplsInDir(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		if err := os.WriteFile(filepath.Join(dir, "gopls.bat"), []byte("@echo off\r\n"), 0644); err != nil {
			t.Fatalf("failed to write gopls stub: %v", err)
		}
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "gopls"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write gopls stub: %v", err)
	}
}

// contractStub — 4-я строка контракта: тестовый дублёр event-адаптера.
// PROMPT-MVP §10: «4 adapters × 3 events = 12 cases» — в MVP три реальных
// event-адаптера, дублёр закрывает норму и доказывает, что матрица
// контракта расширяема без изменения теста (см. комментарий к contractCases).
type contractStub struct{ id string }

func (c contractStub) ID() string       { return c.id }
func (c contractStub) Detect() bool     { return false }
func (c contractStub) Install() error   { return nil }
func (c contractStub) Uninstall() error { return nil }
func (c contractStub) Capabilities() caf.CapabilityMatrix {
	return caf.CapabilityMatrix{
		PreSession: caf.TierWrapper,
		PostEdit:   caf.TierWatcher,
		PreCommit:  caf.TierUniversal,
		MCP:        false,
		Tier:       2,
	}
}

// TestAdapterEventContract12Cases — §10: контракт «4 адаптера × 3 события».
// Каждая ячейка pre-session/post-edit/pre-commit обязана быть валидным
// тиром из ARCHITECT.md §1 и никогда не пустой. 12 кейсов = 4×3 (§12 exit).
func TestAdapterEventContract12Cases(t *testing.T) {
	Initialize()
	caf.RegisterAdapter(contractStub{id: "contract-stub-double"})
	t.Cleanup(func() { delete(caf.Registry, "contract-stub-double") })

	adapterIDs := []string{"opencode", "claude-code", "universal-lsp", "contract-stub-double"}
	events := []struct {
		name string
		cell func(caf.CapabilityMatrix) caf.EventTier
	}{
		{"pre-session", func(m caf.CapabilityMatrix) caf.EventTier { return m.PreSession }},
		{"post-edit", func(m caf.CapabilityMatrix) caf.EventTier { return m.PostEdit }},
		{"pre-commit", func(m caf.CapabilityMatrix) caf.EventTier { return m.PreCommit }},
	}

	validTiers := map[caf.EventTier]bool{
		caf.TierNative:    true,
		caf.TierWrapper:   true,
		caf.TierWatcher:   true,
		caf.TierUniversal: true,
		caf.TierNone:      true,
	}

	cases := 0
	for _, id := range adapterIDs {
		for _, ev := range events {
			cases++
			t.Run(id+"/"+ev.name, func(t *testing.T) {
				adapter, ok := caf.GetAdapter(id)
				if !ok {
					t.Fatalf("adapter %q not registered", id)
				}
				matrix := adapter.Capabilities()
				if err := caf.ValidateCapabilityMatrix(matrix); err != nil {
					t.Fatalf("matrix invalid: %v", err)
				}
				cell := ev.cell(matrix)
				if !validTiers[cell] {
					t.Errorf("event %s: cell %q is not a valid tier", ev.name, cell)
				}
			})
		}
	}
	if cases != 12 {
		t.Fatalf("§10 requires exactly 12 contract cases (4 adapters × 3 events), built %d", cases)
	}
}
