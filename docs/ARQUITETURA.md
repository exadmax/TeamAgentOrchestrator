# Arquitetura

## Visão geral

```
┌────────── Wails v3 (janela nativa) ───────────┐
│ frontend/  HTML+CSS+JS puro, visual Win95      │
│   abas: Início · Mesa · Organograma · Sprints  │
│         Tokens · Novo Agente · Clientes ·      │
│         Console                                │
└───────────────▲───────────────┬────────────────┘
       eventos  │               │ chamadas Call.ByName("main.App.X")
 (@wailsio/runtime)             ▼
┌──────────────── app.go (fachada) ──────────────┐
│ internal/team     organograma, agentes (tao.json)
│ internal/clients  catálogo e detecção dos CLIs
│ internal/runner   processo do CLI, timeout, log
│ internal/stream   JSON do CLI → Event comum
│ internal/failure  classificação de erros, esperas
│ internal/tasks    tarefas e sprints (tao.tasks.json)
│ internal/store    .tao/: sessões, relógios, trocas, etapas
│ internal/gitflow  worktree, commit, push, PR (gh), merge
│ internal/importer owners.json + docs/tasks → time e quadro
│ internal/fsx      troca atômica de arquivo no Windows
└────────────────────────────────────────────────┘
          │ os/exec
          ▼
   claude -p … --output-format stream-json
   codex exec --json …
   opencode run --format json …
```

## Conceitos

| Conceito | O que é | Onde fica |
|---|---|---|
| **Projeto** | Pasta (repositório git) que o time trabalha | qualquer caminho |
| **Time** | Organograma de agentes do projeto | `<projeto>/tao.json` (versionável) |
| **Cliente** | Um CLI de agente: executável, formato de saída, modelos e níveis | catálogo embutido + `tao.json` |
| **Agente** | Nó do organograma: nome, função, emblema, rosto, cliente, modelo, superior | `tao.json` → `agents` |
| **Sessão** | Uma execução de um agente: comando, eventos, uso de tokens, resultado | `<projeto>/.tao/sessions.jsonl` e `.tao/logs/` |
| **Relógio** | Quando um modelo sem cota volta a funcionar | `.tao/clocks.json` |
| **Troca** | Modelo substituto (temporário ou permanente) por agente/tarefa | `.tao/models.json` |
| **Tarefa / Sprint** | Atividades com responsável, revisor, etapas, branch, PR e histórico | `<projeto>/tao.tasks.json` (versionável) |
| **Reserva** | Agente que assume quando os modelos de outro estão sem cota | `tao.json` → `agents[].fallback` |
| **Fluxo git** | Branch por tarefa, base e prefixo | `tao.json` → `git` |
| **Worktree** | Cópia de trabalho da branch de uma tarefa | `<projeto>/.tao/worktrees/<id>` |

## Fluxo de uma sessão

1. A interface pede `RunAgent(agentID, prompt)`.
2. `clients` monta o argv a partir do modelo de comando do cliente (`{prompt}`, `{model}`).
3. `runner` inicia o processo, lê stdout linha a linha, aplica o timeout por silêncio
   (`IdleKill`, 300 s) e grava o log bruto em `.tao/logs/`.
4. Cada linha passa por `stream.Parse(client, line)` e vira um `stream.Event`
   (texto, ferramenta, uso, resultado, erro). O app emite o evento `line` para a interface.
5. No fim, `failure.Classify(exit, tail)` decide se houve erro (cota, auth, modelo,
   contexto, recusa, launch, transitório, timeout). Cota e esperas viram relógio em `store`.
6. A sessão é registrada em `sessions.jsonl` com uso de tokens, que alimenta a aba Tokens.

## Fluxo de uma tarefa

```
proposta ──Aprovar──▶ a fazer ──Rodar──▶ fazendo ──agente termina──▶ em revisão ──Revisar──▶ aprovada ──Fazer merge──▶ concluída
   (gerada por agente)                      │  ▲                         │                  (sem git: concluída direto)
                                            │  └── MUDANÇAS: ... ◀───────┘ (revisor de outro cliente)
                                            └── termina em pergunta ou falha ─▶ bloqueada (Mesa: Responder/Delegar)
```

Etapas guardadas em `.tao/state/<id>.json`: `start` (branch e worktree), `work`, `submit` (commit), `pr`,
`review`, `verdict`, `merge`. Sem git, só `start`, `work`, `review` e `verdict` aparecem.

- **Git:** `start` cria `tao/<id>-<slug>` num worktree (ou reaproveita a branch do `origin`, ou a pasta do projeto
  quando ela já está nessa branch). `submit` faz `git add -A` e o commit `ID: título`. `pr` envia a branch e roda
  `gh pr create` quando o `origin` é um servidor; com `origin` local ou sem `gh`, só envia. `merge` usa
  `gh pr merge` quando há PR; senão exige a pasta do projeto na base e limpa, faz `merge --no-ff` e envia a base.
  No fim remove o worktree e a branch.
- **Rota de uma sessão** (`routeLocked`): troca da tarefa → troca do cliente → modelo do agente → modelo do nível.
  Se esse modelo tem relógio de cota e a troca automática está ligada, usa o próximo modelo livre do cliente; sem
  nenhum, a reserva do agente. Uma falha de cota ou de modelo no meio da tarefa refaz a rota até 3 vezes.
  Falha de modelo (inexistente) põe o modelo fora por um dia.

- **Rodar:** o responsável recebe título, critério de pronto, resumo anterior, histórico recente e a
  resposta do Diogo (se houver). Se a última linha for uma pergunta, a tarefa bloqueia.
- **Revisar:** o revisor da tarefa ou, sem ele, um agente ativo de outro cliente (prefere o emblema
  `revisao`). Ele responde `APROVADO` ou `MUDANÇAS: ...`.
- **Gerar com Agente:** o agente devolve um array JSON de atividades; elas entram como propostas.
- Cada sessão tem tempo máximo de 45 min (agente em laço) além do timeout por silêncio de 300 s.
- Toda escrita de estado é atômica e repete a troca do arquivo por até meio segundo: no Windows a
  troca falha enquanto outro leitor está com o arquivo aberto (`internal/fsx`).

## Saída dos clientes (formato lido)

| Cliente | Flag | Eventos usados |
|---|---|---|
| Claude Code | `-p --output-format stream-json --verbose` | `system/init` (modelo, sessão), `assistant` (texto, `tool_use`, uso), `result` (uso total, custo, erro) |
| Codex CLI | `exec --json` | `thread.started`, `item.completed` (`agent_message`, `command_execution`…), `turn.completed` (uso), `error`/`turn.failed` |
| OpenCode | `run --format json` | `text`, `tool_use`, `step_finish` (tokens), `error` |

Amostras reais ficam em `testdata/stream/` e são a base dos testes de `internal/stream`.

## Origem (port do Python)

| Python (`tools/orq/orq.py`) | Go |
|---|---|
| `FAILURE_PATTERNS`, `classify_failure`, `parse_wait`, `retry_hint`, `hint_secs`, `fmt_secs`, `parse_tokens_used`, `last_question`, `error_model` | `internal/failure` |
| `.orq/sessions.jsonl`, `clocks.txt`, `models.json`, `state/<id>.json` (`_apply`, `PHASES`) | `internal/store` |
| `resolve_exe`, `owners.json → tools` | `internal/clients` |
| `run_session` (Popen, heartbeat, `IDLE_KILL`) | `internal/runner` |
| `owners.json → roles` | `internal/team` |
| menus de terminal | `frontend/` (Win95) |
| `act_start`, `act_submit`, `open_pr`, `act_merge_close` (git e gh) | `internal/gitflow` + `app_git.go` |
| `next_model`, `agent_options`, `fallback_tool` | `routeLocked` em `app_models.go` |
| `tools/cards/card.py` | `internal/tasks` (status iguais: proposed, ready, in_progress, review, done) |
