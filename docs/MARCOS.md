# Marcos

Um marco por vez; cada um termina com o Diogo vendo o resultado.

## M1 — Núcleo em Go e janela Win95 (concluído)

Pronto quando: `wails build` gera o `.exe`; a janela abre no visual Win95; o organograma
mostra e edita os agentes de `tao.json`; o assistente cria um agente novo; a aba Clientes
detecta Claude Code, Codex e OpenCode instalados; o Console roda um agente de verdade e mostra
a saída ao vivo; a aba Tokens soma o uso das sessões; `go test ./...` passa.

| Pacote | Dono no M1 |
|---|---|
| `internal/failure` | Codex |
| `internal/store` | Codex |
| `internal/clients` | OpenCode |
| `internal/stream`, `internal/runner`, `internal/team`, `app.go`, `frontend/` | Claude Code |

## M2 — Tarefas, sprints e Mesa do Agente (concluído)

Pronto quando: sprints criadas, iniciadas e fechadas com retro e transporte de atividades;
atividades criadas à mão ou geradas por um agente (chegam como propostas); cada atividade
roda no responsável, vai para revisão num agente de outro cliente e termina aprovada ou volta
com mudanças; a pergunta final de um agente bloqueia a atividade e aparece na Mesa, onde
Responder, Delegar e Arquivar agem sobre ela; o estado das etapas fica em `.tao/state/`.

| Pacote | Dono no M2 |
|---|---|
| `internal/tasks` | Codex |
| `internal/fsx`, `app_tasks.go`, telas Sprints/Mesa/tarefa, correções do runner | Claude Code |

## M3 — Fluxo git e resiliência

Começar/submeter/revisar/merge com branch e PR (`gh`), troca de modelo e agente reserva com
relógios visíveis, revisor sempre de outro cliente, importador do JinChronicles
(`owners.json` + `docs/tasks`).

## M4 — Distribuição

Instalar/atualizar clientes via npm pela aba Clientes, auto-reparo, instalador NSIS,
builds Linux e macOS.
