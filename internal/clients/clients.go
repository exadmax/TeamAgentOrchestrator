// Package clients descreve os CLIs de agente suportados (Claude Code, Codex CLI,
// OpenCode e Gemini CLI): como montar o argv de cada modo, quais modelos e níveis
// cada um oferece e como detectar a instalação local.
//
// Só biblioteca padrão: a montagem do argv e a exibição são lógica pura, e o I/O
// (LookPath, --version, npm) fica isolado em resolve.go e detect.go.
package clients

import (
	"strconv"
	"strings"
)

// Output é o formato de saída que internal/stream sabe ler.
type Output string

const (
	StreamJSON Output = "stream-json"
	JSONL      Output = "json"
	Text       Output = "text"
)

// Modos aceitos por BuildArgv.
const (
	modeHeadless    = "headless"
	modeInteractive = "interactive"
)

// Níveis do catálogo, equivalentes a barato/medio/forte no owners.json.
const (
	tierJunior = "junior"
	tierMedium = "medium"
	tierSenior = "senior"
)

// maxShownArg é o corte de cada argumento em ShowArgv.
const maxShownArg = 60

// Command é o comando base de um modo de execução.
type Command struct {
	Cmd        []string `json:"cmd"`         // comando base
	PromptFlag string   `json:"prompt_flag"` // "" = prompt posicional no fim
}

// Client é um CLI de agente e tudo que o orquestrador precisa saber dele.
type Client struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Exe         string            `json:"exe"`
	NpmPackage  string            `json:"npm_package"`
	Color       string            `json:"color"`
	Output      Output            `json:"output"`
	Headless    Command           `json:"headless"`
	Interactive Command           `json:"interactive"`
	ModelFlag   string            `json:"model_flag"`
	Models      []string          `json:"models"`        // do menor para o maior
	Tiers       map[string]string `json:"tiers"`         // "junior", "medium", "senior" -> modelo
	Env         []string          `json:"env,omitempty"` // KEY=VALUE extra para as sessões
}

// Catalog devolve os quatro clientes embutidos, com os modelos e níveis do
// owners.json do JinChronicles (barato=junior, medio=medium, forte=senior).
// Cada chamada devolve cópias novas, seguras para o chamador alterar.
func Catalog() []Client {
	return []Client{
		{
			ID:         "claude",
			Name:       "Claude Code",
			Exe:        "claude",
			NpmPackage: "@anthropic-ai/claude-code",
			Color:      "#c15f3c",
			Output:     StreamJSON,
			Headless: Command{
				Cmd: []string{
					"claude",
					"--output-format", "stream-json",
					"--verbose",
					"--permission-mode", "acceptEdits",
				},
				PromptFlag: "-p",
			},
			Interactive: Command{Cmd: []string{"claude"}},
			ModelFlag:   "--model",
			Models:      []string{"haiku", "sonnet", "opus"},
			Tiers: map[string]string{
				tierJunior: "haiku",
				tierMedium: "sonnet",
				tierSenior: "opus",
			},
		},
		{
			ID:         "codex",
			Name:       "Codex CLI",
			Exe:        "codex",
			NpmPackage: "@openai/codex",
			Color:      "#1a1a1a",
			Output:     JSONL,
			Headless: Command{
				Cmd: []string{
					"codex", "exec",
					"--json",
					"--skip-git-repo-check",
					"-s", "workspace-write",
				},
			},
			Interactive: Command{Cmd: []string{"codex"}},
			ModelFlag:   "-m",
			Models:      []string{"gpt-6-luna", "gpt-6.1-sol", "gpt-6-astra"},
			Tiers: map[string]string{
				tierJunior: "gpt-6-luna",
				tierMedium: "gpt-6.1-sol",
				tierSenior: "gpt-6-astra",
			},
		},
		{
			ID:         "opencode",
			Name:       "OpenCode",
			Exe:        "opencode",
			NpmPackage: "@opencode/cli",
			Color:      "#2e7d32",
			Output:     JSONL,
			Headless: Command{
				Cmd: []string{
					"opencode", "run",
					"--format", "json",
				},
			},
			Interactive: Command{
				Cmd:        []string{"opencode"},
				PromptFlag: "--prompt",
			},
			ModelFlag: "-m",
			Models: []string{
				"opencode/nemotron-3.5-lightning-free",
				"opencode/mimo-v2.6-flash-free",
				"opencode/glm-5.3-flash-free",
				"opencode/deepseek-v4.1-flash-free",
				"opencode/flint-40b-free",
				"opencode/big-pickle",
			},
			Tiers: map[string]string{
				tierJunior: "opencode/glm-5.3-flash-free",
				tierMedium: "opencode/big-pickle",
				tierSenior: "opencode/big-pickle",
			},
		},
		{
			ID:         "gemini",
			Name:       "Gemini CLI",
			Exe:        "gemini",
			NpmPackage: "@google/gemini-cli",
			Color:      "#1a73e8",
			Output:     StreamJSON,
			Headless: Command{
				Cmd: []string{
					"gemini",
					"--output-format", "stream-json",
					"--approval-mode", "auto_edit",
				},
				PromptFlag: "-p",
			},
			Interactive: Command{
				Cmd:        []string{"gemini"},
				PromptFlag: "-i",
			},
			ModelFlag: "-m",
			Env:       []string{"GEMINI_CLI_TRUST_WORKSPACE=true"}, // o projeto foi aberto de propósito no orquestrador
			Models:    []string{"gemini-3.8-flash"},
			Tiers: map[string]string{
				tierJunior: "gemini-3.8-flash",
				tierMedium: "gemini-3.8-flash",
				tierSenior: "gemini-3.8-flash",
			},
		},
	}
}

// Find procura um cliente pelo ID no catálogo dado.
func Find(cs []Client, id string) (Client, bool) {
	for _, c := range cs {
		if c.ID == id {
			return c, true
		}
	}
	return Client{}, false
}

// BuildArgv monta o argv do modo ("headless" ou "interactive"). O modelo, quando
// pedido e suportado, entra logo depois do comando base; o prompt vai no fim,
// atrás de PromptFlag quando o modo tem um. Prompt vazio não vira argumento.
// Modo desconhecido devolve nil.
func BuildArgv(c Client, mode, prompt, model string) []string {
	var cmd Command
	switch mode {
	case modeHeadless:
		cmd = c.Headless
	case modeInteractive:
		cmd = c.Interactive
	default:
		return nil
	}

	argv := make([]string, 0, len(cmd.Cmd)+4)
	argv = append(argv, cmd.Cmd...)
	if model != "" && c.ModelFlag != "" {
		argv = append(argv, c.ModelFlag, model)
	}
	if prompt != "" {
		if cmd.PromptFlag != "" {
			argv = append(argv, cmd.PromptFlag)
		}
		argv = append(argv, prompt)
	}
	return argv
}

// ShowArgv junta o argv para exibir na interface, com aspas em quem tem espaço e
// cada argumento cortado em maxShownArg caracteres com "…".
func ShowArgv(argv []string) string {
	parts := make([]string, len(argv))
	for i, arg := range argv {
		arg = shorten(arg, maxShownArg)
		if arg == "" || strings.ContainsAny(arg, " \t") {
			arg = strconv.Quote(arg)
		}
		parts[i] = arg
	}
	return strings.Join(parts, " ")
}

// shorten corta s em max runes e marca o corte com "…".
func shorten(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
