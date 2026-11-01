// main_cmds_test.go — тесты CLI-команд: runInit, runHookEvent (fail-safe
// exit-коды), runDoctor, runAdaptersList, детекция (§2.1, §2.2, §2.5, §2.10).
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/adapters"
	"github.com/Reider85/OryaCodeAuthorityFirst/internal/caf/storage"
)

// tempHome переводит HOME/USERPROFILE во временную директорию: установка
// адаптеров пишет конфиг агента — реальный домашний каталог не трогаем.
func tempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	return home
}

func TestRunInitCreatesCAFStructure(t *testing.T) {
	repo := t.TempDir()

	if err := runInit(repo, false, ""); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	if _, err := os.Stat(storage.GetCAFDir(repo)); err != nil {
		t.Errorf(".caf/ directory missing after init: %v", err)
	}
	if _, err := os.Stat(storage.GetManifestPath(repo)); err != nil {
		t.Errorf("manifest missing after init: %v", err)
	}
}

func TestRunInitWithAutoDetectAndAdapters(t *testing.T) {
	tempHome(t)
	adapters.Initialize()
	repo := t.TempDir()

	// auto-detect + установка opencode + игнор неизвестного ID (warn, не фатал).
	if err := runInit(repo, true, "opencode,does-not-exist"); err != nil {
		t.Fatalf("runInit with adapters: %v", err)
	}
	if known := knownAdapterIDs(); known == "" {
		t.Error("knownAdapterIDs must list registered adapters after Initialize")
	}
}

func TestInstallAdapterUnknownIsWarning(t *testing.T) {
	tempHome(t)
	adapters.Initialize()
	// Неизвестный ID — только предупреждение, без паники/ошибки.
	installAdapter("does-not-exist")
}

func TestRunHookEventExitCodes(t *testing.T) {
	adapters.Initialize()
	repo := t.TempDir()
	if err := storage.InitializeCAF(repo); err != nil {
		t.Fatal(err)
	}

	t.Run("unknown event is usage error (exit 1)", func(t *testing.T) {
		if got := runHookEvent("no-such-event", repo, "s1", nil, "", false); got != 1 {
			t.Errorf("exit code = %d, want 1", got)
		}
	})

	t.Run("known events are fail-safe exit 0 (INV-ARCH-004)", func(t *testing.T) {
		events := []struct {
			event   string
			files   []string
			message string
		}{
			{"pre-session", nil, ""},
			{"post-edit", []string{"README.md"}, "update readme"},
			{"pre-commit", nil, "feat: add limiter\n\nTraces to: FR-ADP-003"},
		}
		for _, ev := range events {
			if got := runHookEvent(ev.event, repo, "s1", ev.files, ev.message, false); got != 0 {
				t.Errorf("%s exit code = %d, want 0 (hooks never block)", ev.event, got)
			}
		}
	})
}

func TestReadCommitEditMsg(t *testing.T) {
	repo := t.TempDir()
	if got := readCommitEditMsg(repo); got != "" {
		t.Errorf("missing COMMIT_EDITMSG must yield empty string, got %q", got)
	}

	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".git", "COMMIT_EDITMSG"), []byte("feat: from git\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readCommitEditMsg(repo); got != "feat: from git\n" {
		t.Errorf("readCommitEditMsg = %q", got)
	}
}

func TestRunDoctorOnBareRepo(t *testing.T) {
	// Доктор на неинициализированном репо: честный отчёт без паники (§2.10).
	if err := runDoctor(t.TempDir()); err != nil {
		t.Errorf("runDoctor must report, not fail: %v", err)
	}
}

func TestRunAdaptersListAndDetection(t *testing.T) {
	adapters.Initialize()

	if err := runAdaptersList(); err != nil {
		t.Errorf("runAdaptersList: %v", err)
	}

	// Детекция зависит от машины (opencode может стоять/нет) —
	// контракт: не паникует и печатает таблицу (§2.5, <5s бюджет).
	detected := reportDetection()
	for _, id := range detected {
		switch id {
		case "opencode", "claude-code", "universal-lsp":
		default:
			t.Errorf("detected unknown adapter %q", id)
		}
	}
	_ = runAutoDetect()
}
