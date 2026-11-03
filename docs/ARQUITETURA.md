# Arquitetura

## Visão geral

```
┌──────────── Wails (janela nativa) ────────────┐
│ frontend/  HTML+CSS+JS puro, visual Win95      │
│   abas: Início · Mesa · Organograma · Sprints  │
│         Tokens · Novo Agente · Clientes ·      │
│         Console                                │
└───────────────▲───────────────┬────────────────┘
       eventos  │               │ chamadas (bindings)
   (runtime.Events)             ▼
┌──────────────── app.go (fachada) ──────────────┐
│ internal/team     organograma, agentes (tao.json)
│ internal/clients  catálogo e detecção dos CLIs
│ internal/runner   processo do CLI, timeout, log
│ internal/stream   JSON do CLI → Event comum
│ internal/failure  classificação de erros, esperas
│ internal/store    .tao/: sessões, relógios, trocas
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
| **Tarefa / Sprint** | Trabalho atribuído a agentes (marco 2) | `.tao/tasks/` |

## Fluxo de uma sessão

1. A interface pede `RunAgent(agentID, prompt)`.
2. `clients` monta o argv a partir do modelo de comando do cliente (`{prompt}`, `{model}`).
3. `runner` inicia o processo, lê stdout linha a linha, aplica o timeout por silêncio
   (`IdleKill`, 300 s) e grava o log bruto em `.tao/logs/`.
4. Cada linha passa por `stream.Parse(client, line)` e vira um `stream.Event`
   (texto, ferramenta, uso, resultado, erro). O app emite `session:event` para a interface.
5. No fim, `failure.Classify(exit, tail)` decide se houve erro (cota, auth, modelo,
   contexto, recusa, launch, transitório, timeout). Cota e esperas viram relógio em `store`.
6. A sessão é registrada em `sessions.jsonl` com uso de tokens, que alimenta a aba Tokens.

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
| `tools/cards/card.py` | marco 2 (`internal/tasks`) |
