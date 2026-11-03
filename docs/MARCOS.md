# Marcos

Um marco por vez; cada um termina com o Diogo vendo o resultado.

## M1 — Núcleo em Go e janela Win95 (atual)

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

## M2 — Tarefas, sprints e Mesa do Agente

Tarefas e sprints genéricas (substituem o `card.py`), geração de atividades por um agente,
máquina de etapas por tarefa, Mesa do Agente (pergunta do agente → Responder, Delegar,
Arquivar) e Quadro de Tarefas.

## M3 — Fluxo git e resiliência

Começar/submeter/revisar/merge com branch e PR (`gh`), troca de modelo e agente reserva com
relógios visíveis, revisor sempre de outro cliente, importador do JinChronicles
(`owners.json` + `docs/tasks`).

## M4 — Distribuição

Instalar/atualizar clientes via npm pela aba Clientes, auto-reparo, instalador NSIS,
builds Linux e macOS.
