// Package gitflow manages isolated task branches and worktrees.
package gitflow

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const slugLimit = 40
const directoryMode = 0755

type Exec func(ctx context.Context, dir string, name string, args ...string) (string, error)

func DefaultExec(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	hideConsole(cmd)
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil {
		return result, fmt.Errorf("falha ao executar %s: %w: %s", name, err, result)
	}
	return result, nil
}

type Repo struct {
	Dir      string
	Base     string
	Prefix   string
	Exec     Exec
	LookPath func(string) (string, error)
}

func Open(dir string) *Repo { return &Repo{Dir: dir, Prefix: "tao/", Exec: DefaultExec} }
func (r *Repo) run(ctx context.Context, dir, name string, args ...string) (string, error) {
	fn := r.Exec
	if fn == nil {
		fn = DefaultExec
	}
	return fn(ctx, dir, name, args...)
}
func (r *Repo) git(ctx context.Context, dir string, args ...string) (string, error) {
	return r.run(ctx, dir, "git", append([]string{"-c", "core.quotepath=off"}, args...)...)
}
func (r *Repo) IsRepo(ctx context.Context) bool {
	out, err := r.git(ctx, r.Dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}
func (r *Repo) DefaultBase(ctx context.Context) string {
	if out, err := r.git(ctx, r.Dir, "symbolic-ref", "refs/remotes/origin/HEAD"); err == nil && strings.HasPrefix(out, "refs/remotes/origin/") {
		return strings.TrimPrefix(out, "refs/remotes/origin/")
	}
	if out, err := r.git(ctx, r.Dir, "symbolic-ref", "--short", "HEAD"); err == nil && out != "" {
		return out
	}
	return "main"
}
func (r *Repo) base(ctx context.Context) string {
	if r.Base != "" {
		return r.Base
	}
	return r.DefaultBase(ctx)
}
func (r *Repo) HasRemote(ctx context.Context) bool {
	out, err := r.git(ctx, r.Dir, "remote")
	if err != nil {
		return false
	}
	for _, name := range strings.Fields(out) {
		if name == "origin" {
			return true
		}
	}
	return false
}
func (r *Repo) HasGh() bool {
	fn := r.LookPath
	if fn == nil {
		fn = exec.LookPath
	}
	_, err := fn("gh")
	return err == nil
}
func BranchName(prefix, taskID, title string) string {
	var b strings.Builder
	separator := false
	for _, ch := range strings.ToLower(title) {
		switch {
		case strings.ContainsRune("áàâãäå", ch):
			ch = 'a'
		case strings.ContainsRune("éèêë", ch):
			ch = 'e'
		case strings.ContainsRune("íìîï", ch):
			ch = 'i'
		case strings.ContainsRune("óòôõö", ch):
			ch = 'o'
		case strings.ContainsRune("úùûü", ch):
			ch = 'u'
		case ch == 'ç':
			ch = 'c'
		}
		if ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			if separator && b.Len() > 0 {
				b.WriteByte('-')
			}
			separator = false
			b.WriteRune(ch)
		} else {
			separator = true
		}
	}
	slug := b.String()
	if len(slug) > slugLimit {
		slug = slug[:slugLimit]
		if i := strings.LastIndexByte(slug, '-'); i > 0 {
			slug = slug[:i]
		}
	}
	slug = strings.TrimRight(slug, "-")
	if slug == "" {
		return prefix + taskID
	}
	return prefix + taskID + "-" + slug
}
func (r *Repo) WorktreePath(taskID string) string {
	return filepath.Join(r.Dir, ".tao", "worktrees", taskID)
}
func (r *Repo) commonDir(ctx context.Context, dir string) (string, error) {
	out, err := r.git(ctx, dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(dir, out)
	}
	return filepath.Abs(out)
}
func (r *Repo) exclude(common string) error {
	path := filepath.Join(common, "info", "exclude")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("não foi possível ler exclusões: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == ".tao/" {
			return nil
		}
	}
	if err = os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return fmt.Errorf("não foi possível criar exclusões: %w", err)
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, []byte(".tao/\n")...)
	if err = os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("não foi possível salvar exclusões: %w", err)
	}
	return nil
}
func (r *Repo) Ensure(ctx context.Context, taskID, branch string) (string, error) {
	common, err := r.commonDir(ctx, r.Dir)
	if err != nil {
		return "", err
	}
	if err = r.exclude(common); err != nil {
		return "", err
	}
	path, err := filepath.Abs(r.WorktreePath(taskID))
	if err != nil {
		return "", err
	}
	if _, err = os.Stat(path); err == nil {
		other, e := r.commonDir(ctx, path)
		top, te := r.git(ctx, path, "rev-parse", "--show-toplevel")
		if e == nil && te == nil && samePath(common, other) && samePath(path, top) {
			return path, nil
		}
		return "", fmt.Errorf("a pasta da tarefa não pertence a este repositório")
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("não foi possível acessar a tarefa: %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return "", fmt.Errorf("não foi possível criar a pasta da tarefa: %w", err)
	}
	args := []string{"worktree", "add", path, branch}
	if _, e := r.git(ctx, r.Dir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); e != nil {
		// a branch that exists only on origin (an imported card in progress) keeps its commits
		start := r.base(ctx)
		if _, e := r.git(ctx, r.Dir, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch); e == nil {
			start = "origin/" + branch
		}
		args = []string{"worktree", "add", "-b", branch, path, start}
	}
	_, err = r.git(ctx, r.Dir, args...)
	return path, err
}
func samePath(a, b string) bool {
	a, _ = filepath.Abs(a)
	b, _ = filepath.Abs(b)
	if resolved, err := filepath.EvalSymlinks(a); err == nil {
		a = resolved
	}
	if resolved, err := filepath.EvalSymlinks(b); err == nil {
		b = resolved
	}
	return a == b
}
func (r *Repo) Commit(ctx context.Context, worktree, message string) (bool, error) {
	if _, err := r.git(ctx, worktree, "add", "-A"); err != nil {
		return false, err
	}
	out, err := r.git(ctx, worktree, "diff", "--cached", "--name-only")
	if err != nil || out == "" {
		return false, err
	}
	_, err = r.git(ctx, worktree, "commit", "-m", message)
	return err == nil, err
}
func (r *Repo) Ahead(ctx context.Context, branch string) (int, error) {
	out, err := r.git(ctx, r.Dir, "rev-list", "--count", r.base(ctx)+".."+branch)
	if err != nil {
		return 0, err
	}
	count, err := strconv.Atoi(out)
	if err != nil {
		return 0, fmt.Errorf("contagem de commits inválida: %w", err)
	}
	return count, nil
}
func (r *Repo) DiffStat(ctx context.Context, branch string) (string, error) {
	return r.git(ctx, r.Dir, "diff", "--stat", r.base(ctx)+"..."+branch)
}
func (r *Repo) Push(ctx context.Context, branch string) error {
	_, err := r.git(ctx, r.Dir, "push", "-u", "origin", branch)
	return err
}
func validURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}
func (r *Repo) OpenPR(ctx context.Context, branch, title, body string) (string, error) {
	if !r.HasGh() {
		return "", fmt.Errorf("gh não encontrado")
	}
	if out, err := r.run(ctx, r.Dir, "gh", "pr", "view", branch, "--json", "url", "-q", ".url"); err == nil && validURL(strings.TrimSpace(out)) {
		return strings.TrimSpace(out), nil
	}
	f, err := os.CreateTemp("", "tao-pr-*.md")
	if err != nil {
		return "", fmt.Errorf("não foi possível criar o texto do PR: %w", err)
	}
	defer os.Remove(f.Name())
	_, writeErr := f.WriteString(body)
	closeErr := f.Close()
	if writeErr != nil {
		return "", fmt.Errorf("não foi possível salvar o texto do PR: %w", writeErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("não foi possível fechar o texto do PR: %w", closeErr)
	}
	out, err := r.run(ctx, r.Dir, "gh", "pr", "create", "--base", r.base(ctx), "--head", branch, "--title", title, "--body-file", f.Name())
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	result := strings.TrimSpace(lines[len(lines)-1])
	if !validURL(result) {
		return "", fmt.Errorf("URL do PR não encontrada")
	}
	return result, nil
}
func (r *Repo) cleanBase(ctx context.Context, base string) bool {
	branch, err := r.git(ctx, r.Dir, "symbolic-ref", "--short", "HEAD")
	if err != nil || branch != base {
		return false
	}
	out, err := r.git(ctx, r.Dir, "status", "--porcelain")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// DefaultExec trims the first status line; restore its leading status space.
		if line[0] != ' ' && len(line) > 2 && line[1] == ' ' && line[2] != ' ' {
			line = " " + line
		}
		if len(line) < 4 {
			return false
		}
		path := strings.Trim(line[3:], "\"")
		if path == ".tao/" || strings.HasPrefix(path, ".tao/") || path == "tao.json" || path == "tao.tasks.json" {
			continue
		}
		return false
	}
	return true
}
func (r *Repo) Merge(ctx context.Context, taskID, branch, prURL, message string) error {
	base := r.base(ctx)
	if prURL != "" && r.HasGh() {
		if _, err := r.run(ctx, r.Dir, "gh", "pr", "merge", prURL, "--merge", "--delete-branch=false"); err != nil {
			return err
		}
		if r.cleanBase(ctx, base) {
			_, _ = r.git(ctx, r.Dir, "pull", "--ff-only")
		}
	} else {
		if !r.cleanBase(ctx, base) {
			return fmt.Errorf("a pasta do projeto precisa estar na %s e sem alterações para o merge local", base)
		}
		if _, err := r.git(ctx, r.Dir, "merge", "--no-ff", "-m", message, branch); err != nil {
			return err
		}
		if r.HasRemote(ctx) {
			if _, err := r.git(ctx, r.Dir, "push", "origin", base); err != nil {
				return err
			}
		}
	}
	_ = r.Remove(ctx, taskID, branch)
	return nil
}
func (r *Repo) Remove(ctx context.Context, taskID, branch string) error {
	_, err := r.git(ctx, r.Dir, "worktree", "remove", "--force", r.WorktreePath(taskID))
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not a working tree") {
		return err
	}
	_, _ = r.git(ctx, r.Dir, "branch", "-D", branch)
	if r.HasRemote(ctx) {
		_, _ = r.git(ctx, r.Dir, "push", "origin", "--delete", branch)
	}
	return nil
}
