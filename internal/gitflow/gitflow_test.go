package gitflow

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func command(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := DefaultExec(context.Background(), dir, "git", args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func repository(t *testing.T) *Repo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	dir := t.TempDir()
	command(t, dir, "init", "-b", "main")
	command(t, dir, "config", "user.name", "Gitflow Test")
	command(t, dir, "config", "user.email", "gitflow@example.com")
	command(t, dir, "config", "core.excludesFile", filepath.Join(dir, ".git", "empty-exclude"))
	write(t, filepath.Join(dir, "initial.txt"), "initial\n")
	command(t, dir, "add", "initial.txt")
	command(t, dir, "commit", "-m", "Initial commit")
	return Open(dir)
}
func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func task(t *testing.T, r *Repo) (string, string) {
	t.Helper()
	branch := BranchName(r.Prefix, "T1", "Example task")
	path, err := r.Ensure(context.Background(), "T1", branch)
	if err != nil {
		t.Fatal(err)
	}
	return path, branch
}
func TestBranchName(t *testing.T) {
	for _, tt := range []struct{ title, want string }{
		{"Áàâã Ç éê í óôõ ú", "tao/T1-aaaa-c-ee-i-ooo-u"},
		{"", "tao/T1"}, {"!?", "tao/T1"},
		{strings.Repeat("a", 45), "tao/T1-" + strings.Repeat("a", 40)},
		{strings.Repeat("abc ", 15), "tao/T1-" + strings.TrimSuffix(strings.Repeat("abc-", 10), "-")},
	} {
		if got := BranchName("tao/", "T1", tt.title); got != tt.want {
			t.Errorf("BranchName(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}
func TestWorktreeLifecycle(t *testing.T) {
	ctx := context.Background()
	r := repository(t)
	if !r.IsRepo(ctx) || Open(t.TempDir()).IsRepo(ctx) {
		t.Fatal("incorrect repository detection")
	}
	if got := r.DefaultBase(ctx); got != "main" {
		t.Fatal(got)
	}
	if r.HasRemote(ctx) {
		t.Fatal("unexpected remote")
	}
	path, branch := task(t, r)
	if command(t, path, "branch", "--show-current") != branch {
		t.Fatal("incorrect branch")
	}
	again, err := r.Ensure(ctx, "T1", branch)
	if err != nil || again != path {
		t.Fatalf("reuse: %q %v", again, err)
	}
	data, err := os.ReadFile(filepath.Join(r.Dir, ".git", "info", "exclude"))
	if err != nil || strings.Count(string(data), ".tao/\n") != 1 {
		t.Fatalf("exclude: %q %v", data, err)
	}
	if got := command(t, r.Dir, "status", "--porcelain"); got != "" {
		t.Fatal(got)
	}
	if committed, err := r.Commit(ctx, path, "Empty"); committed || err != nil {
		t.Fatalf("empty commit: %v %v", committed, err)
	}
	write(t, filepath.Join(path, "feature.txt"), "feature\n")
	if committed, err := r.Commit(ctx, path, "Add feature"); !committed || err != nil {
		t.Fatalf("commit: %v %v", committed, err)
	}
	if count, err := r.Ahead(ctx, branch); count != 1 || err != nil {
		t.Fatalf("ahead: %d %v", count, err)
	}
	if stat, err := r.DiffStat(ctx, branch); err != nil || !strings.Contains(stat, "feature.txt") {
		t.Fatalf("diff: %s %v", stat, err)
	}
	write(t, filepath.Join(r.Dir, "initial.txt"), "dirty\n")
	if err := r.Merge(ctx, "T1", branch, "", "Merge feature"); err == nil || !strings.Contains(err.Error(), "sem alterações") {
		t.Fatalf("dirty merge: %v", err)
	}
	command(t, r.Dir, "restore", "initial.txt")
	write(t, filepath.Join(r.Dir, "tao.json"), "{}")
	write(t, filepath.Join(r.Dir, "tao.tasks.json"), "{}")
	if err := r.Merge(ctx, "T1", branch, "", "Merge feature"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "feature.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree still exists: %v", err)
	}
	if _, err := r.git(ctx, r.Dir, "show-ref", "--verify", "refs/heads/"+branch); err == nil {
		t.Fatal("branch still exists")
	}
}
func TestRemote(t *testing.T) {
	ctx := context.Background()
	r := repository(t)
	remote := t.TempDir()
	command(t, remote, "init", "--bare")
	command(t, r.Dir, "remote", "add", "origin", remote)
	command(t, r.Dir, "push", "-u", "origin", "main")
	command(t, r.Dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	if !r.HasRemote(ctx) || r.DefaultBase(ctx) != "main" {
		t.Fatal("remote detection failed")
	}
	path, branch := task(t, r)
	write(t, filepath.Join(path, "remote.txt"), "remote\n")
	if _, err := r.Commit(ctx, path, "Add remote feature"); err != nil {
		t.Fatal(err)
	}
	if err := r.Push(ctx, branch); err != nil {
		t.Fatal(err)
	}
	if command(t, remote, "rev-parse", "refs/heads/"+branch) != command(t, path, "rev-parse", "HEAD") {
		t.Fatal("push failed")
	}
	if err := r.Merge(ctx, "T1", branch, "", "Merge remote feature"); err != nil {
		t.Fatal(err)
	}
	if command(t, remote, "rev-parse", "main") != command(t, r.Dir, "rev-parse", "main") {
		t.Fatal("main not pushed")
	}
}
func TestGh(t *testing.T) {
	ctx := context.Background()
	r := repository(t)
	_, branch := task(t, r)
	const prURL = "https://github.com/example/repo/pull/1"
	r.LookPath = func(string) (string, error) { return "gh", nil }
	var calls [][]string
	existing := true
	r.Exec = func(ctx context.Context, dir, name string, args ...string) (string, error) {
		if name != "gh" {
			return DefaultExec(ctx, dir, name, args...)
		}
		calls = append(calls, append([]string(nil), args...))
		switch args[1] {
		case "view":
			if existing {
				return prURL, nil
			}
			return "", errors.New("missing")
		case "create":
			want := []string{"pr", "create", "--base", "main", "--head", branch, "--title", "Title", "--body-file"}
			if !reflect.DeepEqual(args[:len(want)], want) {
				t.Errorf("create args: %v", args)
			}
			data, err := os.ReadFile(args[len(args)-1])
			if err != nil || string(data) != "Body" {
				t.Errorf("body: %q %v", data, err)
			}
			return "Created\n\n" + prURL + "\n", nil
		case "merge":
			return "", nil
		}
		return "", errors.New("unexpected command")
	}
	if got, err := r.OpenPR(ctx, branch, "Title", "Body"); err != nil || got != prURL || len(calls) != 1 {
		t.Fatalf("existing PR: %q %v", got, err)
	}
	existing = false
	if got, err := r.OpenPR(ctx, branch, "Title", "Body"); err != nil || got != prURL {
		t.Fatalf("create PR: %q %v", got, err)
	}
	if err := r.Merge(ctx, "T1", branch, prURL, "Merge"); err != nil {
		t.Fatal(err)
	}
	want := []string{"pr", "merge", prURL, "--merge", "--delete-branch=false"}
	if !reflect.DeepEqual(calls[len(calls)-1], want) {
		t.Fatalf("merge args: %v", calls)
	}
	r.LookPath = func(string) (string, error) { return "", errors.New("missing") }
	if _, err := r.OpenPR(ctx, branch, "Title", "Body"); err == nil {
		t.Fatal("missing gh accepted")
	}
}
func TestEnsureStartsFromRemoteBranch(t *testing.T) {
	ctx := context.Background()
	r := repository(t)
	remote := t.TempDir()
	command(t, remote, "init", "--bare")
	command(t, r.Dir, "remote", "add", "origin", remote)
	command(t, r.Dir, "push", "-u", "origin", "main")
	// someone else's branch, known here only as origin/feature
	command(t, r.Dir, "switch", "-c", "feature")
	write(t, filepath.Join(r.Dir, "feature.txt"), "feature\n")
	command(t, r.Dir, "add", "feature.txt")
	command(t, r.Dir, "commit", "-m", "feature")
	command(t, r.Dir, "push", "origin", "feature")
	command(t, r.Dir, "switch", "main")
	command(t, r.Dir, "branch", "-D", "feature")
	path, err := r.Ensure(ctx, "T5", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(path, "feature.txt")); err != nil {
		t.Fatal("worktree lost the commits of origin/feature:", err)
	}
}
