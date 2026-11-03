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

## M4 — Internacionalização (concluído)

**Internacionalização (feita):** o sistema inteiro em inglês por padrão e português do Brasil como opção
(Ferramentas → Idioma): interface, mensagens e erros do backend, histórico das tarefas (gravado como chave +
argumentos, então muda de idioma junto), prompts enviados aos agentes, parecer do revisor nos dois idiomas,
README e documentos em inglês com versão em português, e a página do GitHub Pages bilíngue. Catálogos em
`internal/i18n/locales/` e `frontend/src/i18n/`, com teste de paridade de chaves e marcadores.

| Pacote | Dono no M4 (i18n) |
|---|---|
| `internal/*` traduzidos, histórico com chave | Codex |
| `frontend/` traduzido, troca de idioma na interface | Codex |
| `internal/i18n`, camada do app, prompts, documentos, Pages | Claude Code |

## M5 — Executável, instalador e releases para Windows (concluído)

- **Instalador** (NSIS, `build/windows/nsis/project.nsi`): instala só para o usuário, sem pedir administrador, em
  `%LOCALAPPDATA%\Programs\TeamAgent Orchestrator`, com atalhos no menu Iniciar e na área de trabalho, página da
  licença GPL, inglês e português (escolhido pelo idioma do Windows e lembrado pelo desinstalador), "executar agora"
  na última página, WebView2 instalado se faltar e atualização no lugar. Instalador e desinstalador esperam se o app
  estiver aberto.
- **Desinstalador:** remove o programa, os atalhos, o registro e o cache do WebView2; mantém as configurações e os projetos.
- **Primeira execução:** aberto pelo atalho, a pasta atual é a de instalação; por isso o app só abre a pasta atual
  se ela for um projeto (`tao.json`) ou repositório git. Senão, uma janela de boas-vindas pede uma pasta.
- **Releases:** `.github/workflows/release.yml` testa e gera o instalador no Windows a cada pull request (fica como
  artefato) e, numa tag `v*` igual à versão de `build/config.yml`, publica uma release no GitHub com
  `TeamAgent-amd64-installer.exe`. O site do Pages ganhou a seção Baixar, que aponta para a última release.
- **Versão:** um número só, em `build/config.yml`; o `TestVersionsMatch` confere as cópias (recursos do exe,
  instalador, pacotes, barra de status).
- **Licença:** GPL-3.0-or-later (`LICENSE`).

Gerar localmente: `wails3 package` (precisa do NSIS: `winget install NSIS.NSIS`) grava
`bin/TeamAgent-amd64-installer.exe`; `INSTALL_SCOPE=machine` gera um instalador para Arquivos de Programas.

O instalador ainda não tem assinatura digital: o SmartScreen mostra "editor desconhecido".

## M6 — A seguir

Instalar e atualizar clientes via npm pela aba Clientes, auto-reparo, builds Linux e macOS, assinatura de código.
