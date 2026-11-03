package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/exadmax/TeamAgentOrchestrator/internal/failure"
	"github.com/exadmax/TeamAgentOrchestrator/internal/store"
	"github.com/exadmax/TeamAgentOrchestrator/internal/tasks"
)

// Who does things by hand in the task history.
const human = "diogo"

// SprintView is a sprint plus its progress.
type SprintView struct {
	tasks.Sprint
	Progress tasks.Progress `json:"progress"`
}

// PhaseView is one step of the task pipeline as the UI shows it.
type PhaseView struct {
	Phase    string `json:"phase"`
	Status   string `json:"status"`
	Agent    string `json:"agent,omitempty"`
	Model    string `json:"model,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// TaskView is a task plus names and its pipeline.
type TaskView struct {
	tasks.Task
	StatusLabel  string      `json:"status_label"`
	AssigneeName string      `json:"assignee_name"`
	Running      bool        `json:"running"`
	Phases       []PhaseView `json:"phases"`
}

// BoardView is the Sprints tab.
type BoardView struct {
	Sprints []SprintView   `json:"sprints"`
	Tasks   []TaskView     `json:"tasks"`
	Active  string         `json:"active"`
	Backlog tasks.Progress `json:"backlog"`
	Git     GitView        `json:"git"`
}

// noGitPhases is the pipeline of a project without the git flow: no submit, PR or merge.
var noGitPhases = []string{"start", "work", "review", "verdict"}

// board loads the task board, applies f and saves it, all under the board lock.
func (a *App) board(f func(b *tasks.Board) error) error {
	a.bmu.Lock()
	defer a.bmu.Unlock()
	a.mu.Lock()
	dir := a.dir
	a.mu.Unlock()
	if dir == "" {
		return errors.New("nenhum projeto aberto")
	}
	b, err := tasks.Load(dir)
	if err != nil {
		return err
	}
	if f == nil {
		return nil
	}
	if err := f(b); err != nil {
		return err
	}
	if err := b.Save(dir); err != nil {
		return err
	}
	go func() { a.emit("board", a.Board()) }() // Board() locks bmu: it must run outside the caller
	return nil
}

// Board returns the whole task board for the UI.
func (a *App) Board() BoardView {
	a.bmu.Lock()
	a.mu.Lock()
	dir, st, t, gv := a.dir, a.store, a.team, a.gitView
	running := map[string]bool{}
	for id := range a.taskRuns {
		running[id] = true
	}
	a.mu.Unlock()
	b, err := tasks.Load(dir)
	a.bmu.Unlock()
	if err != nil || t == nil {
		return BoardView{Sprints: []SprintView{}, Tasks: []TaskView{}}
	}
	v := BoardView{Backlog: b.Progress(""), Sprints: []SprintView{}, Tasks: []TaskView{}, Git: gv} // [] not null for the UI
	phases := noGitPhases
	if gv.On {
		phases = store.Phases
	}
	for _, sp := range b.Sprints {
		v.Sprints = append(v.Sprints, SprintView{Sprint: sp, Progress: b.Progress(sp.ID)})
		if sp.Status == tasks.Active {
			v.Active = sp.ID
		}
	}
	for _, tk := range b.Tasks {
		tv := TaskView{Task: tk, StatusLabel: tk.Status.Label(), Running: running[tk.ID]}
		if ag, ok := t.Get(tk.Assignee); ok {
			tv.AssigneeName = ag.Name
		}
		state := st.State(tk.ID)
		for _, p := range phases {
			pv := PhaseView{Phase: p}
			if rec := state.Phases[p]; rec != nil {
				pv.Status, pv.Agent, pv.Model, pv.Attempts, pv.Detail = rec.Status, rec.Agent, rec.Model, rec.Attempts, rec.Detail
				if pv.Status == "running" && !tv.Running {
					pv.Status = "failed" // the app closed mid-session: the run is gone, the task can be run again
				}
			}
			tv.Phases = append(tv.Phases, pv)
		}
		v.Tasks = append(v.Tasks, tv)
	}
	return v
}

// ---------------------------------------------------------------- sprints

func (a *App) NewSprint(title, goal string) error {
	return a.board(func(b *tasks.Board) error { _, err := b.NewSprint(title, goal, time.Now()); return err })
}

func (a *App) StartSprint(id string) error {
	return a.board(func(b *tasks.Board) error { return b.StartSprint(id, time.Now()) })
}

func (a *App) CloseSprint(id string, retro []string, carryTo string) error {
	return a.board(func(b *tasks.Board) error { return b.CloseSprint(id, retro, carryTo, time.Now()) })
}

// ---------------------------------------------------------------- manual task edits

func (a *App) AddTask(n tasks.NewTask) error {
	return a.board(func(b *tasks.Board) error { _, err := b.AddTask(n, human, time.Now()); return err })
}

func (a *App) UpdateTask(t tasks.Task) error {
	return a.board(func(b *tasks.Board) error { return b.UpdateTask(t, human, time.Now()) })
}

func (a *App) ApproveTask(id string) error {
	return a.board(func(b *tasks.Board) error { return b.Approve(id, human, time.Now()) })
}

func (a *App) ArchiveTask(id string) error {
	return a.board(func(b *tasks.Board) error { return b.Archive(id, human, time.Now()) })
}

func (a *App) ReopenTask(id, reason string) error {
	err := a.board(func(b *tasks.Board) error { return b.Reopen(id, human, reason, time.Now()) })
	if err == nil {
		a.mu.Lock()
		_ = a.store.ResetState(id)
		a.mu.Unlock()
	}
	return err
}

func (a *App) SetTaskPriority(id string, high bool) error {
	p := tasks.Normal
	if high {
		p = tasks.High
	}
	return a.board(func(b *tasks.Board) error { return b.SetPriority(id, p, human, time.Now()) })
}

// ---------------------------------------------------------------- agents working on tasks

// agentFor checks the agent and routes it: the result may be another model or the agent's reserve
// when the planned model is out of quota (see routeLocked).
func (a *App) agentFor(id, task string) (route, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.team == nil {
		return route{}, errors.New("nenhum projeto aberto")
	}
	ag, ok := a.team.Get(id)
	if !ok {
		return route{}, fmt.Errorf("agente %q não existe", id)
	}
	if ag.Paused {
		return route{}, fmt.Errorf("%s está pausado", ag.Name)
	}
	return a.routeLocked(ag, task)
}

// noteRoute tells the activity panel when a session does not run as planned.
func (a *App) noteRoute(planned string, r route) {
	if r.note != "" {
		a.push(Line{Agent: planned, Kind: "info", Text: r.note})
	}
}

// RunTask hands the task to its assignee. The result moves the task: done work goes to review,
// a final question blocks it, a failure stays in progress with the reason in the history.
func (a *App) RunTask(id string) error { return a.runTask(id, "", 0) }

// runTask runs the task once; switches counts the hops already made to another model or the reserve.
func (a *App) runTask(id, note string, switches int) error {
	var tk tasks.Task
	if err := a.board(func(b *tasks.Board) error {
		t, ok := b.Task(id)
		if !ok {
			return fmt.Errorf("tarefa %s não existe", id)
		}
		if t.Assignee == "" {
			return fmt.Errorf("%s não tem responsável", id)
		}
		if t.Status == tasks.Proposed {
			return fmt.Errorf("%s ainda é uma proposta: aprove antes", id)
		}
		tk = t
		return nil
	}); err != nil {
		return err
	}
	r, err := a.agentFor(tk.Assignee, id)
	if err != nil {
		return err
	}
	ag, model := r.agent, r.model
	a.noteRoute(tk.Assignee, r)
	a.mu.Lock()
	prompt := a.promptLocked(ag, taskPrompt(tk, note))
	st := a.store
	a.mu.Unlock()

	repo, workDir := a.gitRepo(), ""
	if repo != nil {
		if workDir, err = a.gitStart(repo, tk); err != nil {
			a.warn(id, st.Mark(id, "start", "failed", store.PhaseInfo{Agent: ag.ID, Detail: err.Error()}))
			return err
		}
	}
	if err := a.board(func(b *tasks.Board) error { return b.Start(id, ag.ID, time.Now()) }); err != nil {
		return err
	}
	a.warn(id, st.Mark(id, "start", "ok", store.PhaseInfo{Agent: ag.ID, Client: ag.Client, Detail: workDir}))
	a.warn(id, st.Mark(id, "work", "running", store.PhaseInfo{Agent: ag.ID, Client: ag.Client, Model: model}))
	a.setTaskRun(id, true)
	err = a.launch(ag.ID, ag.Client, model, prompt, fmt.Sprintf("[%s] %s", id, tk.Title), id, workDir, func(o Outcome) {
		a.setTaskRun(id, false)
		now := time.Now()
		switch {
		case o.Kind != "" && switches < maxSwitches && a.switchable(o.Kind):
			// the failure set a clock on this model: the next route picks another model or the reserve
			a.warn(id, st.Mark(id, "work", "failed", store.PhaseInfo{Agent: ag.ID, Client: ag.Client, Model: model, Detail: string(o.Kind)}))
			if err := a.runTask(id, note, switches+1); err != nil {
				a.warn(id, a.board(func(b *tasks.Board) error {
					return b.Ask(id, ag.ID, fmt.Sprintf("falhou (%s) e não deu para trocar: %v", o.Kind, err), now)
				}))
			}
		case o.Kind != "":
			a.warn(id, st.Mark(id, "work", "failed", store.PhaseInfo{Agent: ag.ID, Client: ag.Client, Model: model, Detail: string(o.Kind)}))
			a.warn(id, a.board(func(b *tasks.Board) error {
				return b.Ask(id, ag.ID, fmt.Sprintf("falhou (%s): %s", o.Kind, failure.Help(o.Kind)), now)
			}))
		case o.Question != "":
			a.warn(id, st.Mark(id, "work", "blocked", store.PhaseInfo{Agent: ag.ID, Client: ag.Client, Model: model, Detail: "pergunta"}))
			a.warn(id, a.board(func(b *tasks.Board) error { return b.Ask(id, ag.ID, o.Question, now) }))
		default:
			a.warn(id, st.Mark(id, "work", "ok", store.PhaseInfo{Agent: ag.ID, Client: ag.Client, Model: model}))
			if repo != nil {
				a.gitSubmit(repo, st, id, ag.ID, o.Final)
			}
			a.warn(id, a.board(func(b *tasks.Board) error { return b.Submit(id, ag.ID, oneLine(o.Final), now) }))
		}
	})
	if err != nil {
		a.setTaskRun(id, false)
		a.warn(id, st.Mark(id, "work", "failed", store.PhaseInfo{Agent: ag.ID, Client: ag.Client, Detail: err.Error()}))
	}
	return err
}

// AnswerTask answers the agent's question (or the failure) and puts it back to work.
func (a *App) AnswerTask(id, answer string) error {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return errors.New("escreva a resposta")
	}
	var question string
	if err := a.board(func(b *tasks.Board) error {
		t, _ := b.Task(id)
		question = t.Question
		return b.Answer(id, human, answer, time.Now())
	}); err != nil {
		return err
	}
	note := "Resposta do Diogo: " + answer
	if question != "" {
		note = fmt.Sprintf("Você perguntou: %q. %s", question, note)
	}
	return a.runTask(id, note, 0)
}

// DelegateTask passes the task to another agent and starts it there.
func (a *App) DelegateTask(id, to, note string) error {
	if _, err := a.agentFor(to, id); err != nil {
		return err
	}
	if err := a.board(func(b *tasks.Board) error { return b.Delegate(id, to, human, time.Now()) }); err != nil {
		return err
	}
	if strings.TrimSpace(note) != "" {
		note = "Observação de quem delegou: " + note
	}
	return a.runTask(id, note, 0)
}

// ReviewTask asks a reviewer (the task's, or one picked from another client) to approve or not.
func (a *App) ReviewTask(id string) error {
	var tk tasks.Task
	if err := a.board(func(b *tasks.Board) error {
		t, ok := b.Task(id)
		if !ok {
			return fmt.Errorf("tarefa %s não existe", id)
		}
		if t.Status != tasks.Review {
			return fmt.Errorf("%s não está em revisão", id)
		}
		tk = t
		return nil
	}); err != nil {
		return err
	}
	// the reviewer must be on another client than whoever did the work (it may have been the reserve)
	a.mu.Lock()
	st0 := a.store
	a.mu.Unlock()
	worker := ""
	if ph := st0.State(id).Phases["work"]; ph != nil {
		worker = ph.Client
	}
	if worker == "" {
		worker = a.clientOf(tk.Assignee)
	}
	revID := tk.Reviewer
	if revID != "" && a.clientOf(revID) == worker {
		a.push(Line{Agent: id, Kind: "info", Text: fmt.Sprintf("%s usa o mesmo cliente de quem fez a tarefa (%s): escolhendo outro revisor", revID, worker)})
		revID = ""
	}
	if revID == "" {
		revID = a.pickReviewer(tk.Assignee, worker)
	}
	if revID == "" {
		return errors.New("nenhum revisor disponível: precisa de um agente ativo em outro cliente")
	}
	r, err := a.agentFor(revID, id)
	if err != nil {
		return err
	}
	if r.agent.Client == worker {
		return fmt.Errorf("o revisor %s caiu no mesmo cliente de quem fez a tarefa (%s)", r.agent.Name, worker)
	}
	rev, model := r.agent, r.model
	a.noteRoute(revID, r)
	a.mu.Lock()
	prompt := a.promptLocked(rev, reviewPrompt(tk))
	st := a.store
	a.mu.Unlock()
	repo, workDir := a.gitRepo(), ""
	if repo != nil && tk.Branch != "" {
		if workDir, err = a.gitStart(repo, tk); err != nil {
			return err
		}
		prompt += a.reviewContext(repo, tk)
	}
	a.warn(id, st.Mark(id, "review", "running", store.PhaseInfo{Agent: rev.ID, Client: rev.Client, Model: model}))
	a.setTaskRun(id, true)
	err = a.launch(rev.ID, rev.Client, model, prompt, fmt.Sprintf("[%s] revisão", id), id, workDir, func(o Outcome) {
		a.setTaskRun(id, false)
		now := time.Now()
		if o.Kind != "" {
			a.warn(id, st.Mark(id, "review", "failed", store.PhaseInfo{Agent: rev.ID, Client: rev.Client, Model: model, Detail: string(o.Kind)}))
			return
		}
		approve, note, ok := parseVerdict(o.Final)
		if !ok {
			a.warn(id, st.Mark(id, "review", "blocked", store.PhaseInfo{Agent: rev.ID, Client: rev.Client, Detail: "sem parecer"}))
			a.warn(id, a.board(func(b *tasks.Board) error {
				return b.Ask(id, rev.ID, "o revisor não deu um parecer claro: "+oneLine(o.Final), now)
			}))
			return
		}
		verdict := "changes"
		if approve {
			verdict = "approve"
		}
		a.warn(id, st.Mark(id, "review", "ok", store.PhaseInfo{Agent: rev.ID, Client: rev.Client, Model: model}))
		a.warn(id, st.Mark(id, "verdict", "ok", store.PhaseInfo{Agent: rev.ID, Verdict: verdict}))
		a.warn(id, a.board(func(b *tasks.Board) error { return b.Review(id, rev.ID, approve, note, now) }))
		if !approve {
			a.warn(id, st.Mark(id, "work", "pending", store.PhaseInfo{Detail: "revisão pediu mudanças"}))
		}
	})
	if err != nil {
		a.setTaskRun(id, false)
	}
	return err
}

// clientOf returns the client of an agent ("" if unknown).
func (a *App) clientOf(id string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.team == nil {
		return ""
	}
	ag, _ := a.team.Get(id)
	return ag.Client
}

// pickReviewer prefers an active reviewer-emblem agent on another client, then anyone on another client.
func (a *App) pickReviewer(author, worker string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.team == nil {
		return ""
	}
	best := ""
	for _, ag := range a.team.Agents {
		if ag.Paused || ag.ID == author || ag.Client == worker {
			continue
		}
		if ag.Emblem == "revisao" {
			return ag.ID
		}
		if best == "" {
			best = ag.ID
		}
	}
	return best
}

// GenerateTasks asks an agent to break an objective into tasks; they arrive as proposals.
func (a *App) GenerateTasks(agentID, objective, sprint string) error {
	objective = strings.TrimSpace(objective)
	if objective == "" {
		return errors.New("escreva o objetivo")
	}
	r, err := a.agentFor(agentID, "")
	if err != nil {
		return err
	}
	ag, model := r.agent, r.model
	a.noteRoute(agentID, r)
	a.mu.Lock()
	var roster []string
	var ids []string
	for _, x := range a.team.Agents {
		roster = append(roster, fmt.Sprintf("- %s: %s (%s)", x.ID, x.Name, x.Role))
		ids = append(ids, x.ID)
	}
	prompt := a.promptLocked(ag, fmt.Sprintf(`Quebre o objetivo abaixo em 3 a 8 atividades pequenas e verificáveis, e atribua cada uma ao agente mais adequado do time.
Objetivo: %s

Time (id: nome (função)):
%s

Não execute nada e não altere arquivos: só planeje. Responda SOMENTE com um array JSON, sem texto antes ou depois:
[{"title": "atividade curta", "done": "como saber que terminou", "assignee": "<id do agente>", "priority": "normal"}]`,
		objective, strings.Join(roster, "\n")))
	a.mu.Unlock()
	return a.launch(ag.ID, ag.Client, model, prompt, "planejar: "+objective, "", "", func(o Outcome) {
		if o.Kind != "" {
			return
		}
		list, err := tasks.ParseGenerated(o.Final, ids)
		if err != nil {
			a.push(Line{Agent: ag.ID, Kind: "error", Error: true, Text: "não consegui ler as atividades geradas: " + err.Error()})
			return
		}
		a.warn(ag.ID, a.board(func(b *tasks.Board) error {
			for _, n := range list {
				n.Sprint, n.Origin, n.Proposed = sprint, "agent:"+ag.ID, true
				if _, err := b.AddTask(n, ag.ID, time.Now()); err != nil {
					return err
				}
			}
			return nil
		}))
		a.push(Line{Agent: ag.ID, Kind: "end", Text: fmt.Sprintf("%d atividades propostas na aba Sprints", len(list))})
	})
}

// warn surfaces a failed state write (it used to be silently dropped) in "Atividade Recente".
func (a *App) warn(task string, err error) {
	if err != nil {
		a.push(Line{Agent: task, Kind: "error", Error: true, Text: "não consegui gravar o estado: " + err.Error()})
	}
}

func (a *App) setTaskRun(id string, on bool) {
	a.mu.Lock()
	if on {
		a.taskRuns[id] = true
	} else {
		delete(a.taskRuns, id)
	}
	a.mu.Unlock()
	go func() { a.emit("board", a.Board()) }() // Board() locks bmu: it must run outside the caller
}

// ---------------------------------------------------------------- prompts and parsing

func taskPrompt(t tasks.Task, note string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Tarefa %s: %s\nPronto quando: %s\n", t.ID, t.Title, orDash(t.Done))
	if t.Summary != "" {
		fmt.Fprintf(&b, "Resultado anterior: %s\n", t.Summary)
	}
	for _, v := range t.Verdicts {
		if strings.HasSuffix(v, ":changes") {
			fmt.Fprintf(&b, "A revisão anterior pediu mudanças; veja o histórico abaixo.\n")
			break
		}
	}
	if n := len(t.History); n > 0 {
		b.WriteString("Histórico recente:\n")
		for _, e := range t.History[max(0, n-5):] {
			fmt.Fprintf(&b, "- %s (%s)\n", e.What, e.By)
		}
	}
	if note != "" {
		b.WriteString(note + "\n")
	}
	b.WriteString("\nFaça a tarefa. Ao terminar, escreva por último uma linha com o resumo do que foi feito. " +
		"Se precisar de uma decisão do Diogo para continuar, pare e termine a resposta com UMA pergunta (terminada em ?).")
	return b.String()
}

func reviewPrompt(t tasks.Task) string {
	return fmt.Sprintf(`Revise a tarefa %s feita por outro agente: %s
Pronto quando: %s
Resumo de quem fez: %s

Confira nos arquivos do projeto se o critério de pronto foi cumprido. Não altere nada.
Responda com uma linha que comece por APROVADO ou por MUDANÇAS: seguida do que precisa mudar.`,
		t.ID, t.Title, orDash(t.Done), orDash(t.Summary))
}

// verdictRe tolerates markdown around the keyword: "**APROVADO** - ok", "> MUDANÇAS: ...".
var verdictRe = regexp.MustCompile(`(?im)^[\s*_>#-]*(APROVADO|MUDAN[ÇC]AS)[\s*_:\-–]*(.*)$`)

// parseVerdict reads the reviewer's answer: the last APROVADO / MUDANÇAS line wins.
func parseVerdict(text string) (approve bool, note string, ok bool) {
	m := verdictRe.FindAllStringSubmatch(text, -1)
	if len(m) == 0 {
		return false, "", false
	}
	last := m[len(m)-1]
	return strings.EqualFold(last[1], "APROVADO"), strings.TrimSpace(last[2]), true
}

func oneLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	out := strings.TrimSpace(lines[len(lines)-1])
	if r := []rune(out); len(r) > 300 {
		out = string(r[:299]) + "…"
	}
	return out
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
