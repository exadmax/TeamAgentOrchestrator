# TeamAgent Orquestrador

Orquestrador de times de agentes de IA com cara de Windows 95. Você monta o organograma
(quem responde a quem, qual cliente e qual modelo cada agente usa), manda instruções e
acompanha tudo ao vivo: cada agente roda pelo **próprio cliente de linha de comando**
(Claude Code, Codex CLI, OpenCode, Gemini CLI) e o orquestrador lê a saída JSON deles.

Port em Go do `tools/orq` (Python) do JinChronicles, agora genérico para qualquer projeto.

## Telas

| Aba | O que faz |
|---|---|
| Início | o escritório: um computador por agente com a saída ao vivo; resumo do time e atividade recente |
| Mesa do Agente | o agente na mesa dizendo a última mensagem; Responder, Delegar, Arquivar |
| Organograma | árvore do time; propriedades, rostinho, pausar, novo subordinado, remover |
| Sprints | marco 2 |
| Tokens | uso por agente lido do JSON de cada cliente |
| Novo Agente | assistente em 3 passos |
| Clientes | CLIs encontrados na máquina, versões, atualização no npm e prompt de teste |
| Console | prompt estilo MS-DOS por agente, com a saída ao vivo |

## Rodar

Requisitos: Go 1.26+, Node 20+, [Wails v2](https://wails.io) (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
e pelo menos um cliente de agente instalado (`npm i -g @anthropic-ai/claude-code @openai/codex @opencode/cli`).

```bash
wails dev          # janela com recarga
wails build        # build/bin/TeamAgent.exe
go test ./...      # testes do backend
```

Ao abrir, o orquestrador usa o último projeto (ou a pasta atual). **Arquivo → Abrir projeto…**
escolhe outra pasta; se ela não tiver `tao.json`, ganha um time inicial (Orquestrador no Claude,
Arquiteto, Engenheiro no Codex e Revisor no OpenCode).

- `tao.json`: o time (versionável).
- `.tao/`: sessões, logs, relógios de cota e trocas de modelo (fora do git).

Teste de ponta a ponta com um CLI de verdade (gasta um pouco de cota):

```bash
TAO_E2E=1 TAO_E2E_CLIENT=codex go test -run TestEndToEnd .
```

Só a interface, sem o Go (API simulada): `npm --prefix frontend run dev`.

## Documentos

- [AGENTS.md](AGENTS.md): regras para os agentes que trabalham neste repositório
- [docs/ARQUITETURA.md](docs/ARQUITETURA.md): pacotes, fluxo de uma sessão e formatos lidos
- [docs/MARCOS.md](docs/MARCOS.md): M1 (atual) a M4

Os desenhos em `frontend/src/assets` vêm do protótipo feito no Claude Design para este projeto.
As fontes Pixelify Sans e VT323 são da licença SIL Open Font License.
