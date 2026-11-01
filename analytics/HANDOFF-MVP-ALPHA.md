---
doc: HANDOFF-MVP-ALPHA
layer: handoff
version: 0.4.1
language: english
status: stable
audience: [LLM agent starting the ALPHA phase, senior developer pairing with LLM]
related: [PROMPT-MVP.md, PROMPT-ALPHA.md, ROADMAP.md, ARCHITECT.md, README.md]
phase_from: MVP (shipped as tag v0.4.1-mvp.1)
phase_to: ALPHA (not started)
---

# HANDOFF: MVP → ALPHA (PROMPT-MVP §14 step 19)

> **Purpose.** This document transfers context from the finished MVP phase to the ALPHA agent, as required by PROMPT-MVP.md §14 step 19 ("Hand off to ALPHA (PROMPT-ALPHA.md)"). It records the verified pre-conditions from PROMPT-ALPHA.md §1, the honest status of MVP exit criteria, and the known gaps ALPHA inherits.

## 1. What shipped

MVP is released as tag **`v0.4.1-mvp.1`** (commit `2502d79`, on `main`). All 18 steps of PROMPT-MVP §14 are complete:

| Steps | Deliverable |
|-------|-------------|
| 1-5 | Docs read, Go module, directory layout, cobra CLI, YAML session storage, `context.md` generator |
| 6-8 | Universal pre-commit hook (fail-safe), Tier-1 opencode + Claude Code adapters, Tier-4 universal LSP fallback |
| 9 | `caf doctor` — 5 basic health checks |
| 10-13 | 5-state machine, a11y helpers, state components, 3 TUI screens (onboarding, dashboard, adapters) |
| 14-16 | Full test suite (15 golden snapshots, focus/alt-text/high-contrast tests), §6 perf budgets verified, §21 checklist closed |
| 17-18 | README (quick start, install, 5-minute tour), LICENSE (Apache-2.0), CONTRIBUTING, version release |

The product surface at handoff: `caf init [--auto-detect]`, `caf who`, `caf context`, `caf hook {pre-session,post-edit,pre-commit}`, `caf dashboard`, `caf adapters list`, `caf doctor`, `caf --version`.

## 2. PROMPT-ALPHA §1 pre-conditions — verified at handoff

| # | Pre-condition (PROMPT-ALPHA §1) | Status | Evidence |
|---|--------------------------------|--------|----------|
| 1 | MVP shipped; tag `v0.4.1-mvp.1` exists; MVP exit criteria pass | ✅ code-verified / ⚠️ field items pending | `git describe --tags` → `v0.4.1-mvp.1`; `go vet`, `go test ./...` (14 pkgs, all ok), `go build` pass; see §3 for field items |
| 2 | MVP codebase accessible (adapter pattern, storage, hooks, `pkg/ui/states.go`, `pkg/ui/a11y.go`, `pkg/ui/components/{skeleton,errorbox,emptycta,focusring}.go`) | ✅ | all paths exist in this repo at the tag |
| 3 | Security design document exists (threat model + sandbox spec) | ✅ | ARCHITECT.md §8.1 (threat model), §8.3 (sandboxed YAML parser spec) — design only, implementation is ALPHA step 9 |
| 4 | Documents current: ARD.md, ARCHITECT.md, ROADMAP.md §3, UI.md §5/§11/§15/§16, PROMPT-MVP.md §15-§22, ADR.md ADR-004/009/014 | ✅ | all at v0.4.x-triz2 / v0.4.1-ui-strong |
| 5 | MVP UI invariants enforceable in CI (5-state snapshots, focus-visibility, alt-text, transition budget, high-contrast) | ✅ **closed by this step** | `.github/workflows/ci.yml` (added in step 19): vet, tests with coverage ≥80% floor, UI benchmarks, binary smoke test |

All four UI exit criteria (PROMPT-MVP §12 items 9-12) pass as Go tests: `TestFocusVisibility`, `TestAltTextPresence`, `TestHighContrastRender` (pkg/ui/screens/a11y_screens_test.go), `TestBudgetTransition` (≤100ms), plus 15 golden snapshots (3 screens × 5 states).

**Per-package coverage at handoff (floor 80%, NFR-MAINT-01):** cmd/caf 88.7%, internal/caf 100%, adapters 100/89.3/85.7/86.7%, config 96.2%, doctor 92.7%, hooks 93.5%, lsp 80.3%, storage 94.6%, pkg/ui 92.5%, components 100%, screens 88.7%.

## 3. MVP exit criteria — honest status (PROMPT-MVP §12)

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Pilot repo uses `caf` 7 days straight | ⚠️ **field item — human must verify** |
| 2 | Universal pre-commit hook fires on 3+ different agents | ⚠️ **field item — human must verify** |
| 3 | LSP fallback builds symbol graph for Go repo up to 1000 files | ⚠️ **field item — human must verify** (code + tests exist) |
| 4 | TUI dashboard first paint <100ms | ✅ `BenchmarkFirstPaint`, `TestBudgetFirstPaint` |
| 5 | Auto-detect finds opencode + Claude Code in <5s | ✅ (tests in cmd/caf) |
| 6 | Adapter contract suite passes (12 cases, 4 adapters) | ✅ contract tests green |
| 7 | `caf doctor` runs without errors; 5 checks pass | ✅ runs clean (fresh repo: 1 passed / 2 warnings / 2 unknown — expected before `caf init`) |
| 8 | Security design document reviewed and approved | ✅ ARCHITECT.md §8 (human approved via review) |
| 9-12 | UI 5-state audit, §21 checklist, focus-visibility, high-contrast | ✅ all as automated tests (see §2) |

The human decided to start ALPHA with items 1-3 still field-pending (INV-RM-002 is medium criticality; the ALPHA agent must not treat them as done).

## 4. Known gaps ALPHA inherits

1. **`.caf/` and `analytics/.idea/` are untracked local artifacts.** `.gitignore` covers binaries and test artifacts only. ALPHA step 9 wraps `internal/caf/storage/yaml.go` with the sandbox — decide then whether `.caf/` should stay untracked by convention (it is per-repo runtime data).
2. **`caf when --pattern` is a deliberate stub** (deferred to ALPHA, FR-PROV-004). Do not "fix" it inside MVP scope — it is ALPHA work.
3. **Field exit criteria §3 items 1-3 are unverified.** If the human reports a hook or LSP failure during ALPHA development, treat it as an MVP bug fix, not ALPHA scope creep.
4. **CI runs on ubuntu only.** Several tests branch on `runtime.GOOS == "windows"` (skips are intentional). Windows parity is not covered by CI.
5. **`caf doctor` knows only 5 checks.** The 11-check v2 (with 4-state semantics and self-healing) is ALPHA step 16 — do not extend the MVP doctor in place without the ALPHA spec.

## 5. How to start ALPHA

Do not improvise. The ALPHA agent's entry point is **PROMPT-ALPHA.md §15 ("First task")**: verify this handoff (§2 above), read ARD §4.2/§4.8/§4.9, ARCHITECT §8/§9, ROADMAP §3, UI §5/§11/§15/§16, then implement security-first (sandbox → storage → triviality → pre-commit → adapters → doctor v2 → migrate → UI screens). Invariants INV-PROMPT-ALPHA-001..010 apply from the first commit; MVP UI invariants (INV-PROMPT-MVP-006..010) remain in force and are CI-enforced.

Verification commands used at handoff (re-run them before the first ALPHA PR):

```bash
go vet ./...
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out | tail -1   # ≥80% total
go test ./pkg/ui/... -run=^$ -bench=. -benchtime=10x
go build -o caf ./cmd/caf && ./caf --version # expect 0.4.1-mvp.1
```
