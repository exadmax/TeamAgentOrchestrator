# Architecture

*[Leia em português](ARQUITETURA.md)*

## Overview

```
┌────────── Wails v3 (native window) ───────────┐
│ frontend/  plain HTML+CSS+JS, Win95 look       │
│   tabs: Home · Desk · Org Chart · Sprints      │
│         Tokens · New Agent · Clients · Console │
│   frontend/src/i18n: en.json, pt-BR.json       │
└───────────────▲───────────────┬────────────────┘
        events  │               │ calls Call.ByName("main.App.X")
 (@wailsio/runtime)             ▼
┌─────────────── app.go (facade) ────────────────┐
│ internal/i18n     messages in en / pt-BR
│ internal/team     org chart, agents (tao.json)
│ internal/clients  catalog and detection of the CLIs
│ internal/runner   the CLI process, timeouts, log
│ internal/stream   CLI JSON → common Event
│ internal/failure  error classification, waits
│ internal/tasks    tasks and sprints (tao.tasks.json)
│ internal/store    .tao/: sessions, clocks, swaps, steps
│ internal/gitflow  worktree, commit, push, PR (gh), merge
│ internal/importer owners.json + docs/tasks → team and board
│ internal/fsx      atomic file replace on Windows
└────────────────────────────────────────────────┘
          │ os/exec
          ▼
   claude -p … --output-format stream-json
   codex exec --json …
   opencode run --format json …
```

## Concepts

| Concept | What it is | Where it lives |
|---|---|---|
| **Project** | Folder (git repository) the team works on | any path |
| **Team** | The project's org chart of agents | `<project>/tao.json` (versioned) |
| **Client** | An agent CLI: executable, output format, models and tiers | built-in catalog + `tao.json` |
| **Agent** | A node of the chart: name, role, emblem, face, client, model, boss | `tao.json` → `agents` |
| **Session** | One run of an agent: command, events, token usage, result | `<project>/.tao/sessions.jsonl` and `.tao/logs/` |
| **Clock** | When a model out of quota works again | `.tao/clocks.json` |
| **Swap** | Replacement model (temporary or permanent) per client or task | `.tao/models.json` |
| **Task / Sprint** | Work items with assignee, reviewer, steps, branch, PR and history | `<project>/tao.tasks.json` (versioned) |
| **Reserve** | Agent that takes over when another agent's models are out of quota | `tao.json` → `agents[].fallback` |
| **Git flow** | Branch per task, base and prefix | `tao.json` → `git` |
| **Worktree** | Working copy of a task's branch | `<project>/.tao/worktrees/<id>` |
| **Language** | `en` (default) or `pt-BR` for UI, messages and prompts | `%APPDATA%\TeamAgentOrchestrator\settings.json` |

## Languages (i18n)

- `internal/i18n` embeds `locales/<lang>/*.json` (fmt formats). `i18n.T(key, args...)` and `i18n.Err(...)` use the
  current language; `i18n.Set` changes it. Every user-facing string of the backend goes through it: errors,
  failure help, task history, the starter team, importer texts, git step details and the prompts sent to agents.
- Task history is stored as `key` + `args` (plus an English `what` for old readers), so it is shown in the current
  language even after a switch. Session logs and step details stay in the language they were written in.
- The frontend has its own catalogs (`frontend/src/i18n/*.json`, `{name}` placeholders) and translates the codes
  the backend sends (agent status, task status, steps, clock sources).
- `App.SetLanguage` saves the choice, retitles the window and emits `language`, `snapshot` and `board`; the
  frontend rebuilds the whole window.
- `go test ./internal/i18n/` checks that both languages have the same keys and placeholders, in both layers.
- The reviewer verdict is read in both languages: `APPROVED`/`CHANGES` or `APROVADO`/`MUDANÇAS`.

## The flow of a session

1. The interface calls `RunAgent(agentID, prompt)`.
2. `clients` builds the argv from the client's command template (`{prompt}`, `{model}`).
3. `runner` starts the process, reads stdout line by line, applies the silence timeout
   (`IdleKill`, 300 s) and writes the raw log to `.tao/logs/`.
4. Each line goes through `stream.Parse(client, line)` and becomes a `stream.Event`
   (text, tool, usage, result, error). The app emits the `line` event to the interface.
5. At the end, `failure.Classify(exit, tail)` decides whether there was an error (quota, auth, model,
   context, refusal, launch, transient, timeout). Quotas and waits become a clock in `store`.
6. The session is recorded in `sessions.jsonl` with its token usage, which feeds the Tokens tab.

## The flow of a task

```
proposed ──Approve──▶ ready ──Run──▶ in progress ──agent ends──▶ review ──Review──▶ approved ──Merge──▶ done
 (made by an agent)                     │  ▲                          │              (no git: done right away)
                                        │  └── CHANGES: ... ◀─────────┘ (reviewer on another client)
                                        └── ends on a question or a failure ─▶ blocked (Desk: Reply/Delegate)
```

Steps stored in `.tao/state/<id>.json`: `start` (branch and worktree), `work`, `submit` (commit), `pr`,
`review`, `verdict`, `merge`. Without git only `start`, `work`, `review` and `verdict` show up.

- **Git:** `start` creates `tao/<id>-<slug>` in a worktree (or reuses the branch from `origin`, or the project
  folder when it is already on that branch). `submit` runs `git add -A` and commits `ID: title`. `pr` pushes the
  branch and runs `gh pr create` when `origin` is a server; with a local `origin` or no `gh`, it only pushes.
  `merge` uses `gh pr merge` when there is a PR; otherwise it needs the project folder on the base and clean, runs
  `merge --no-ff` and pushes the base. At the end it removes the worktree and the branch.
- **Route of a session** (`routeLocked`): task swap → client swap → agent model → tier model. If that model has a
  quota clock and automatic switching is on, it uses the next free model of the client; with none, the agent's
  reserve. A quota or model failure in the middle of a task re-routes it up to 3 times. A model failure
  (model does not exist) keeps that model off for a day.
- **Run:** the assignee gets the title, done criterion, previous summary, recent history and the user's answer
  (if any). If the last line is a question, the task is blocked.
- **Review:** the task's reviewer or, without one, an active agent on another client (prefers the `revisao`
  emblem), always on a client different from whoever did the work.
- **Generate with Agent:** the agent returns a JSON array of tasks; they arrive as proposals.
- Every session has a 45 min maximum (looping agent) besides the 300 s silence timeout.
- Every state write is atomic and retries the file replace for up to half a second: on Windows the replace
  fails while another reader has the file open (`internal/fsx`).

## Client output (format read)

| Client | Flag | Events used |
|---|---|---|
| Claude Code | `-p --output-format stream-json --verbose` | `system/init` (model, session), `assistant` (text, `tool_use`, usage), `result` (total usage, cost, error) |
| Codex CLI | `exec --json` | `thread.started`, `item.completed` (`agent_message`, `command_execution`…), `turn.completed` (usage), `error`/`turn.failed` |
| OpenCode | `run --format json` | `text`, `tool_use`, `step_finish` (tokens), `error` |

Real samples live in `testdata/stream/` and back the tests of `internal/stream`.

## Origin (port from Python)

| Python (`tools/orq/orq.py`) | Go |
|---|---|
| `FAILURE_PATTERNS`, `classify_failure`, `parse_wait`, `retry_hint`, `hint_secs`, `fmt_secs`, `parse_tokens_used`, `last_question`, `error_model` | `internal/failure` |
| `.orq/sessions.jsonl`, `clocks.txt`, `models.json`, `state/<id>.json` (`_apply`, `PHASES`) | `internal/store` |
| `resolve_exe`, `owners.json → tools` | `internal/clients` |
| `run_session` (Popen, heartbeat, `IDLE_KILL`) | `internal/runner` |
| `owners.json → roles` | `internal/team` |
| terminal menus | `frontend/` (Win95) |
| `act_start`, `act_submit`, `open_pr`, `act_merge_close` (git and gh) | `internal/gitflow` + `app_git.go` |
| `next_model`, `agent_options`, `fallback_tool` | `routeLocked` in `app_models.go` |
| `tools/cards/card.py` | `internal/tasks` (same statuses: proposed, ready, in_progress, review, done) |
