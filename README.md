# CodeAuthorityFirst (caf)

**Track who wrote what, when, and why.** CodeAuthorityFirst records provenance for AI coding sessions — which agent touched which file, from which prompt, through which iterations — and shows it in a TUI dashboard.

- **Single binary** — pure Go, no CGO, no runtime dependencies (Linux / macOS / Windows).
- **Offline & LLM-free** — no cloud calls, no API keys; the default works without any LLM.
- **Fail-safe hooks** — if `caf` is missing or crashes, git hooks warn and continue; commits are **never** blocked.
- **MVP status** — `0.4.1-mvp.1-dev`; scope is frozen per [ROADMAP.md §2](analytics/ROADMAP.md).

## Features (MVP)

| Feature | What it does |
|---|---|
| **Provenance tracker** | `caf who <file>` shows the session, prompt and iterations that modified a file (FR-PROV-003) |
| **Universal pre-commit hook** | One `.git/hooks/pre-commit` script fires for *all* AI agents — 25+ covered by a single hook (FR-ADP-003) |
| **Auto-detect** | `caf init --auto-detect` scans `$PATH` for known agents and recommends an adapter (FR-ADP-005) |
| **Tier 1 adapters** | Native wiring for **opencode** and **Claude Code** (FR-ADP-001) |
| **Tier 4 LSP fallback** | `gopls`-based symbol graph for any Go repo — zero-config coverage (FR-ADP-004) |
| **TUI dashboard** | 5 widgets: Today, Authorship Score, Active Agents, Recent Decisions, Cross-repo (FR-UI-001) |
| **Onboarding wizard** | 4-step interactive setup (`caf init --tui`), ~30 seconds (FR-UI-008) |
| **Adapters manager** | Capability matrix + 4-state health view for every adapter (FR-UI-008) |
| **`caf doctor`** | 5 health checks: `.caf/` validity, config schema, sessions, hook, LSP (FR-OBS-004) |
| **`context.md` generator** | Markdown digest of recent sessions for the next agent session (FR-PROV-001) |

## Requirements

- **Go 1.22+** — build-time only; the produced binary has no runtime dependencies
- **git** — required only for the pre-commit hook
- Optional: `gopls` (Tier 4 LSP fallback), `opencode` / `claude-code` (Tier 1 agents)

## Install

```bash
git clone https://github.com/Reider85/OryaCodeAuthorityFirst.git
cd OryaCodeAuthorityFirst
go build -o caf ./cmd/caf        # Windows: go build -o caf.exe ./cmd/caf
mv caf /usr/local/bin/           # or add the build dir to $PATH
```

Verify:

```console
$ caf --version
caf version 0.4.1-mvp.1-dev
```

No CGO, no Node.js, no Python — `go build` is the only step (INV-PROMPT-MVP-002).

## Quick start

```bash
cd your-repo
caf init --auto-detect    # create .caf/, install universal hook, detect agents
caf doctor                # 5 health checks
caf adapters list         # capability matrix + status
# ... work with an AI agent (opencode, Claude Code) ...
caf who src/main.go       # which session touched this file?
caf dashboard             # TUI hub
```

## 5-minute tour

### 1. Build (30 s)

```console
$ go build -o caf ./cmd/caf
$ caf --version
caf version 0.4.1-mvp.1-dev
```

### 2. Initialize (10 s)

```console
$ caf init --auto-detect
Initializing CodeAuthorityFirst in /path/to/your-repo
OK: .caf/ directory structure created
OK: universal pre-commit hook installed at .git/hooks/pre-commit

ID             TIER           STATUS
claude-code    T1 (native)    not detected
opencode       T1 (native)    installed
universal-lsp  T4 (fallback)  not detected

Recommended adapter: opencode (install hooks with: caf init --adapters opencode)

Next steps:
  1. caf dashboard  — open the TUI hub
  2. caf doctor     — run health checks
  3. use an AI agent (opencode, Claude Code) — sessions start appearing
```

*(output reflects this machine; your detection results will differ.)*

Prefer the interactive wizard? `caf init --tui` runs the 4-step onboarding — Welcome → Detection → Installation → Summary:

```
┌──────────────────────────────────────────┐
│ CodeAuthorityFirst 0.4.1 — onboarding    │
└──────────────────────────────────────────┘
▸ Step 1/4: Welcome
──────────────────────────────
Welcome to CodeAuthorityFirst!

This wizard will help you set up your
development environment with AI agents
and code quality tools.

Press [Enter] to continue, [q] to abort.

onboarding focus:                [Enter] next  [Esc] back  [q] abort  [?] help
```

### 3. Health check (5 s)

```console
$ caf doctor
CodeAuthorityFirst Health Report
Timestamp: 2026-11-01T20:14:16+04:00
Overall: warning
Summary: 1 passed, 0 failed, 2 warnings, 2 unknown

Health Checks:
  ✓ CAF directory validity: .caf/ directory is valid
  ⚠ Config schema: config.yml not found, using defaults
    Suggestion: Run 'caf init' to create config file
  ? Sessions present: No sessions yet
    Suggestion: Run an AI agent in this repository to start tracking
  ⚠ Hook installed: Universal pre-commit hook not installed
    Suggestion: Run 'caf init' to install the hook
  ? LSP responds: gopls not available
    Suggestion: Install Go and gopls for symbol graph support
```

Health states are 4-level (`✓` healthy / `⚠` degraded / `✗` unhealthy / `?` unknown) — never collapsed to a bare pass/fail.

### 4. Ask provenance (once a session exists)

```console
$ caf who internal/api/policies.go
Session 2026-10-24-a1b2c3
  Agent:        opencode
  Model:        glm-4.6
  Started:      2026-10-24T11:02:14+03:00
  InitialPrompt: Need to add rate limiter on POST /v1/policies.
  Iteration i1: read internal/api/policies.go
  Iteration i2: edit internal/api/policies.go — added middleware RateLimit(token, 10/min)
```

No sessions yet? `caf who` says so and exits 0 — it never blocks you.

### 5. Open the dashboard (30 s)

```console
$ caf dashboard
```

```
┌─────────────────────────────────────────┐
│ CodeAuthorityFirst 0.4.1 — dashboard    │
└─────────────────────────────────────────┘
▸ Today
┌──────────────────┐
│ 5 cards due      │
│ 3 pending        │
│ decisions        │
│ 1 new blindspots │
└──────────────────┘
  Authorship Score
┌──────────────────┐
│ ACS 3.8 / target │
│ 3.5              │
│ Recall  72%      │
│ Coverage 64%     │
│ 30-day: →        │
└──────────────────┘
  Active Agents
┌──────────────────┐
│ ✓ opencode       │
│   (healthy)      │
│ ⚠ claude-code    │
│   (degraded)     │
└──────────────────┘

  Recent Decisions
┌─────────────────────────────────────┐
│ • Use token bucket for rate limiting│
│ • Prefer YAML over SQLite           │
└─────────────────────────────────────┘
  Cross-repo
┌──────────────────┐
│ ⚠ OryaAgent ← Sec│
│   stale decision │
└──────────────────┘
```

`▸ ` marks the focused widget (Tab / Shift+Tab cycles), every non-text element has a text alternative, and each widget has loading / success / error / empty / partial states. Press `?` for the full keymap.

### 6. Inspect adapters (5 s)

```console
$ caf adapters list --tui
```

```
┌────────────────────────────────────────┐
│ CodeAuthorityFirst 0.4.1 — adapters    │
└────────────────────────────────────────┘
Adapters — capability matrix
ID              TIER   pre-session  post-edit  pre-commit  MCP   HEALTH
──────────────────────────────────────────────────────────────────────
▸ opencode       T1     native      native     native     native ✓ healthy
  claude-code    T1     native      native     native     native ⚠ degraded (hooks not wired)
  universal-lsp  T4     —           watcher    universal  no    ✓ healthy

adapters focus: [i] install  [c] claude  [a] auto-detect  [Enter] detail  [r] refresh  [?] help  [q] quit
```

Without `--tui` you get the same matrix as plain text:

```console
$ caf adapters list
ID             TIER           pre-session  post-edit  pre-commit  MCP  STATUS
claude-code    T1 (native)    native       native     native      yes  not detected
opencode       T1 (native)    native       native     native      yes  installed (hooks not wired — run caf init --adapters opencode)
universal-lsp  T4 (fallback)  wrapper      watcher    universal   no   not detected
```

You are done — the hook now records sessions for every agent that commits to this repo.

## Command reference

| Command | Purpose | Key flags |
|---|---|---|
| `caf init` | Create `.caf/`, install universal hook, wire adapters | `--auto-detect`, `--adapters opencode,claude-code`, `--tui`, `--repo` |
| `caf hook <event>` | Run `pre-session` / `post-edit` / `pre-commit` (fail-safe: never blocks) | `--repo`, `--session`, `--files`, `--message`, `--from-agent` |
| `caf who <file[:line]>` | Find the session that modified a file | `--repo` |
| `caf when --pattern <glob>` | **ALPHA scope** — MVP acknowledges and defers, no fabricated data | `--pattern` (required) |
| `caf context [--refresh]` | Show / regenerate `.caf/context.md` | `--refresh` |
| `caf dashboard` | Launch the TUI hub | `--tui` |
| `caf adapters list` | Capability matrix + detection status | `--tui` |
| `caf doctor [repo]` | 5 health checks (always exit 0) | — |
| `caf auto-detect` | Machine-wide agent scan (<5 s) | — |

## How it works

### `.caf/` layout (created by `caf init`)

```
.caf/
├── sessions/       # *.yml — one file per AI session (FR-PROV-002)
├── decisions/      # decision records (populated in ALPHA)
├── mental-model/   # SRS cards (BETA)
├── cache/          # LSP symbol-graph cache
├── metrics/        # reserved
├── reviews/        # reserved
├── manifest.yml    # repo identity + last sync
├── config.yml      # optional; defaults work without it
└── context.md      # generated by `caf context --refresh`
```

### Fail-safe hook (INV-ARCH-004)

`caf init` installs this into `.git/hooks/pre-commit` — one script, every agent:

```sh
#!/bin/sh
# .git/hooks/pre-commit — installed by caf init
# Universal: fires for all AI agents writing to this repo.
# Fail-safe: if caf binary is missing, warn and continue (never block commit).
CAF_PATH=$(command -v caf 2>/dev/null)
if [ -z "$CAF_PATH" ]; then
  echo "⚠ caf binary not in $PATH; pre-commit hook skipped (sessions continue)."
  exit 0
fi
exec "$CAF_PATH" hook pre-commit --repo "$(pwd)" "$@"
```

`caf hook …` itself also never blocks: hook errors are printed as warnings and exit 0.

### Adapter capability matrix (MVP)

| Adapter | Tier | pre-session | post-edit | pre-commit | MCP |
|---|---|---|---|---|---|
| `opencode` | T1 (native) | native | native | native | yes |
| `claude-code` | T1 (native) | native | native | native | yes |
| `universal-lsp` | T4 (fallback) | wrapper | watcher | universal | no |

Tier 2/3 adapters (Cursor, Cline, KiloCode, …) are ALPHA/BETA scope — see [BACKLOG.md](analytics/BACKLOG.md).

## Keyboard shortcuts (TUI)

| Key | Action |
|---|---|
| `Tab` / `Shift+Tab` | Cycle widgets |
| `j` / `k`, `↑` / `↓` | Move in lists |
| `g` / `G` | Top / bottom |
| `Enter` | Activate / open detail |
| `Esc` | Back / dismiss |
| `r` | Refresh |
| `q` | Quit |
| `?` | Help overlay (all bindings for the current screen) |
| `1`–`9` | Switch screen |
| `/` | Search |

Every feature is keyboard-reachable; mouse is never required (INV-UI-009).

## Scope & roadmap

MVP scope is **frozen**. Not here (by design):

- ❌ Decision Log — ALPHA · ❌ Mental Model / SRS — BETA · ❌ MCP server, cross-repo sync, LLM features — PROD
- ❌ Tier 1 adapters beyond opencode + Claude Code — ALPHA
- ❌ Sandboxed parser *implementation* — ALPHA (MVP ships the design only)

If you need any of these: that is [ROADMAP.md](analytics/ROADMAP.md) phase scope, see [BACKLOG.md](analytics/BACKLOG.md).

## Documentation

| Document | Contents |
|---|---|
| [analytics/ARD.md](analytics/ARD.md) | Requirements (FR-*), success criteria |
| [analytics/ARCHITECT.md](analytics/ARCHITECT.md) | Architecture, storage, hooks, adapters |
| [analytics/ROADMAP.md](analytics/ROADMAP.md) | Phases MVP → FUTURE, exit criteria |
| [analytics/UI.md](analytics/UI.md) | Screen specs, keyboard map, a11y audit |
| [analytics/ADR.md](analytics/ADR.md) | Architectural decisions |
| [analytics/BACKLOG.md](analytics/BACKLOG.md) | Deferred items per phase |
| [analytics/PROMPT-MVP.md](analytics/PROMPT-MVP.md) | The MVP generation contract this code follows |

## Development

```bash
go test ./...        # unit + contract + TUI snapshot tests (14 packages)
go vet ./...
go build ./...       # must stay CGO-free
```

Test suite includes: adapter contract tests (4 adapters × 3 events), 5-state `teatest` snapshots (3 screens × 5 states), focus-visibility and alt-text presence tests, and performance benchmarks against the §6 budgets.

Project layout follows [PROMPT-MVP.md §4](analytics/PROMPT-MVP.md): `cmd/caf/` (CLI), `internal/caf/` (storage, hooks, adapters, doctor, lsp, config), `pkg/ui/` (Bubble Tea TUI).
