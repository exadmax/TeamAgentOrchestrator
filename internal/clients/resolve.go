package clients

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// npmPackages são os pacotes npm que instalam um shim .cmd/.bat no Windows.
var npmPackages = []string{
	"@anthropic-ai/claude-code",
	"@openai/codex",
	"@google/gemini-cli",
	"opencode-ai",
	"@opencode/cli",
}

// ResolveExe é o port de resolve_exe do orq.py: como exec.LookPath, mas no
// Windows troca o shim npm (.cmd/.bat) pelo .exe real dentro de
// node_modules/<pacote>. Um .cmd passa pelo cmd.exe, que quebra prompts com
// aspas, "|" e "<".
func ResolveExe(name string) (string, bool) {
	exe, err := exec.LookPath(name)
	if err != nil || exe == "" {
		return "", false
	}
	if runtime.GOOS != "windows" || !isNpmShim(exe) {
		return exe, true
	}
	if real := shimTarget(exe); real != "" {
		return real, true
	}
	if real := findNpmExe(filepath.Dir(exe), name); real != "" {
		return real, true
	}
	return exe, true
}

// shimExe matches the native binary an npm shim calls directly:
// "%dp0%\node_modules\@opencode\cli\bin\opencode.exe" %*
var shimExe = regexp.MustCompile(`(?i)"%dp0%\\(node_modules\\[^"]+\.exe)"`)

// shimTarget reads the .cmd shim and returns the .exe it launches, when it launches one
// (shims that run node with a .js file return "").
func shimTarget(shim string) string {
	data, err := os.ReadFile(shim)
	if err != nil {
		return ""
	}
	m := shimExe.FindSubmatch(data)
	if m == nil {
		return ""
	}
	path := filepath.Join(filepath.Dir(shim), string(m[1]))
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

// isNpmShim diz se o caminho é um shim de npm (.cmd ou .bat).
func isNpmShim(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".cmd", ".bat":
		return true
	default:
		return false
	}
}

// findNpmExe procura, sob <shimDir>/node_modules/<pacote>, um .exe cujo nome
// bata com o do comando pedido (sem extensão, ignorando maiúsculas).
func findNpmExe(shimDir, name string) string {
	want := stem(name)
	for _, pkg := range npmPackages {
		root := filepath.Join(shimDir, "node_modules", filepath.FromSlash(pkg))
		found := ""
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || found != "" {
				return nil
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".exe") {
				return nil
			}
			if strings.EqualFold(stem(path), want) {
				found = path
				return fs.SkipAll
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}

// stem devolve o nome do arquivo sem diretório e sem extensão, em minúsculas.
func stem(path string) string {
	base := filepath.Base(path)
	return strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base)))
}
