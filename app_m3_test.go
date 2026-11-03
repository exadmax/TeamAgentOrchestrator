package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/exadmax/TeamAgentOrchestrator/internal/gitflow"
	"github.com/exadmax/TeamAgentOrchestrator/internal/store"
	"github.com/exadmax/TeamAgentOrchestrator/internal/tasks"
	"github.com/exadmax/TeamAgentOrchestrator/internal/team"
)

func gitInit(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não está no PATH")
	}
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-b", "main"}, {"config", "user.name", "Teste"}, {"config", "user.email", "teste@example.com"},
		{"commit", "--allow-empty", "-m", "init"},
	} {
		if out, err := gitflow.DefaultExec(context.Background(), dir, "git", args...); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return dir
}

func TestOpenDetectsGitFlow(t *testing.T) {
	a := NewApp()
	if err := a.open(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if v := a.GitStatus(); v.On || a.gitRepo() != nil {
		t.Fatalf("plain folder has git flow: %+v", v)
	}
	if err := a.open(gitInit(t)); err != nil {
		t.Fatal(err)
	}
	v := a.GitStatus()
	if !v.On || v.Base != "main" || v.Prefix != "tao/" || v.Remote {
		t.Fatalf("git view = %+v", v)
	}
	if v, err := a.SetGit(team.Git{Off: true}); err != nil || v.On || a.gitRepo() != nil {
		t.Fatalf("SetGit off: %+v %v", v, err)
	}
	if err := a.open(a.dir); err != nil { // the setting is in tao.json
		t.Fatal(err)
	}
	if a.GitStatus().On {
		t.Fatal("git flow came back on after reopening")
	}
}

// The git part of a task without an agent: branch and worktree, commit, approval, local merge.
func TestTaskGitFlowLocalMerge(t *testing.T) {
	dir := gitInit(t)
	a := NewApp()
	if err := a.open(dir); err != nil {
		t.Fatal(err)
	}
	if err := a.AddTask(tasks.NewTask{Title: "Criar olá", Assignee: "engenheiro"}); err != nil {
		t.Fatal(err)
	}
	var tk tasks.Task
	_ = a.board(func(b *tasks.Board) error { tk, _ = b.Task("T001"); return nil })
	repo := a.gitRepo()
	wt, err := a.gitStart(repo, tk)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(wt), ".tao/worktrees/T001") {
		t.Fatalf("worktree = %s", wt)
	}
	if err := os.WriteFile(filepath.Join(wt, "ola.txt"), []byte("olá\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := a.board(func(b *tasks.Board) error { return b.Start("T001", "engenheiro", now) }); err != nil {
		t.Fatal(err)
	}
	a.gitSubmit(repo, a.store, "T001", "engenheiro", "criei ola.txt")
	st := a.store.State("T001")
	if st.Phases["submit"].Status != "ok" || st.Phases["pr"].Status != "skipped" {
		t.Fatalf("phases = submit %+v pr %+v", st.Phases["submit"], st.Phases["pr"])
	}
	if err := a.MergeTask("T001"); err == nil {
		t.Fatal("merged before the review")
	}
	if err := a.board(func(b *tasks.Board) error {
		if err := b.Submit("T001", "engenheiro", "ok", now); err != nil {
			return err
		}
		return b.Review("T001", "revisor", true, "", now)
	}); err != nil {
		t.Fatal(err)
	}
	if b := a.Board(); b.Tasks[0].Status != tasks.Review || !b.Tasks[0].Approved || len(b.Tasks[0].Phases) != len(store.Phases) {
		t.Fatalf("after approval: %+v", b.Tasks[0])
	}
	if err := a.MergeTask("T001"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ola.txt")); err != nil {
		t.Fatal("the work did not reach main:", err)
	}
	if b := a.Board(); b.Tasks[0].Status != tasks.Done {
		t.Fatalf("status = %s", b.Tasks[0].Status)
	}
}

// A model out of quota: with auto-switch the next free model is used, then the reserve agent.
func TestRouteSwitchesModelThenReserve(t *testing.T) {
	a := NewApp()
	if err := a.open(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	eng, _ := a.team.Get("engenheiro")
	rev, _ := a.team.Get("revisor")
	eng.Fallback = rev.ID
	if _, err := a.UpdateAgent(eng); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	eng, _ = a.team.Get("engenheiro")
	model := a.modelLocked(eng, "")
	a.mu.Unlock()
	if err := a.store.SetClock(eng.Client, model, time.Hour, "quota", "teste"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.agentFor("engenheiro", ""); err == nil || !strings.Contains(err.Error(), "troca automática") {
		t.Fatalf("without auto-switch: %v", err)
	}
	if err := a.SetAutoSwitch(true); err != nil {
		t.Fatal(err)
	}
	r, err := a.agentFor("engenheiro", "")
	if err != nil || r.agent.ID != "engenheiro" || r.model == model || r.note == "" {
		t.Fatalf("model switch: %+v %v", r, err)
	}
	for _, m := range a.Models().Clients {
		if m.ID == eng.Client {
			for _, x := range m.Models {
				_ = a.store.SetClock(eng.Client, x, time.Hour, "quota", "teste")
			}
		}
	}
	r, err = a.agentFor("engenheiro", "")
	if err != nil || r.agent.ID != rev.ID {
		t.Fatalf("reserve: %+v %v", r, err)
	}
	if err := a.SetModelOverride(eng.Client, "x-model", 0, "T009"); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	got := a.modelLocked(eng, "T009")
	a.mu.Unlock()
	if got != "x-model" {
		t.Fatalf("task override = %q", got)
	}
}
