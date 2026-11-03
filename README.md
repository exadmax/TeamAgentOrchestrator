# TeamAgent Orchestrator

*[Leia em português](README.pt-BR.md)*

An orchestrator for teams of AI coding agents, with a Windows 95 look. You build the org chart (who reports
to whom, which client and which model each agent uses), hand out tasks and watch everything live: every agent
runs through **its own command-line client** (Claude Code, Codex CLI, OpenCode, Gemini CLI) and the
orchestrator reads their JSON output.

A Go port of the `tools/orq` Python script from JinChronicles, now generic for any project.

Project page with a screen tour: [exadmax.github.io/TeamAgentOrchestrator](https://exadmax.github.io/TeamAgentOrchestrator/)
(published by GitHub Pages from `docs/`).

## Download

Windows 10/11 (64-bit): **[download the installer](https://github.com/exadmax/TeamAgentOrchestrator/releases/latest/download/TeamAgent-amd64-installer.exe)**
([all releases](https://github.com/exadmax/TeamAgentOrchestrator/releases)). It installs for your user only, with no
administrator prompt. It is not code-signed yet, so SmartScreen may say *unknown publisher*: *More info → Run anyway*.
You still need at least one agent client (see below). To remove it: Settings → Apps → TeamAgent Orchestrator.

## Languages

The whole app speaks **English by default** and **Brazilian Portuguese** as an option: **Tools → Language**.
The choice covers the interface, the messages and errors from the backend, the task history and the prompts
sent to the agents (a reviewer may answer `APPROVED`/`CHANGES` or `APROVADO`/`MUDANÇAS`). It is saved in
`%APPDATA%\TeamAgentOrchestrator\settings.json`. Records written before a switch (session logs, step details)
keep the language they were written in.

## Screens

| Tab | What it does |
|---|---|
| Home | the office: one computer per agent with its live output; team summary and recent activity |
| Agent Desk | the agent says what it needs (a question, a review, a proposal); Reply, Delegate, Archive; Task Board |
| Org Chart | the team tree; properties, face, pause, new subordinate, remove |
| Sprints | sprints and tasks: create by hand or generate with an agent, run, review on another client, merge, close with a retro |
| Tokens | usage per agent, read from each client's JSON |
| New Agent | a 3-step wizard |
| Clients | CLIs found on the machine, versions, npm updates and a test prompt |
| Console | an MS-DOS style prompt per agent, with live output |

Windows in the **Tools** menu: *Models and quotas* (quota clocks, model swaps, automatic switching), *Git flow*
(branch per task, base, prefix) and *Language*. **File → Import cards** reads `docs/equipe/owners.json` and `docs/tasks/`.

## Build from source

Requirements: Go 1.26+, Node 20+, [Wails v3](https://wails.io) (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`)
and at least one agent client installed and logged in (`npm i -g @anthropic-ai/claude-code @openai/codex @opencode/cli`).

```bash
wails3 dev         # window with hot reload
wails3 build       # bin/TeamAgent.exe
wails3 package     # bin/TeamAgent-amd64-installer.exe (needs NSIS: winget install NSIS.NSIS)
go test ./...      # backend tests (includes the check that both languages have the same keys)
```

On start the orchestrator opens the last project, or the current folder when it is a project or a git repository;
otherwise it asks for one. **File → Open project…** picks another folder; if it has no `tao.json`, it gets a starter team (Orchestrator and Architect on Claude, Engineer on Codex,
QA Reviewer on OpenCode).

- `tao.json`: the team (versioned).
- `tao.tasks.json`: sprints and tasks (versioned).
- `.tao/`: sessions, logs, quota clocks, model swaps, task steps and task worktrees (outside git;
  the orchestrator adds `.tao/` to `.git/info/exclude` by itself).

In a git repository each task runs on its own branch (`tao/T001-title`) inside `.tao/worktrees/T001`: the agent
never touches your folder. When it finishes, the orchestrator commits, pushes the branch and opens the PR with
`gh` (when `origin` is on GitHub); after approval, **Merge** brings it into the base (through the PR, or locally).

End-to-end test with a real CLI (uses a little quota):

```bash
TAO_E2E=1 TAO_E2E_CLIENT=codex go test -run TestEndToEnd .
```

Only the interface, without Go (simulated API): `npm --prefix frontend run dev` and open the address in a normal browser.

To inspect the real window without screen access, run the app with `TAO_DEVTOOLS_PORT=9333`: WebView2 then opens
the DevTools protocol on that port (`http://127.0.0.1:9333/json`).

## Releases

`.github/workflows/release.yml` builds and tests on Windows for every pull request. To publish a version: change it in
`build/config.yml` (the test `TestVersionsMatch` lists the other files that carry it), merge, then push a tag
`v<version>`; the workflow attaches the installer to a new GitHub release.

## Translating

- Backend messages: `internal/i18n/locales/<lang>/*.json` (fmt formats, `%s`, `%[2]s`).
- Interface: `frontend/src/i18n/<lang>.json` (`{name}` placeholders).
- `go test ./internal/i18n/` fails when a key or a placeholder exists in one language and not in the other.

## Documents

- [AGENTS.md](AGENTS.md): rules for the agents working on this repository
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): packages, the flow of a session and of a task, formats read
- [docs/MILESTONES.md](docs/MILESTONES.md): M1 to M5 done (M5: Windows installer and releases); M6 next

The drawings in `frontend/src/assets` come from the prototype made in Claude Design for this project.
The Pixelify Sans and VT323 fonts are under the SIL Open Font License.

## License

TeamAgent Orchestrator is free software under the [GNU General Public License v3.0 or later](LICENSE).
