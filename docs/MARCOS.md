# Marcos

*[Read in English](MILESTONES.md)*

Um marco por vez; cada um termina com o dono do projeto vendo o resultado.

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

## M3 — Fluxo git e resiliência (concluído)

Pronto quando: cada tarefa roda numa branch própria em `.tao/worktrees/<id>`; o trabalho vira commit, a
branch é enviada e o PR é aberto com `gh` (ou o merge é local, sem GitHub); a aprovação deixa a tarefa
esperando o merge, feito por um botão; um modelo sem cota troca sozinho para outro modelo do cliente ou
para o agente reserva (com a troca automática ligada), e os relógios aparecem em Ferramentas → Modelos e
cotas; o revisor é sempre de outro cliente que não o de quem fez; Arquivo → Importar cartões traz os papéis
e os cartões do JinChronicles; a interface roda no Wails v3.

| Pacote | Dono no M3 |
|---|---|
| `internal/gitflow` | Codex |
| `internal/importer`, campos git e merge em `internal/tasks` | Codex |
| `app_git.go`, `app_models.go`, migração para o Wails v3, telas, `docs/index.html` | Claude Code |

## M4 — Distribuição e internacionalização (em andamento)

**Internacionalização (feita):** o sistema inteiro em inglês por padrão e português do Brasil como opção
(Ferramentas → Idioma): interface, mensagens e erros do backend, histórico das tarefas (gravado como chave +
argumentos, então muda de idioma junto), prompts enviados aos agentes, parecer do revisor nos dois idiomas,
README e documentos em inglês com versão em português, e a página do GitHub Pages bilíngue. Catálogos em
`internal/i18n/locales/` e `frontend/src/i18n/`, com teste de paridade de chaves e marcadores.

**A seguir:** instalar e atualizar clientes via npm pela aba Clientes, auto-reparo, instalador NSIS,
builds Linux e macOS.

| Pacote | Dono no M4 (i18n) |
|---|---|
| `internal/*` traduzidos, histórico com chave | Codex |
| `frontend/` traduzido, troca de idioma na interface | Codex |
| `internal/i18n`, camada do app, prompts, documentos, Pages | Claude Code |
