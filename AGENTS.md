# AGENTS.md — TeamAgentOrchestrator

Instructions for every agent working on this repository (Claude Code, Codex, OpenCode, Gemini CLI).

## The project

An orchestrator for teams of AI coding agents with a gamified Windows 95 interface. A Go port of the
`tools/orq` Python script from JinChronicles, now generic: any project, any team. Agents run through their own
command-line clients (Claude Code, Codex CLI, OpenCode, Gemini CLI) and the orchestrator reads their JSON output.

- Backend: Go 1.26+, no dependencies besides Wails and the standard library.
- Interface: Wails v3 (native window, WebView2) with plain HTML/CSS/JS in `frontend/`; the JS calls Go through
  `Call.ByName("main.App.<Method>")` (`frontend/src/api.js`).
- Architecture and milestones: `docs/ARCHITECTURE.md` and `docs/MILESTONES.md` (Portuguese versions:
  `docs/ARQUITETURA.md`, `docs/MARCOS.md`).

## Rules

1. **Languages:** code, identifiers, file names, comments and commits in English. The product speaks English by
   default and Brazilian Portuguese as an option: **no user-facing string literal in code**. Backend text goes
   through `internal/i18n` (`i18n.T` / `i18n.Err`, catalogs in `internal/i18n/locales/<lang>/*.json`); interface
   text through `t()` (`frontend/src/i18n/<lang>.json`). Every new key goes into **both** languages with the same
   placeholders (`go test ./internal/i18n/` checks it). Prompts sent to agents are user-facing too.
   Documents: English first (`README.md`, `docs/*.md`), with the Portuguese version kept in sync
   (`README.pt-BR.md`, `docs/ARQUITETURA.md`, `docs/MARCOS.md`).
2. **Boundaries:** only touch the files your task gave you. Each package in `internal/` has one owner per
   milestone (see `docs/MILESTONES.md`).
3. **Testable pure logic:** `internal/failure`, `internal/store`, `internal/stream`, `internal/team` and
   `internal/i18n` do not import Wails nor do network I/O. Every new package comes with tests
   (`go test ./...` must pass).
4. **Injectable time:** functions that depend on the time take `now time.Time` (or a `func() time.Time`), so tests
   are deterministic.
5. **State files** live in `.tao/` inside the orchestrated project (outside git), as readable JSON or JSONL.
   Never write secrets.
6. **No scattered magic numbers:** limits (timeouts, waits) are named constants at the top of the package or
   in configuration.
7. **Before handing over:** `gofmt -l .` empty, `go vet ./...` and `go test ./...` passing.
8. Do not commit or push unless the task asks for it.

## Commands

```bash
go test ./...                 # backend tests
wails3 dev                    # app with hot reload
wails3 build                  # binary in bin/
```
