// golden_test.go — инфраструктура golden-снапшотов 5 состояний (PROMPT-MVP §10).
//
// Спецификация §10 требует teatest (github.com/charmbracelet/teatest), но модуль
// удалён автором: 404 на GitHub, proxy.golang.org и pkg.go.dev (проверено
// 2026-11). Вместо него — эквивалент на stdlib: стабильные снапшоты
// testdata/*.golden с флагом -update. Это закрывает CI-пункт §21 item 1
// («5 snapshots exist per screen», INV-PROMPT-MVP-006): отсутствующий
// golden-файл валит тест.
package screens

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var update = flag.Bool("update", false, "перезаписать golden-файлы текущим выводом View()")

func TestMain(m *testing.M) {
	// Фиксируем цветовой профиль: без этого вывод зависит от TTY/платформы
	// и golden-файлы расходятся между локальной машиной и CI (§10).
	lipgloss.SetColorProfile(termenv.ANSI256)
	lipgloss.SetHasDarkBackground(true)
	os.Exit(m.Run())
}

// assertGolden сверяет got с testdata/<name>.golden.
// Обновление: go test ./pkg/ui/screens/... -update
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	got = normalizeNewlines(got)
	path := filepath.Join("testdata", name+".golden")

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("create testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %s missing (%v) — run: go test ./pkg/ui/screens/... -update", path, err)
	}
	if normalizeNewlines(string(want)) != got {
		t.Errorf("golden mismatch: %s\n--- want ---\n%s\n--- got ---\n%s", path, normalizeNewlines(string(want)), got)
	}
}

// normalizeNewlines убирает CRLF: View() всегда использует "\n", но git/Windows
// может чекаутить golden с CRLF — сравнение должно быть платформо-независимым.
func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
