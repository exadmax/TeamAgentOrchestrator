# TeamAgent Orquestrador

Orquestrador de times de agentes de IA com cara de Windows 95. Você monta o organograma
(quem responde a quem, qual cliente e qual modelo cada agente usa), manda instruções e
acompanha tudo ao vivo: cada agente roda pelo **próprio cliente de linha de comando**
(Claude Code, Codex CLI, OpenCode, Gemini CLI) e o orquestrador lê a saída JSON deles.

Port em Go do `tools/orq` (Python) do JinChronicles, agora genérico para qualquer projeto.

Página do projeto, com tour pelas telas: [`docs/index.html`](docs/index.html) (publicada pelo GitHub Pages a partir de `docs/`).

## Telas

| Aba | O que faz |
|---|---|
| Início | o escritório: um computador por agente com a saída ao vivo; resumo do time e atividade recente |
| Mesa do Agente | o agente diz o que precisa (pergunta, revisão, proposta); Responder, Delegar, Arquivar; Quadro de Tarefas |
| Organograma | árvore do time; propriedades, rostinho, pausar, novo subordinado, remover |
| Sprints | sprints e atividades: criar à mão ou gerar com um agente, rodar, revisar em outro cliente, fazer merge, fechar com retro |
| Tokens | uso por agente lido do JSON de cada cliente |
| Novo Agente | assistente em 3 passos |
| Clientes | CLIs encontrados na máquina, versões, atualização no npm e prompt de teste |
| Console | prompt estilo MS-DOS por agente, com a saída ao vivo |

Janelas do menu **Ferramentas**: *Modelos e cotas* (relógios de cota, trocas de modelo, troca automática) e
*Fluxo git* (branch por tarefa, base, prefixo). **Arquivo → Importar cartões** lê `docs/equipe/owners.json` e `docs/tasks/`.

## Rodar

Requisitos: Go 1.26+, Node 20+, [Wails v3](https://wails.io) (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`)
e pelo menos um cliente de agente instalado (`npm i -g @anthropic-ai/claude-code @openai/codex @opencode/cli`).

```bash
wails3 dev         # janela com recarga
wails3 build       # bin/TeamAgent.exe
go test ./...      # testes do backend
```

Ao abrir, o orquestrador usa o último projeto (ou a pasta atual). **Arquivo → Abrir projeto…**
escolhe outra pasta; se ela não tiver `tao.json`, ganha um time inicial (Orquestrador no Claude,
Arquiteto, Engenheiro no Codex e Revisor no OpenCode).

- `tao.json`: o time (versionável).
- `tao.tasks.json`: sprints e atividades (versionável).
- `.tao/`: sessões, logs, relógios de cota, trocas de modelo, etapas e os worktrees das tarefas (fora do git;
  o orquestrador põe `.tao/` em `.git/info/exclude` sozinho).

Em um repositório git, cada tarefa roda numa branch própria (`tao/T001-titulo`) dentro de `.tao/worktrees/T001`:
o agente nunca mexe na sua pasta. Ao terminar, o orquestrador faz o commit, envia a branch e abre o PR com o `gh`
(quando há `origin` no GitHub); depois da aprovação, **Fazer merge** junta na base (pelo PR, ou localmente).

Teste de ponta a ponta com um CLI de verdade (gasta um pouco de cota):

```bash
TAO_E2E=1 TAO_E2E_CLIENT=codex go test -run TestEndToEnd .
```

Só a interface, sem o Go (API simulada): `npm --prefix frontend run dev` e abra o endereço num navegador comum.

Para inspecionar a janela de verdade sem acesso à tela, rode o app com `TAO_DEVTOOLS_PORT=9333`: o WebView2 abre
o protocolo DevTools nessa porta (`http://127.0.0.1:9333/json`).

## Documentos

- [AGENTS.md](AGENTS.md): regras para os agentes que trabalham neste repositório
- [docs/ARQUITETURA.md](docs/ARQUITETURA.md): pacotes, fluxo de uma sessão e formatos lidos
- [docs/MARCOS.md](docs/MARCOS.md): M1, M2 e M3 concluídos; M4 a seguir

Os desenhos em `frontend/src/assets` vêm do protótipo feito no Claude Design para este projeto.
As fontes Pixelify Sans e VT323 são da licença SIL Open Font License.
