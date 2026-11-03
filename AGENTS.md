# AGENTS.md — TeamAgentOrchestrator

Instruções para todos os agentes deste repositório (Claude Code, Codex, OpenCode, Gemini CLI).

## O projeto

Orquestrador de times de agentes de IA com interface gamificada estilo Windows 95.
Port em Go do `tools/orq` (Python) do JinChronicles, agora genérico: qualquer projeto,
qualquer time. Os agentes rodam pelos próprios clientes de linha de comando (Claude Code,
Codex CLI, OpenCode, Gemini CLI) e o orquestrador lê a saída JSON deles para a interface.

- Backend: Go 1.26+, sem dependências além do Wails e da biblioteca padrão.
- Interface: Wails v2 (janela nativa, WebView2) com HTML/CSS/JS puro em `frontend/`.
- Arquitetura e marcos: `docs/ARQUITETURA.md` e `docs/MARCOS.md`.

## Regras

1. **Idiomas:** conversa e documentos em pt-BR; código, identificadores, nomes de arquivo e
   commits em inglês. Texto que aparece na interface fica em pt-BR.
2. **Fronteiras:** só mexa nos arquivos que a tarefa lhe deu. Cada pacote em `internal/` tem
   um dono por marco (veja `docs/MARCOS.md`).
3. **Lógica pura testável:** `internal/failure`, `internal/store`, `internal/stream` e
   `internal/team` não importam Wails nem fazem I/O de rede. Todo pacote novo vem com testes
   (`go test ./...` precisa passar).
4. **Tempo e relógio injetáveis:** funções que dependem da hora recebem `now time.Time`
   (ou um `func() time.Time`), para os testes serem determinísticos.
5. **Arquivos de estado** ficam em `.tao/` dentro do projeto orquestrado (fora do git), em JSON
   ou JSONL legíveis. Nunca grave segredos.
6. **Sem números mágicos espalhados:** limites (timeout, esperas) ficam em constantes
   nomeadas no topo do pacote ou na configuração.
7. **Antes de entregar:** `gofmt -l .` vazio, `go vet ./...` e `go test ./...` passando.
8. Não faça commit nem push a menos que a tarefa peça.

## Comandos

```bash
go test ./...                 # testes do backend
wails dev                     # app com recarga
wails build                   # binário em build/bin/
```
