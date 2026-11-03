package clients

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPrompt = "faca isso"

func TestBuildArgv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		client string
		mode   string
		prompt string
		model  string
		want   []string
	}{
		{
			name:   "claude headless sem modelo",
			client: "claude",
			mode:   "headless",
			prompt: testPrompt,
			want: []string{
				"claude", "--output-format", "stream-json", "--verbose",
				"--permission-mode", "acceptEdits", "-p", testPrompt,
			},
		},
		{
			name:   "claude headless com modelo",
			client: "claude",
			mode:   "headless",
			prompt: testPrompt,
			model:  "sonnet",
			want: []string{
				"claude", "--output-format", "stream-json", "--verbose",
				"--permission-mode", "acceptEdits", "--model", "sonnet", "-p", testPrompt,
			},
		},
		{
			name:   "claude interativo sem modelo",
			client: "claude",
			mode:   "interactive",
			prompt: testPrompt,
			want:   []string{"claude", testPrompt},
		},
		{
			name:   "claude interativo com modelo",
			client: "claude",
			mode:   "interactive",
			prompt: testPrompt,
			model:  "opus",
			want:   []string{"claude", "--model", "opus", testPrompt},
		},
		{
			name:   "codex headless sem modelo",
			client: "codex",
			mode:   "headless",
			prompt: testPrompt,
			want: []string{
				"codex", "exec", "--json", "--skip-git-repo-check",
				"-s", "workspace-write", testPrompt,
			},
		},
		{
			name:   "codex headless com modelo",
			client: "codex",
			mode:   "headless",
			prompt: testPrompt,
			model:  "gpt-6.1-sol",
			want: []string{
				"codex", "exec", "--json", "--skip-git-repo-check",
				"-s", "workspace-write", "-m", "gpt-6.1-sol", testPrompt,
			},
		},
		{
			name:   "codex interativo sem modelo",
			client: "codex",
			mode:   "interactive",
			prompt: testPrompt,
			want:   []string{"codex", testPrompt},
		},
		{
			name:   "codex interativo com modelo",
			client: "codex",
			mode:   "interactive",
			prompt: testPrompt,
			model:  "gpt-6-astra",
			want:   []string{"codex", "-m", "gpt-6-astra", testPrompt},
		},
		{
			name:   "opencode headless sem modelo",
			client: "opencode",
			mode:   "headless",
			prompt: testPrompt,
			want:   []string{"opencode", "run", "--format", "json", testPrompt},
		},
		{
			name:   "opencode headless com modelo",
			client: "opencode",
			mode:   "headless",
			prompt: testPrompt,
			model:  "opencode/big-pickle",
			want: []string{
				"opencode", "run", "--format", "json",
				"-m", "opencode/big-pickle", testPrompt,
			},
		},
		{
			name:   "opencode interativo sem modelo",
			client: "opencode",
			mode:   "interactive",
			prompt: testPrompt,
			want:   []string{"opencode", "--prompt", testPrompt},
		},
		{
			name:   "opencode interativo com modelo",
			client: "opencode",
			mode:   "interactive",
			prompt: testPrompt,
			model:  "opencode/big-pickle",
			want:   []string{"opencode", "-m", "opencode/big-pickle", "--prompt", testPrompt},
		},
		{
			name:   "gemini headless sem modelo",
			client: "gemini",
			mode:   "headless",
			prompt: testPrompt,
			want: []string{
				"gemini", "--output-format", "stream-json",
				"--approval-mode", "auto_edit", "-p", testPrompt,
			},
		},
		{
			name:   "gemini headless com modelo",
			client: "gemini",
			mode:   "headless",
			prompt: testPrompt,
			model:  "gemini-3.8-flash",
			want: []string{
				"gemini", "--output-format", "stream-json",
				"--approval-mode", "auto_edit", "-m", "gemini-3.8-flash", "-p", testPrompt,
			},
		},
		{
			name:   "gemini interativo sem modelo",
			client: "gemini",
			mode:   "interactive",
			prompt: testPrompt,
			want:   []string{"gemini", "-i", testPrompt},
		},
		{
			name:   "gemini interativo com modelo",
			client: "gemini",
			mode:   "interactive",
			prompt: testPrompt,
			model:  "gemini-3.8-flash",
			want:   []string{"gemini", "-m", "gemini-3.8-flash", "-i", testPrompt},
		},
		{
			name:   "claude headless com prompt vazio nao passa -p",
			client: "claude",
			mode:   "headless",
			model:  "haiku",
			want: []string{
				"claude", "--output-format", "stream-json", "--verbose",
				"--permission-mode", "acceptEdits", "--model", "haiku",
			},
		},
		{
			name:   "gemini interativo com prompt vazio nao passa -i",
			client: "gemini",
			mode:   "interactive",
			want:   []string{"gemini"},
		},
	}

	catalog := Catalog()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, ok := Find(catalog, tt.client)
			if !ok {
				t.Fatalf("Find(%q) = false", tt.client)
			}
			got := BuildArgv(c, tt.mode, tt.prompt, tt.model)
			if !equalArgv(got, tt.want) {
				t.Fatalf("BuildArgv(%s, %s, %q, %q) = %#v, want %#v",
					tt.client, tt.mode, tt.prompt, tt.model, got, tt.want)
			}
		})
	}
}

func TestBuildArgvUnknownMode(t *testing.T) {
	t.Parallel()

	c, ok := Find(Catalog(), "claude")
	if !ok {
		t.Fatal("claude nao esta no catalogo")
	}
	if got := BuildArgv(c, "batch", testPrompt, ""); got != nil {
		t.Fatalf("BuildArgv modo desconhecido = %#v, want nil", got)
	}
}

func TestBuildArgvDoesNotMutateCatalog(t *testing.T) {
	t.Parallel()

	c, ok := Find(Catalog(), "codex")
	if !ok {
		t.Fatal("codex nao esta no catalogo")
	}
	before := append([]string(nil), c.Headless.Cmd...)

	_ = BuildArgv(c, "headless", testPrompt, "gpt-6-luna")
	_ = BuildArgv(c, "headless", "outro prompt", "gpt-6-astra")

	if !equalArgv(c.Headless.Cmd, before) {
		t.Fatalf("BuildArgv alterou o comando do catalogo: %#v, want %#v", c.Headless.Cmd, before)
	}
}

func TestShowArgv(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("a", maxShownArg+10)

	tests := []struct {
		name string
		argv []string
		want string
	}{
		{name: "vazio", argv: nil, want: ""},
		{name: "simples", argv: []string{"claude", "-p", "oi"}, want: "claude -p oi"},
		{
			name: "aspas em quem tem espaco",
			argv: []string{"claude", "-p", "resolva isso"},
			want: `claude -p "resolva isso"`,
		},
		{
			name: "argumento longo cortado",
			argv: []string{"codex", long},
			want: "codex " + strings.Repeat("a", maxShownArg) + "…",
		},
		{
			name: "argumento vazio entre aspas",
			argv: []string{"gemini", ""},
			want: `gemini ""`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ShowArgv(tt.argv); got != tt.want {
				t.Fatalf("ShowArgv(%#v) = %q, want %q", tt.argv, got, tt.want)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		out  string
		want string
	}{
		{name: "codex", out: "codex-cli 0.166.0", want: "0.166.0"},
		{name: "opencode com v", out: "opencode v2.2.0", want: "2.2.0"},
		{name: "claude com sufixo", out: "2.0.14 (Claude Code)", want: "2.0.14"},
		{name: "multilinha", out: "aviso\nclaude 1.2.3\n", want: "1.2.3"},
		{name: "sem versao", out: "command not found", want: ""},
		{name: "vazio", out: "", want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseVersion(tt.out); got != tt.want {
				t.Fatalf("ParseVersion(%q) = %q, want %q", tt.out, got, tt.want)
			}
		})
	}
}

func TestFind(t *testing.T) {
	t.Parallel()

	cs := Catalog()

	c, ok := Find(cs, "codex")
	if !ok {
		t.Fatal("Find(codex) = false, want true")
	}
	if c.Name != "Codex CLI" || c.Exe != "codex" {
		t.Fatalf("Find(codex) = %+v, want nome \"Codex CLI\" e exe \"codex\"", c)
	}

	if c, ok := Find(cs, "nao-existe"); ok {
		t.Fatalf("Find(nao-existe) = %+v, true; want zero, false", c)
	}
}

func TestCatalog(t *testing.T) {
	t.Parallel()

	cs := Catalog()
	if len(cs) != 4 {
		t.Fatalf("len(Catalog()) = %d, want 4", len(cs))
	}

	wantIDs := []string{"claude", "codex", "opencode", "gemini"}
	seen := make(map[string]bool, len(cs))
	for i, c := range cs {
		if c.ID != wantIDs[i] {
			t.Errorf("Catalog()[%d].ID = %q, want %q", i, c.ID, wantIDs[i])
		}
		if seen[c.ID] {
			t.Errorf("ID repetido no catalogo: %q", c.ID)
		}
		seen[c.ID] = true

		if c.Name == "" || c.Exe == "" || c.NpmPackage == "" {
			t.Errorf("%s: nome/exe/pacote npm vazio: %+v", c.ID, c)
		}
		if !strings.HasPrefix(c.Color, "#") {
			t.Errorf("%s: cor %q nao parece hex", c.ID, c.Color)
		}
		if c.Output != StreamJSON && c.Output != JSONL && c.Output != Text {
			t.Errorf("%s: output desconhecido %q", c.ID, c.Output)
		}
		if len(c.Headless.Cmd) == 0 || len(c.Interactive.Cmd) == 0 {
			t.Errorf("%s: comando headless/interativo vazio", c.ID)
		}
		if len(c.Models) == 0 {
			t.Errorf("%s: sem modelos", c.ID)
		}
		for _, tier := range []string{tierJunior, tierMedium, tierSenior} {
			model, ok := c.Tiers[tier]
			if !ok {
				t.Errorf("%s: nivel %q ausente", c.ID, tier)
				continue
			}
			if !contains(c.Models, model) {
				t.Errorf("%s: nivel %q aponta para modelo fora da lista: %q", c.ID, tier, model)
			}
		}
	}

	// O catalogo devolve copias novas: alterar uma nao afeta a proxima chamada.
	cs[0].Models[0] = "alterado"
	if Catalog()[0].Models[0] == "alterado" {
		t.Fatal("Catalog() compartilha estado entre chamadas")
	}
}

func TestDetectMissingClient(t *testing.T) {
	t.Parallel()

	d := Detect(context.Background(), Client{
		ID:  "fantasma",
		Exe: "tao-cliente-que-nao-existe-9f3c",
	})
	if d.Installed {
		t.Fatalf("Detect de executavel inexistente = %+v, want Installed false", d)
	}
	if d.Path != "" || d.Version != "" {
		t.Fatalf("Detect de executavel inexistente = %+v, want Path e Version vazios", d)
	}
	if d.ID != "fantasma" {
		t.Fatalf("Detect perdeu o cliente embutido: %+v", d)
	}
}

func equalArgv(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func TestShimTarget(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "node_modules", "@opencode", "cli", "bin", "opencode.exe")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(dir, "opencode.cmd")
	os.WriteFile(shim, []byte(`@ECHO off`+"\r\n"+`"%dp0%\node_modules\@opencode\cli\bin\opencode.exe"   %*`), 0o644)
	if got := shimTarget(shim); got != exe {
		t.Fatalf("shimTarget = %q, want %q", got, exe)
	}
	node := filepath.Join(dir, "codex.cmd")
	os.WriteFile(node, []byte(`"%_prog%"  "%dp0%\node_modules\@openai\codex\bin\codex.js" %*`), 0o644)
	if got := shimTarget(node); got != "" {
		t.Fatalf("node shim gave %q", got)
	}
}
