package clients

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// versionTimeout limita "<exe> --version" na detecção local.
	versionTimeout = 10 * time.Second
	// npmTimeout limita a consulta da última versão publicada no npm.
	npmTimeout = 20 * time.Second
)

// versionRe é a primeira versão x.y.z encontrada no texto.
var versionRe = regexp.MustCompile(`\d+\.\d+\.\d+`)

// Detected é o resultado da detecção de um cliente na máquina.
type Detected struct {
	Client
	Path      string `json:"path"`
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Latest    string `json:"latest"`
}

// ParseVersion devolve a primeira versão x.y.z do texto: "codex-cli 0.166.0" e
// "opencode v2.2.0" dão "0.166.0" e "2.2.0"; sem versão, devolve "".
func ParseVersion(out string) string {
	return versionRe.FindString(out)
}

// Detect resolve o executável do cliente e, se existir, roda "<exe> --version"
// com timeout de versionTimeout. Sem o executável, devolve Installed falso.
func Detect(ctx context.Context, c Client) Detected {
	if ctx == nil {
		ctx = context.Background()
	}
	d := Detected{Client: c}

	path, ok := ResolveExe(c.Exe)
	if !ok {
		return d
	}
	d.Path = path
	d.Installed = true

	vctx, cancel := context.WithTimeout(ctx, versionTimeout)
	defer cancel()
	// Alguns CLIs escrevem a versão em stderr e/ou saem com código != 0.
	cmd := exec.CommandContext(vctx, path, "--version")
	hideConsole(cmd)
	out, _ := cmd.CombinedOutput()
	d.Version = ParseVersion(string(out))
	return d
}

// DetectAll detecta todos os clientes em paralelo, mantendo a ordem da entrada.
func DetectAll(ctx context.Context, cs []Client) []Detected {
	out := make([]Detected, len(cs))
	var wg sync.WaitGroup
	for i := range cs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out[i] = Detect(ctx, cs[i])
		}(i)
	}
	wg.Wait()
	return out
}

// LatestNpm consulta a última versão publicada do pacote no npm
// ("npm view <pkg> version"), com timeout de npmTimeout.
func LatestNpm(ctx context.Context, pkg string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	nctx, cancel := context.WithTimeout(ctx, npmTimeout)
	defer cancel()

	cmd := exec.CommandContext(nctx, "npm", "view", pkg, "version")
	hideConsole(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("npm view %s version: %w", pkg, err)
	}
	version := ParseVersion(string(out))
	if version == "" {
		return "", fmt.Errorf("npm view %s version: no version in %q", pkg, strings.TrimSpace(string(out)))
	}
	return version, nil
}
