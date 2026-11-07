package main

import (
	"context"
	"fmt"
	"github.com/exadmax/TeamAgentOrchestrator/internal/i18n"
	"strings"
	"time"

	"github.com/exadmax/TeamAgentOrchestrator/internal/gitflow"
	"github.com/exadmax/TeamAgentOrchestrator/internal/importer"
	"github.com/exadmax/TeamAgentOrchestrator/internal/store"
	"github.com/exadmax/TeamAgentOrchestrator/internal/tasks"
	"github.com/exadmax/TeamAgentOrchestrator/internal/team"
)

// gitTimeout bounds one git/gh step (a push or a PR can hang on the network).
const gitTimeout = 3 * time.Minute

// GitView is the git flow of the open project, as the Sprints tab shows it.
type GitView struct {
	On     bool   `json:"on"`
	Base   string `json:"base"`
	Prefix string `json:"prefix"`
	Remote bool   `json:"remote"`
	Gh     bool   `json:"gh"`
	Off    bool   `json:"off"`    // the project is a git repo but the flow was turned off
	Reason string `json:"reason"` // why the flow is off
}

// gitRepoFor builds the git flow of dir, or nil when dir is not a git repo or the flow is off.
func gitRepoFor(dir string, cfg team.Git) (*gitflow.Repo, GitView) {
	r := gitflow.Open(dir)
	if cfg.Prefix != "" {
		r.Prefix = cfg.Prefix
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	if !r.IsRepo(ctx) {
		return nil, GitView{Reason: i18n.T("git.not_repo")}
	}
	r.Base = cfg.Base
	if r.Base == "" {
		r.Base = r.DefaultBase(ctx)
	}
	v := GitView{On: !cfg.Off, Off: cfg.Off, Base: r.Base, Prefix: r.Prefix, Remote: r.HasRemote(ctx), Gh: r.HasGh()}
	if cfg.Off {
		v.Reason = i18n.T("git.turned_off")
		return nil, v
	}
	return r, v
}

// GitStatus returns the git flow of the open project.
func (a *App) GitStatus() GitView {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.gitView
}

// SetGit changes the git flow settings (saved in tao.json) and reloads it.
func (a *App) SetGit(cfg team.Git) (GitView, error) {
	cfg.Base, cfg.Prefix = strings.TrimSpace(cfg.Base), strings.TrimSpace(cfg.Prefix)
	if _, err := a.mutate(func(t *team.Team) error { t.Git = cfg; return nil }); err != nil {
		return a.GitStatus(), err
	}
	a.mu.Lock()
	dir := a.dir
	a.mu.Unlock()
	repo, view := gitRepoFor(dir, cfg)
	a.mu.Lock()
	a.repo, a.gitView = repo, view
	a.mu.Unlock()
	go func() { a.emit("board", a.Board()) }()
	return view, nil
}

func (a *App) gitRepo() *gitflow.Repo {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.repo
}

// gitStart gives the task its branch and worktree and returns the folder the agent works in.
func (a *App) gitStart(repo *gitflow.Repo, tk tasks.Task) (string, error) {
	branch := tk.Branch
	if branch == "" {
		branch = gitflow.BranchName(repo.Prefix, tk.ID, tk.Title)
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	if cur, err := gitflow.DefaultExec(ctx, repo.Dir, "git", "rev-parse", "--abbrev-ref", "HEAD"); err == nil && cur == branch {
		// the branch is checked out in the project folder itself (an imported card in progress):
		// git refuses a second worktree for it, so the agent works right there
		return repo.Dir, nil
	}
	wt, err := repo.Ensure(ctx, tk.ID, branch)
	if err != nil {
		return "", i18n.Err("git.prepare_failed", branch, err)
	}
	if branch != tk.Branch {
		if err := a.board(func(b *tasks.Board) error { return b.SetGit(tk.ID, branch, "", human, time.Now()) }); err != nil {
			return "", err
		}
	}
	return wt, nil
}

// gitSubmit commits what the agent did, pushes the branch and opens (or finds) the PR.
// Each step is a phase of the task; a failed push or PR does not stop the review.
func (a *App) gitSubmit(repo *gitflow.Repo, st *store.Store, id, by string, final string) {
	var tk tasks.Task
	if err := a.board(func(b *tasks.Board) error { tk, _ = b.Task(id); return nil }); err != nil {
		a.warn(id, err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	msg := fmt.Sprintf("%s: %s\n\n%s", tk.ID, tk.Title, oneLine(final))
	committed, err := repo.Commit(ctx, repo.WorktreePath(id), msg)
	if err != nil {
		a.warn(id, st.Mark(id, "submit", "failed", store.PhaseInfo{Agent: by, Detail: err.Error()}))
		a.push(Line{Agent: id, Kind: "error", Error: true, Text: "commit falhou: " + err.Error()})
		return
	}
	ahead, _ := repo.Ahead(ctx, tk.Branch)
	detail := i18n.T("git.commits_on", ahead, tk.Branch)
	if !committed && ahead == 0 {
		detail = i18n.T("git.no_changes")
	}
	a.warn(id, st.Mark(id, "submit", "ok", store.PhaseInfo{Agent: by, Detail: detail}))

	switch {
	case ahead == 0:
		a.warn(id, st.Mark(id, "pr", "skipped", store.PhaseInfo{Detail: i18n.T("git.nothing_to_push")}))
		return
	case !repo.HasRemote(ctx) || !repo.HasGh():
		a.warn(id, st.Mark(id, "pr", "skipped", store.PhaseInfo{Detail: i18n.T("git.no_remote_or_gh")}))
		return
	case !hostedRemote(ctx, repo):
		// a folder or bare repo as origin: push it, but there is no PR to open
		if err := repo.Push(ctx, tk.Branch); err != nil {
			a.warn(id, st.Mark(id, "pr", "failed", store.PhaseInfo{Detail: "push: " + err.Error()}))
			return
		}
		a.warn(id, st.Mark(id, "pr", "skipped", store.PhaseInfo{Detail: i18n.T("git.local_origin")}))
		return
	}
	if err := repo.Push(ctx, tk.Branch); err != nil {
		a.warn(id, st.Mark(id, "pr", "failed", store.PhaseInfo{Detail: "push: " + err.Error()}))
		return
	}
	body := i18n.T("git.pr_body",
		tk.ID, tk.Title, orDash(tk.Done), orDash(oneLine(final)))
	url, err := repo.OpenPR(ctx, tk.Branch, tk.ID+": "+tk.Title, body)
	if err != nil {
		a.warn(id, st.Mark(id, "pr", "failed", store.PhaseInfo{Detail: err.Error()}))
		return
	}
	a.warn(id, st.Mark(id, "pr", "ok", store.PhaseInfo{Detail: url}))
	a.warn(id, a.board(func(b *tasks.Board) error { return b.SetGit(id, "", url, human, time.Now()) }))
}

// hostedRemote reports whether origin is a server (https, ssh) rather than a folder, where gh has no PR to open.
func hostedRemote(ctx context.Context, repo *gitflow.Repo) bool {
	url, err := gitflow.DefaultExec(ctx, repo.Dir, "git", "remote", "get-url", "origin")
	return err == nil && (strings.Contains(url, "://") || strings.Contains(url, "@"))
}

// MergeTask integrates an approved task: the PR on GitHub when there is one, else a local merge.
func (a *App) MergeTask(id string) error {
	repo := a.gitRepo()
	if repo == nil {
		return i18n.Err("git.flow_off")
	}
	var tk tasks.Task
	if err := a.board(func(b *tasks.Board) error {
		t, ok := b.Task(id)
		switch {
		case !ok:
			return i18n.Err("app.task_not_found", id)
		case !t.Approved:
			return i18n.Err("git.not_approved", id)
		case t.Branch == "":
			return i18n.Err("git.no_branch", id)
		}
		tk = t
		return nil
	}); err != nil {
		return err
	}
	a.mu.Lock()
	st := a.store
	a.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	if err := repo.Merge(ctx, id, tk.Branch, tk.PR, fmt.Sprintf("Merge %s: %s", id, tk.Title)); err != nil {
		a.warn(id, st.Mark(id, "merge", "failed", store.PhaseInfo{Detail: err.Error()}))
		go func() { a.emit("board", a.Board()) }()
		return err
	}
	a.warn(id, st.Mark(id, "merge", "ok", store.PhaseInfo{Detail: i18n.T("git.merged_into", repo.Base)}))
	a.push(Line{Agent: id, Kind: "end", Text: i18n.T("git.merged_line", id, repo.Base)})
	if err := a.board(func(b *tasks.Board) error { return b.Merged(id, human, time.Now()) }); err != nil {
		return err
	}
	a.kick() // the card is closed: the flow hands the next one over
	return nil
}

// reviewContext tells the reviewer where the change is: the diff of the task branch.
func (a *App) reviewContext(repo *gitflow.Repo, tk tasks.Task) string {
	if repo == nil || tk.Branch == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	stat, err := repo.DiffStat(ctx, tk.Branch)
	if err != nil || strings.TrimSpace(stat) == "" {
		stat = i18n.T("git.no_diff", repo.Base)
	}
	return i18n.T("git.review_context",
		tk.Branch, repo.Base, tk.Branch, stat)
}

// ---------------------------------------------------------------- importing a cards project

// ImportInfo says whether the open project can be imported from cards (docs/equipe/owners.json).
type ImportInfo struct {
	Available bool `json:"available"`
	HasTasks  bool `json:"has_tasks"` // importing replaces the current tao.json and tao.tasks.json
}

func (a *App) CanImport() ImportInfo {
	a.mu.Lock()
	dir := a.dir
	a.mu.Unlock()
	info := ImportInfo{Available: dir != "" && importer.Detect(dir)}
	if b, err := tasks.Load(dir); err == nil {
		info.HasTasks = len(b.Tasks) > 0
	}
	return info
}

// ImportCards replaces the team and the board with the roles and cards of the project.
func (a *App) ImportCards() (importer.Report, error) {
	a.mu.Lock()
	dir, busy := a.dir, len(a.runs) > 0
	name := ""
	if a.team != nil {
		name = a.team.Name
	}
	a.mu.Unlock()
	if busy {
		return importer.Report{}, i18n.Err("import.agents_busy")
	}
	t, b, rep, err := importer.Import(dir, name, time.Now())
	if err != nil {
		return rep, err
	}
	a.mu.Lock()
	if a.team != nil {
		t.Git = a.team.Git // keep the project's git settings
	}
	a.mu.Unlock()
	a.bmu.Lock()
	err = b.Save(dir)
	a.bmu.Unlock()
	if err != nil {
		return rep, err
	}
	if err := t.Save(dir); err != nil {
		return rep, err
	}
	if err := a.open(dir); err != nil {
		return rep, err
	}
	a.push(Line{Agent: "importer", Kind: "end", Text: i18n.T("import.done", rep.Agents, rep.Tasks, rep.Sprints)})
	a.emit("snapshot", a.Snapshot())
	go func() { a.emit("board", a.Board()) }()
	return rep, nil
}
