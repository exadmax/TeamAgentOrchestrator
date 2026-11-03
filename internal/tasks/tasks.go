// Package tasks gerencia o quadro versionável de tarefas e sprints.
package tasks

import (
	"encoding/json"
	"fmt"
	"github.com/exadmax/TeamAgentOrchestrator/internal/fsx"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const FileName = "tao.tasks.json"
const historyLimit = 50

type Status string

const (
	Proposed   Status = "proposed"
	Ready      Status = "ready"
	InProgress Status = "in_progress"
	Review     Status = "review"
	Done       Status = "done"
)

var Statuses = []Status{Proposed, Ready, InProgress, Review, Done}

func (s Status) Label() string {
	switch s {
	case Proposed:
		return "Proposta"
	case Ready:
		return "A fazer"
	case InProgress:
		return "Fazendo"
	case Review:
		return "Em revisão"
	case Done:
		return "Concluída"
	}
	return string(s)
}

type Priority string

const (
	Normal Priority = "normal"
	High   Priority = "high"
)

type Event struct {
	At   time.Time `json:"at"`
	By   string    `json:"by"`
	What string    `json:"what"`
}
type Task struct {
	Branch   string    `json:"branch,omitempty"`
	PR       string    `json:"pr,omitempty"`
	Approved bool      `json:"approved,omitempty"`
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Done     string    `json:"done"`
	Assignee string    `json:"assignee"`
	Reviewer string    `json:"reviewer,omitempty"`
	Sprint   string    `json:"sprint,omitempty"`
	Status   Status    `json:"status"`
	Priority Priority  `json:"priority"`
	Origin   string    `json:"origin"`
	Question string    `json:"question,omitempty"`
	Summary  string    `json:"summary,omitempty"`
	Verdicts []string  `json:"verdicts,omitempty"`
	Archived bool      `json:"archived,omitempty"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
	History  []Event   `json:"history,omitempty"`
}

func (t Task) Blocked() bool { return t.Question != "" }

type SprintStatus string

const (
	Planned SprintStatus = "planned"
	Active  SprintStatus = "active"
	Closed  SprintStatus = "closed"
)

type Sprint struct {
	ID      string       `json:"id"`
	Title   string       `json:"title"`
	Goal    string       `json:"goal"`
	Status  SprintStatus `json:"status"`
	Created time.Time    `json:"created"`
	Started *time.Time   `json:"started,omitempty"`
	Closed  *time.Time   `json:"closed,omitempty"`
	Retro   []string     `json:"retro,omitempty"`
}
type Board struct {
	Sprints []Sprint `json:"sprints"`
	Tasks   []Task   `json:"tasks"`
}

func Load(dir string) (*Board, error) {
	b := &Board{Sprints: []Sprint{}, Tasks: []Task{}}
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if os.IsNotExist(err) {
		return b, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ler quadro: %w", err)
	}
	if err = json.Unmarshal(data, b); err != nil {
		return nil, fmt.Errorf("quadro inválido: %w", err)
	}
	return b, nil
}
func (b *Board) Save(dir string) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("codificar quadro: %w", err)
	}
	f, err := os.CreateTemp(dir, ".tao.tasks-*")
	if err != nil {
		return fmt.Errorf("criar temporário: %w", err)
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(data, '\n')); err != nil {
		f.Close()
		return fmt.Errorf("gravar quadro: %w", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("fechar temporário: %w", err)
	}
	if err = fsx.Replace(f.Name(), filepath.Join(dir, FileName)); err != nil {
		return fmt.Errorf("substituir quadro: %w", err)
	}
	return nil
}
func nextID(ids []string, prefix string, width int) string {
	max := 0
	for _, id := range ids {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(id, prefix)
		if suffix == "" || strings.IndexFunc(suffix, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			continue
		}
		n, err := strconv.Atoi(suffix)
		if err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("%s%0*d", prefix, width, max+1)
}
func (b *Board) NewSprint(title, goal string, now time.Time) (Sprint, error) {
	if strings.TrimSpace(title) == "" {
		return Sprint{}, fmt.Errorf("título da sprint é obrigatório")
	}
	ids := []string{}
	for _, s := range b.Sprints {
		ids = append(ids, s.ID)
	}
	s := Sprint{ID: nextID(ids, "S", 1), Title: title, Goal: goal, Status: Planned, Created: now}
	b.Sprints = append(b.Sprints, s)
	return s, nil
}
func (b *Board) sprintIndex(id string) (int, error) {
	for i := range b.Sprints {
		if b.Sprints[i].ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("sprint %s não existe", id)
}
func (b *Board) Sprint(id string) (Sprint, bool) {
	i, e := b.sprintIndex(id)
	if e != nil {
		return Sprint{}, false
	}
	s := b.Sprints[i]
	s.Retro = append([]string(nil), s.Retro...)
	if s.Started != nil {
		v := *s.Started
		s.Started = &v
	}
	if s.Closed != nil {
		v := *s.Closed
		s.Closed = &v
	}
	return s, true
}
func (b *Board) ActiveSprint() (Sprint, bool) {
	for _, s := range b.Sprints {
		if s.Status == Active {
			return b.Sprint(s.ID)
		}
	}
	return Sprint{}, false
}
func (b *Board) StartSprint(id string, now time.Time) error {
	i, e := b.sprintIndex(id)
	if e != nil {
		return e
	}
	if b.Sprints[i].Status != Planned {
		return fmt.Errorf("sprint %s precisa estar planejada", id)
	}
	if s, ok := b.ActiveSprint(); ok {
		return fmt.Errorf("já existe sprint ativa: %s", s.ID)
	}
	b.Sprints[i].Status = Active
	b.Sprints[i].Started = &now
	return nil
}
func (b *Board) validSprint(id string) error {
	if id == "" {
		return nil
	}
	i, e := b.sprintIndex(id)
	if e != nil {
		return e
	}
	if b.Sprints[i].Status == Closed {
		return fmt.Errorf("sprint %s está fechada", id)
	}
	return nil
}
func (b *Board) CloseSprint(id string, retro []string, carryTo string, now time.Time) error {
	i, e := b.sprintIndex(id)
	if e != nil {
		return e
	}
	if b.Sprints[i].Status != Active {
		return fmt.Errorf("sprint %s precisa estar ativa", id)
	}
	if carryTo == id {
		return fmt.Errorf("destino não pode ser a própria sprint")
	}
	if e = b.validSprint(carryTo); e != nil {
		return e
	}
	for j := range b.Tasks {
		t := &b.Tasks[j]
		if t.Sprint == id && t.Status != Done && !t.Archived {
			t.Sprint = carryTo
			t.Updated = now
		}
	}
	b.Sprints[i].Status = Closed
	b.Sprints[i].Closed = &now
	b.Sprints[i].Retro = append([]string(nil), retro...)
	return nil
}

type NewTask struct {
	Title, Done, Assignee, Sprint, Origin string
	Priority                              Priority
	Proposed                              bool
}

func validPriority(p Priority) error {
	if p != Normal && p != High {
		return fmt.Errorf("prioridade inválida: %s", p)
	}
	return nil
}
func (b *Board) AddTask(n NewTask, by string, now time.Time) (Task, error) {
	if strings.TrimSpace(n.Title) == "" {
		return Task{}, fmt.Errorf("título da tarefa é obrigatório")
	}
	if e := b.validSprint(n.Sprint); e != nil {
		return Task{}, e
	}
	if n.Priority == "" {
		n.Priority = Normal
	}
	if e := validPriority(n.Priority); e != nil {
		return Task{}, e
	}
	if n.Origin == "" {
		n.Origin = "manual"
	}
	ids := []string{}
	for _, t := range b.Tasks {
		ids = append(ids, t.ID)
	}
	t := Task{ID: nextID(ids, "T", 3), Title: n.Title, Done: n.Done, Assignee: n.Assignee, Sprint: n.Sprint, Origin: n.Origin, Priority: n.Priority, Status: Ready, Created: now}
	if n.Proposed {
		t.Status = Proposed
	}
	record(&t, by, "criada", now)
	b.Tasks = append(b.Tasks, t)
	return cloneTask(t), nil
}
func cloneTask(t Task) Task {
	t.History = append([]Event(nil), t.History...)
	t.Verdicts = append([]string(nil), t.Verdicts...)
	return t
}
func (b *Board) taskIndex(id string) (int, error) {
	for i := range b.Tasks {
		if b.Tasks[i].ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("tarefa %s não existe", id)
}
func (b *Board) Task(id string) (Task, bool) {
	i, e := b.taskIndex(id)
	if e != nil {
		return Task{}, false
	}
	return cloneTask(b.Tasks[i]), true
}
func record(t *Task, by, what string, now time.Time) {
	t.Updated = now
	t.History = append(t.History, Event{At: now, By: by, What: what})
	if len(t.History) > historyLimit {
		t.History = append([]Event(nil), t.History[len(t.History)-historyLimit:]...)
	}
}
func (b *Board) mutate(id, by string, now time.Time, fn func(*Task) (string, error)) error {
	i, e := b.taskIndex(id)
	if e != nil {
		return e
	}
	t := cloneTask(b.Tasks[i])
	what, e := fn(&t)
	if e != nil {
		return e
	}
	record(&t, by, what, now)
	b.Tasks[i] = t
	return nil
}
func require(t *Task, states ...Status) error {
	if !t.Archived {
		for _, s := range states {
			if t.Status == s {
				return nil
			}
		}
	}
	return fmt.Errorf("tarefa %s não permite esta operação no estado atual", t.ID)
}
func (b *Board) UpdateTask(t Task, by string, now time.Time) error {
	return b.mutate(t.ID, by, now, func(old *Task) (string, error) {
		if strings.TrimSpace(t.Title) == "" {
			return "", fmt.Errorf("título da tarefa é obrigatório")
		}
		if e := validPriority(t.Priority); e != nil {
			return "", e
		}
		if e := b.validSprint(t.Sprint); e != nil {
			return "", e
		}
		old.Title = t.Title
		old.Done = t.Done
		old.Assignee = t.Assignee
		old.Reviewer = t.Reviewer
		old.Sprint = t.Sprint
		old.Priority = t.Priority
		return "editada", nil
	})
}
func (b *Board) Approve(id, by string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) {
		if e := require(t, Proposed); e != nil {
			return "", e
		}
		t.Status = Ready
		return "aprovada", nil
	})
}
func (b *Board) Start(id, by string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) {
		if e := require(t, Ready, InProgress); e != nil {
			return "", e
		}
		t.Approved = false
		t.Status = InProgress
		t.Question = ""
		return "começou", nil
	})
}
func (b *Board) Ask(id, by, question string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) {
		if e := require(t, Proposed, Ready, InProgress, Review); e != nil {
			return "", e
		}
		if strings.TrimSpace(question) == "" {
			return "", fmt.Errorf("pergunta é obrigatória")
		}
		t.Question = question
		return "pergunta: " + question, nil
	})
}
func (b *Board) Answer(id, by, answer string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) {
		if e := require(t, Proposed, Ready, InProgress, Review); e != nil {
			return "", e
		}
		if !t.Blocked() {
			return "", fmt.Errorf("tarefa %s não tem pergunta pendente", id)
		}
		t.Question = ""
		return "resposta: " + answer, nil
	})
}
func (b *Board) Submit(id, by, summary string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) {
		if e := require(t, InProgress); e != nil {
			return "", e
		}
		t.Status = Review
		t.Summary = summary
		return "enviada para revisão", nil
	})
}
func (b *Board) Review(id, by string, approve bool, note string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) {
		if e := require(t, Review); e != nil {
			return "", e
		}
		verdict := "changes"
		what := "revisão: alterações solicitadas"
		t.Approved = false
		t.Status = InProgress
		t.Question = ""
		if approve {
			verdict = "approve"
			what = "revisão: aprovada"
			t.Status = Done
			if t.Branch != "" {
				t.Status = Review
				t.Approved = true
				what = "revisão: aprovada; falta o merge"
			}
		}
		t.Verdicts = append(t.Verdicts, by+":"+verdict)
		if note != "" {
			what += ": " + note
		}
		return what, nil
	})
}
func (b *Board) Delegate(id, to, by string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) {
		if e := require(t, Proposed, Ready, InProgress, Review); e != nil {
			return "", e
		}
		t.Assignee = to
		return "delegada para " + to, nil
	})
}
func (b *Board) Archive(id, by string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) { t.Archived = true; return "arquivada", nil })
}
func (b *Board) Reopen(id, by, reason string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) {
		if t.Status != Done {
			return "", fmt.Errorf("somente tarefa concluída pode ser reaberta")
		}
		t.Status = Ready
		t.Archived = false
		t.Approved = false
		t.Verdicts = nil
		return "reaberta: " + reason, nil
	})
}
func (b *Board) SetPriority(id string, p Priority, by string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) {
		if e := validPriority(p); e != nil {
			return "", e
		}
		t.Priority = p
		return "prioridade alterada", nil
	})
}

type Progress struct {
	Done    int      `json:"done"`
	Total   int      `json:"total"`
	Pct     int      `json:"pct"`
	Blocked []string `json:"blocked"`
}

func (b *Board) Progress(sprint string) Progress {
	p := Progress{}
	for _, t := range b.Tasks {
		if t.Archived || t.Sprint != sprint {
			continue
		}
		p.Total++
		if t.Status == Done {
			p.Done++
		}
		if t.Blocked() {
			p.Blocked = append(p.Blocked, t.ID)
		}
	}
	if p.Total > 0 {
		p.Pct = 100 * p.Done / p.Total
	}
	sort.Strings(p.Blocked)
	return p
}
func (b *Board) Open() []Task {
	out := []Task{}
	for _, t := range b.Tasks {
		if !t.Archived && t.Status != Done {
			out = append(out, cloneTask(t))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, c := out[i], out[j]
		if (a.Priority == High) != (c.Priority == High) {
			return a.Priority == High
		}
		if a.Blocked() != c.Blocked() {
			return a.Blocked()
		}
		return a.ID < c.ID
	})
	return out
}

// SetGit updates supplied git references, recording only changes.
func (b *Board) SetGit(id, branch, pr, by string, now time.Time) error {
	i, err := b.taskIndex(id)
	if err != nil {
		return err
	}
	t := &b.Tasks[i]
	if branch != "" && branch != t.Branch {
		t.Branch = branch
		record(t, by, "branch "+branch, now)
	}
	if pr != "" && pr != t.PR {
		t.PR = pr
		record(t, by, "PR "+pr, now)
	}
	return nil
}

// Merged completes an approved task after its merge.
func (b *Board) Merged(id, by string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (string, error) {
		if err := require(t, Review); err != nil {
			return "", err
		}
		if !t.Approved {
			return "", fmt.Errorf("tarefa %s precisa estar aprovada", id)
		}
		t.Status = Done
		return "merge feito", nil
	})
}
