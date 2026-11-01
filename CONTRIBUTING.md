# Contributing to CodeAuthorityFirst (caf)

Thanks for contributing! This guide encodes the project's conventions — they come from [PROMPT-MVP.md](analytics/PROMPT-MVP.md) and are enforced in review.

## Development setup

Requirements:

- **Go 1.22+** (build-time only; the binary has no runtime dependencies)
- **git** (only for the pre-commit hook)

```bash
git clone https://github.com/Reider85/OryaCodeAuthorityFirst.git
cd OryaCodeAuthorityFirst
go build -o caf ./cmd/caf     # Windows: go build -o caf.exe ./cmd/caf
```

No CGO, no Node.js, no Python — `go build` is the only step (INV-PROMPT-MVP-002).

## Quality gates (run before every PR)

```bash
go test ./...     # unit + contract + TUI snapshot tests
go vet ./...
go build ./...    # must stay CGO-free
```

All of the following must pass:

| Gate | Requirement |
|---|---|
| Unit tests | Every non-trivial function covered; coverage ≥80% (NFR-MAINT-01) |
| Adapter contract tests | 4 adapters × 3 events = 12 cases |
| 5-state snapshots | Every screen has `*_loading`, `*_success`, `*_error`, `*_empty`, `*_partial` golden files (INV-PROMPT-MVP-006) |
| Focus-visibility tests | Focused widget shows a visible marker on every screen (INV-PROMPT-MVP-007) |
| Alt-text tests | Every sparkline / progress bar / color indicator has a text alternative (INV-PROMPT-MVP-008) |
| Benchmarks | Within [§6 performance budgets](analytics/PROMPT-MVP.md): first paint <100ms, frame <16ms, keystroke <50ms, transitions ≤100ms (INV-PROMPT-MVP-009) |
| High-contrast render | No layout breakage in high-contrast mode |

## Commit conventions

Use [Conventional Commits](https://www.conventionalcommits.org/) in **first-person voice** (ADR-006):

```
feat: add universal pre-commit hook for all 25+ agents

I chose a shell script wrapper over Go-based hook because shell is
universal across agents — git invokes the hook, not the agent.

Trade-off: shell is less type-safe than Go, but the hook is
fail-safe (INV-ARCH-004) — if caf binary is missing, the hook
warns and continues, never blocking the commit.

Traces to: FR-ADP-003, ADR-003, ADR-010, INV-ARCH-004
```

Rules:

1. **Every commit ends with a `Traces to: FR-, ADR-, INV-` footer** (INV-PROMPT-MVP-005). Files that don't trace to an FR-* requirement are marked `tooling` in the footer.
2. **UI commits** (anything under `pkg/ui/`) must also reference the UI invariant(s) they implement or strengthen, e.g. `Traces to: FR-UI-001, INV-PROMPT-MVP-006, INV-UI-004`.
3. English commit messages; Russian code comments; English identifiers.

## Scope discipline

MVP scope is **frozen** ([ROADMAP.md §2](analytics/ROADMAP.md)). If your change implements a feature from ALPHA/BETA/PROD/FUTURE (Decision Log, SRS, MCP, cross-repo sync, LLM features, Tier 2/3 adapters, …), don't — open an issue or add it to [BACKLOG.md](analytics/BACKLOG.md) instead (INV-PROMPT-MVP-001).

Out-of-scope requests get the standard reply: *"That is [phase] scope, not MVP. See ROADMAP.md §[N] and BACKLOG.md."*

## UI quality bar (`pkg/ui/`)

Any change to the TUI must uphold all of these — a screen that fails one is a defective deliverable:

- **5 mandatory states** on every screen: loading/skeleton, success, error, empty-with-CTA, partial-failure — no stubs, no TODOs (INV-PROMPT-MVP-006)
- **Visible focus** — the focused widget always shows a marker: `▸ ` prefix, reverse-video, or bright border (INV-PROMPT-MVP-007)
- **Color is never the sole carrier of meaning** — pair every color/sparkline/icon with text or an `altText` (INV-PROMPT-MVP-008)
- **Motion discipline** — transitions ≤100ms; no animation on typing or scrolling; motion only for state changes, navigation, or async feedback (INV-PROMPT-MVP-009)
- **No dead-ends** — every error/empty state has a recovery path and a CTA; every screen has a `?` help overlay (INV-UI-010)
- **Keyboard-only complete** — every feature reachable by keyboard; mouse never required (INV-UI-009)
- **Pre-compute styles at startup** — never call `os.ReadFile`/`yaml.Unmarshal`/style construction inside `View()` (16ms frame budget)
- **§21 self-review checklist** pasted into the PR description, all items checked (INV-PROMPT-MVP-010)

## Pull requests

- One logical change per PR; include the motivation and the trade-off you rejected ("I chose X because Y; Z was rejected because W").
- Paste the [§21 self-review checklist](analytics/PROMPT-MVP.md) into the PR description with all items checked — CI enforces the automated subset (items 1, 3, 5, 7, 11, 13).
- CI (`.github/workflows/ci.yml`) runs on every PR: `go vet`, the full test suite with the 80% coverage floor, UI benchmarks (≤100ms transition budget), and a binary smoke test.
- Mark non-traceable files as `tooling` in the PR description.
- Tests come with the code — never ship a behavior change without a test.

## Language conventions

| Artifact | Language |
|---|---|
| Code comments | Russian |
| Identifiers, API names | English |
| Commit messages, docs, PRs | English |
| Analytics documents | Bilingual RU/EN (see analytics/) |
