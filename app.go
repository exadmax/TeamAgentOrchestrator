package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/exadmax/TeamAgentOrchestrator/internal/clients"
	"github.com/exadmax/TeamAgentOrchestrator/internal/failure"
	"github.com/exadmax/TeamAgentOrchestrator/internal/gitflow"
	"github.com/exadmax/TeamAgentOrchestrator/internal/runner"
	"github.com/exadmax/TeamAgentOrchestrator/internal/store"
	"github.com/exadmax/TeamAgentOrchestrator/internal/stream"
	"github.com/exadmax/TeamAgentOrchestrator/internal/team"
)

const (
	stateDir      = ".tao"
	consoleLimit  = 400 // lines kept per agent console
	activityLimit = 60  // lines in "Atividade Recente"
	testAgent     = "__test__"
	maxRun        = 45 * time.Minute // a session longer than this is a runaway agent
)

// Agent statuses shown in the UI.
const (
	StatusActive = "ativo"
	StatusIdle   = "ocioso"
	StatusError  = "erro"
	StatusPaused = "pausado"
)

// Line is one console line of an agent, already normalized.
type Line struct {
	Agent string    `json:"agent"`
	At    time.Time `json:"at"`
	Kind  string    `json:"kind"` // cmd | init | text | tool | usage | result | error | info | raw | end
	Text  string    `json:"text"`
	Tool  string    `json:"tool,omitempty"`
	Error bool      `json:"error,omitempty"`
}

// AgentView is an agent plus what the UI needs to draw it.
type AgentView struct {
	team.Agent
	Status      string `json:"status"`
	ModelInUse  string `json:"model_in_use"`
	ClientName  string `json:"client_name"`
	ClientColor string `json:"client_color"`
	LastLine    string `json:"last_line"`
	Command     string `json:"command"`
}

// Snapshot is the whole screen state.
type Snapshot struct {
	Project  string             `json:"project"`
	Dir      string             `json:"dir"`
	Agents   []AgentView        `json:"agents"`
	Clients  []clients.Detected `json:"clients"`
	Activity []Line             `json:"activity"`
	Emblems  []string           `json:"emblems"`
	Tiers    []string           `json:"tiers"`
	Clocks   []store.Clock      `json:"clocks"`
}

// TokenRow is one line of the Tokens tab.
type TokenRow struct {
	Agent    string  `json:"agent"`
	Name     string  `json:"name"`
	Client   string  `json:"client"`
	Color    string  `json:"color"`
	Model    string  `json:"model"`
	Status   string  `json:"status"`
	Input    int     `json:"input"`
	Output   int     `json:"output"`
	Total    int     `json:"total"`
	CostUSD  float64 `json:"cost_usd"`
	Sessions int     `json:"sessions"`
	Share    float64 `json:"share"`
}

// App is the facade the Wails frontend calls.
type App struct {
	wapp *application.App // nil in tests

	mu       sync.Mutex
	dir      string
	team     *team.Team
	store    *store.Store
	catalog  []clients.Client
	detected []clients.Detected
	runs     map[string]context.CancelFunc
	failed   map[string]bool
	console  map[string][]Line
	activity []Line
	taskRuns map[string]bool // tasks with a session running
	repo     *gitflow.Repo   // nil = no git flow (not a repo, or turned off)
	gitView  GitView

	bmu sync.Mutex // serializes read-modify-write of tao.tasks.json
}

func NewApp() *App {
	return &App{catalog: clients.Catalog(), runs: map[string]context.CancelFunc{}, failed: map[string]bool{},
		console: map[string][]Line{}, taskRuns: map[string]bool{}}
}

// ServiceStartup is called by Wails before the window opens.
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	dir := lastProject()
	if dir == "" {
		dir, _ = os.Getwd()
	}
	if err := a.open(dir); err != nil {
		log.Printf("open %s: %v", dir, err)
	}
	go a.DetectClients()
	go a.watchClocks(ctx.Done())
	return nil
}

// ServiceShutdown stops every running session when the app closes.
func (a *App) ServiceShutdown() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, cancel := range a.runs {
		cancel()
	}
	return nil
}

// ---------------------------------------------------------------- project

// open loads (or creates) the team of a project folder.
func (a *App) open(dir string) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	t, ok, err := team.Load(dir)
	if err != nil {
		return err
	}
	if !ok {
		t = team.Starter(filepath.Base(dir))
		if err := t.Save(dir); err != nil {
			return err
		}
	}
	repo, view := gitRepoFor(dir, t.Git)
	a.mu.Lock()
	a.dir, a.team, a.store, a.repo, a.gitView = dir, t, store.Open(filepath.Join(dir, stateDir)), repo, view
	a.console, a.activity, a.failed = map[string][]Line{}, nil, map[string]bool{}
	a.mu.Unlock()
	return nil
}

// ChooseProject asks for a folder and opens it.
func (a *App) ChooseProject() (Snapshot, error) {
	if a.wapp == nil {
		return a.Snapshot(), errors.New("sem janela")
	}
	dir, err := a.wapp.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title: "Abrir projeto", CanChooseDirectories: true, CanChooseFiles: false,
	}).PromptForSingleSelection()
	if err != nil || dir == "" {
		return a.Snapshot(), err
	}
	if err := a.open(dir); err != nil {
		return a.Snapshot(), err
	}
	saveLastProject(a.dir)
	snap := a.Snapshot()
	a.emit("snapshot", snap)
	a.emit("board", a.Board())
	return snap, nil
}

// Snapshot returns the full UI state.
func (a *App) Snapshot() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := Snapshot{Dir: a.dir, Clients: a.detected, Emblems: team.Emblems, Tiers: team.Tiers,
		Activity: append([]Line{}, a.activity...)}
	if a.team == nil {
		return s
	}
	s.Project = a.team.Name
	for _, ag := range a.team.Agents {
		s.Agents = append(s.Agents, a.viewLocked(ag))
	}
	s.Clocks = a.store.ActiveClocks()
	return s
}

func (a *App) viewLocked(ag team.Agent) AgentView {
	v := AgentView{Agent: ag, Status: StatusIdle, ModelInUse: a.modelLocked(ag, "")}
	if c, ok := clients.Find(a.catalog, ag.Client); ok {
		v.ClientName, v.ClientColor = c.Name, c.Color
		v.Command = clients.ShowArgv(clients.BuildArgv(c, "headless", "<prompt>", v.ModelInUse))
	}
	switch {
	case a.runs[ag.ID] != nil:
		v.Status = StatusActive
	case ag.Paused:
		v.Status = StatusPaused
	case a.failed[ag.ID]:
		v.Status = StatusError
	}
	if lines := a.console[ag.ID]; len(lines) > 0 {
		v.LastLine = lines[len(lines)-1].Text
	}
	return v
}

// modelLocked: a model swap wins (the task's, then the client's), then the agent's own model, then the
// client's model for its tier.
func (a *App) modelLocked(ag team.Agent, task string) string {
	if a.store != nil {
		if ov, ok := a.store.OverrideFor(ag.Client, task); ok {
			return ov.Model
		}
	}
	if ag.Model != "" {
		return ag.Model
	}
	if c, ok := clients.Find(a.catalog, ag.Client); ok {
		return c.Tiers[ag.Tier]
	}
	return ""
}

// ---------------------------------------------------------------- clients

// DetectClients looks for the CLIs on this machine and pushes the result to the UI.
func (a *App) DetectClients() []clients.Detected {
	found := clients.DetectAll(context.Background(), a.catalog)
	a.mu.Lock()
	a.detected = found
	a.mu.Unlock()
	a.emit("clients", found)
	return found
}

// CheckUpdates asks npm for the latest version of every client.
func (a *App) CheckUpdates() []clients.Detected {
	a.mu.Lock()
	found := append([]clients.Detected{}, a.detected...)
	a.mu.Unlock()
	var wg sync.WaitGroup
	for i := range found {
		if found[i].NpmPackage == "" {
			continue
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if v, err := clients.LatestNpm(context.Background(), found[i].NpmPackage); err == nil {
				found[i].Latest = v
			}
		}(i)
	}
	wg.Wait()
	a.mu.Lock()
	a.detected = found
	a.mu.Unlock()
	a.emit("clients", found)
	return found
}

// ---------------------------------------------------------------- org chart

func (a *App) mutate(f func(t *team.Team) error) (Snapshot, error) {
	a.mu.Lock()
	if a.team == nil {
		a.mu.Unlock()
		return Snapshot{}, errors.New("nenhum projeto aberto")
	}
	next := *a.team
	next.Agents = append([]team.Agent{}, a.team.Agents...)
	err := f(&next)
	if err == nil {
		err = next.Save(a.dir)
	}
	if err == nil {
		a.team = &next
	}
	a.mu.Unlock()
	snap := a.Snapshot()
	if err == nil {
		a.emit("snapshot", snap)
	}
	return snap, err
}

// AddAgent creates a new agent (the "Novo Agente" wizard and "Novo Subordinado").
func (a *App) AddAgent(ag team.Agent) (Snapshot, error) {
	return a.mutate(func(t *team.Team) error {
		_, err := t.Add(ag)
		return err
	})
}

// UpdateAgent saves the properties panel.
func (a *App) UpdateAgent(ag team.Agent) (Snapshot, error) {
	return a.mutate(func(t *team.Team) error { return t.Update(ag) })
}

// RemoveAgent deletes an agent; its subordinates move up.
func (a *App) RemoveAgent(id string) (Snapshot, error) {
	a.StopAgent(id)
	return a.mutate(func(t *team.Team) error { return t.Remove(id) })
}

// TogglePause pauses or resumes an agent.
func (a *App) TogglePause(id string) (Snapshot, error) {
	return a.mutate(func(t *team.Team) error {
		ag, ok := t.Get(id)
		if !ok {
			return fmt.Errorf("agente %q não existe", id)
		}
		ag.Paused = !ag.Paused
		return t.Update(ag)
	})
}

// ---------------------------------------------------------------- running agents

// Console returns the lines kept for an agent.
func (a *App) Console(id string) []Line {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]Line{}, a.console[id]...)
}

// RunAgent sends an instruction to an agent through its CLI. Output arrives as "line" events.
func (a *App) RunAgent(id, prompt string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return errors.New("escreva uma instrução")
	}
	a.mu.Lock()
	if a.team == nil {
		a.mu.Unlock()
		return errors.New("nenhum projeto aberto")
	}
	ag, ok := a.team.Get(id)
	if !ok {
		a.mu.Unlock()
		return fmt.Errorf("agente %q não existe", id)
	}
	if ag.Paused {
		a.mu.Unlock()
		return fmt.Errorf("%s está pausado", ag.Name)
	}
	r, err := a.routeLocked(ag, "")
	if err != nil {
		a.mu.Unlock()
		return err
	}
	full := a.promptLocked(r.agent, prompt)
	a.mu.Unlock()
	a.noteRoute(ag.ID, r)
	return a.launch(r.agent.ID, r.agent.Client, r.model, full, prompt, "", "", nil)
}

// TestClient runs a quick prompt on a client (the "Testar" button of the Clientes tab).
func (a *App) TestClient(clientID, model, prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		prompt = "Olá, você está pronto? Responda em uma linha."
	}
	return a.launch(testAgent, clientID, model, prompt, prompt, "", "", nil)
}

// promptLocked wraps the instruction with who the agent is in the org chart.
func (a *App) promptLocked(ag team.Agent, instruction string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Você é %s (%s) no time do projeto %s.", ag.Name, ag.Role, a.team.Name)
	if boss, ok := a.team.Get(ag.Parent); ok {
		fmt.Fprintf(&b, " Você responde a %s (%s).", boss.Name, boss.Role)
	}
	if subs := a.team.Children(ag.ID); len(subs) > 0 {
		names := make([]string, len(subs))
		for i, s := range subs {
			names[i] = s.Name + " (" + s.Role + ")"
		}
		fmt.Fprintf(&b, " Seus subordinados: %s.", strings.Join(names, ", "))
	}
	if ag.Instructions != "" {
		b.WriteString("\n\n" + ag.Instructions)
	}
	b.WriteString("\n\nInstrução: " + instruction)
	return b.String()
}

// Outcome is how a session ended, handed to the caller that started it (task flow).
type Outcome struct {
	Kind     failure.Kind // "" = ok
	Final    string       // last thing the agent said
	Question string       // the agent stopped on a question
	Exit     int
}

// launch starts a session. task tags the session with a task id (""= free instruction); workDir is
// where the agent works ("" = the project folder; a task with git gets its worktree);
// done, if not nil, runs after the session is recorded.
func (a *App) launch(agentID, clientID, model, fullPrompt, shown, task, workDir string, done func(Outcome)) error {
	c, ok := clients.Find(a.catalog, clientID)
	if !ok {
		return fmt.Errorf("cliente %q desconhecido", clientID)
	}
	exe, ok := clients.ResolveExe(c.Exe)
	if !ok {
		return fmt.Errorf("%s não está instalado (ou não está no PATH)", c.Name)
	}
	argv := clients.BuildArgv(c, "headless", fullPrompt, model)
	argv[0] = exe

	a.mu.Lock()
	if a.runs[agentID] != nil {
		a.mu.Unlock()
		return errors.New("este agente já está trabalhando; aguarde ou pare a sessão")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.runs[agentID] = cancel
	delete(a.failed, agentID)
	dir, st := a.dir, a.store
	a.mu.Unlock()

	run := strings.ReplaceAll(time.Now().Format("20060102-150405.000"), ".", "-")
	logPath := filepath.Join(dir, stateDir, "logs", fmt.Sprintf("%s-%s-%s.log", agentID, clientID, run))
	a.push(Line{Agent: agentID, Kind: "cmd", Text: clients.ShowArgv(clients.BuildArgv(c, "headless", shown, model))})
	a.emit("snapshot", a.Snapshot())

	go func() {
		defer cancel()
		res := runner.Run(ctx, runner.Spec{
			Client: clientID, Argv: argv, Dir: cmp.Or(workDir, dir), Env: c.Env, LogPath: logPath,
			IdleKill: failure.IdleKill, Heartbeat: failure.Heartbeat, MaxRun: maxRun,
		}, runner.Hooks{
			OnEvent: func(ev stream.Event) { a.push(eventLine(agentID, ev)) },
			OnIdle: func(silent time.Duration) {
				a.emit("idle", map[string]any{"agent": agentID, "seconds": int(silent.Seconds())})
			},
		})
		out := a.finish(agentID, clientID, model, run, logPath, task, res, st)
		if done != nil {
			done(out)
		}
	}()
	return nil
}

// finish classifies the run, records the session and the quota clock, and frees the agent.
func (a *App) finish(agentID, clientID, model, run, logPath, task string, res runner.Result, st *store.Store) Outcome {
	kind := failure.Classify(res.Exit, res.Tail)
	if kind == "" && res.IsError {
		// Exit 0 but the CLI reported an error: classify only its own error messages. The raw tail
		// carries JSON keys such as "rate_limit_event" that would read as quota.
		kind = failure.Classify(1, strings.Join(res.Errors, "\n"))
	}
	if res.LaunchErr != "" {
		kind = failure.Launch
	}
	if model == "" {
		model = res.Model
	}
	sess := store.Session{Run: run, Agent: agentID, Client: clientID, Model: model, Mode: "headless", Task: task,
		Start: res.Start, End: res.End, Exit: res.Exit, Error: string(kind), Log: logPath,
		Usage: store.Usage{Input: res.Usage.Input, Output: res.Usage.Output, CacheRead: res.Usage.CacheRead,
			CacheWrite: res.Usage.CacheWrite, CostUSD: res.Usage.CostUSD},
		Status: "ok", Summary: res.Final}
	if kind != "" {
		sess.Status = "error"
		if kind == failure.Quota || kind == failure.Transient {
			now := time.Now()
			wait, told := failure.ParseWait(res.Tail, now)
			sess.Hint, sess.WaitSecs = failure.RetryHint(res.Tail, now), int(wait.Seconds())
			if kind == failure.Quota || wait >= 2*time.Minute {
				if !told {
					wait = failure.DefaultSwitch[failure.Quota]
				}
				_ = st.RecordClock(clientID, model, failure.ErrorModel(res.Tail), wait, string(kind), !told)
			}
		}
		if kind == failure.Model && model != "" {
			// the model is gone or not allowed: keep it off for a day so the switch can pick another one
			_ = st.SetClock(clientID, model, 24*time.Hour, string(kind), "estimado")
		}
	} else if q := failure.LastQuestion(res.Final); q != "" {
		sess.Status, sess.Question = "blocked", q
	}
	if agentID != testAgent {
		_ = st.RecordSession(sess)
	}

	end := Line{Agent: agentID, Kind: "end", Text: fmt.Sprintf("sessão terminou (código %d) · %s · %d tokens",
		res.Exit, failure.FormatDuration(res.End.Sub(res.Start)), res.Usage.Input+res.Usage.Output)}
	if kind != "" {
		end.Error = true
		end.Text = fmt.Sprintf("falha (%s): %s", kind, failure.Help(kind))
		if act := failure.Action(kind); act != "" {
			end.Text += " " + act
		}
		if res.LaunchErr != "" {
			end.Text += " (" + res.LaunchErr + ")"
		}
	}
	a.mu.Lock()
	delete(a.runs, agentID)
	if kind != "" {
		a.failed[agentID] = true
	}
	a.mu.Unlock()
	a.push(end)
	a.emit("snapshot", a.Snapshot())
	return Outcome{Kind: kind, Final: res.Final, Question: sess.Question, Exit: res.Exit}
}

// StopAgent cancels the running session of an agent.
func (a *App) StopAgent(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if cancel := a.runs[id]; cancel != nil {
		cancel()
	}
}

func eventLine(agent string, ev stream.Event) Line {
	l := Line{Agent: agent, Kind: string(ev.Kind), Text: ev.Text, Tool: ev.Tool, Error: ev.IsError}
	switch ev.Kind {
	case stream.Init:
		l.Text = "sessão iniciada"
		if ev.Model != "" {
			l.Text += " · " + ev.Model
		}
	case stream.Usage, stream.Result:
		if ev.Usage != nil {
			l.Kind = "usage"
			l.Text = fmt.Sprintf("tokens: %d entrada · %d saída", ev.Usage.Input, ev.Usage.Output)
		}
	}
	return l
}

// push stores a console line, feeds "Atividade Recente" and sends it to the UI.
func (a *App) push(l Line) {
	if l.Text == "" && l.Tool == "" {
		return
	}
	if l.At.IsZero() {
		l.At = time.Now()
	}
	a.mu.Lock()
	lines := append(a.console[l.Agent], l)
	if len(lines) > consoleLimit {
		lines = lines[len(lines)-consoleLimit:]
	}
	a.console[l.Agent] = lines
	if l.Agent != testAgent && (l.Kind == "text" || l.Kind == "error" || l.Kind == "end" || l.Kind == "cmd") {
		a.activity = append([]Line{l}, a.activity...)
		if len(a.activity) > activityLimit {
			a.activity = a.activity[:activityLimit]
		}
	}
	a.mu.Unlock()
	a.emit("line", l)
}

// ---------------------------------------------------------------- tokens

// Tokens sums the recorded sessions per agent. day = "2006-01-02" or "" for all time.
func (a *App) Tokens(day string) []TokenRow {
	a.mu.Lock()
	st := a.store
	a.mu.Unlock()
	if st == nil {
		return nil
	}
	rows, _ := st.Sessions()
	byAgent := store.UsageBy(rows, func(s store.Session) string { return s.Agent }, day)
	total := 0
	for _, r := range byAgent {
		total += r.Usage.Total()
	}
	views := map[string]AgentView{}
	for _, v := range a.Snapshot().Agents {
		views[v.ID] = v
	}
	out := make([]TokenRow, 0, len(byAgent))
	for _, r := range byAgent {
		row := TokenRow{Agent: r.Key, Name: r.Key, Model: r.Model, Input: r.Usage.Input, Output: r.Usage.Output,
			Total: r.Usage.Total(), CostUSD: r.Usage.CostUSD, Sessions: r.Sessions, Status: StatusIdle}
		if v, ok := views[r.Key]; ok {
			row.Name, row.Client, row.Color, row.Status = v.Name, v.ClientName, v.ClientColor, v.Status
		}
		if total > 0 {
			row.Share = float64(row.Total) / float64(total)
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	return out
}

// ---------------------------------------------------------------- helpers

func (a *App) emit(name string, data any) {
	if a.wapp != nil {
		a.wapp.Event.Emit(name, data)
	}
}

type settings struct {
	LastProject string `json:"last_project"`
}

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "TeamAgentOrchestrator", "settings.json")
}

func lastProject() string {
	var s settings
	if data, err := os.ReadFile(settingsPath()); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	if fi, err := os.Stat(s.LastProject); err == nil && fi.IsDir() {
		return s.LastProject
	}
	return ""
}

func saveLastProject(dir string) {
	path := settingsPath()
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, _ := json.MarshalIndent(settings{LastProject: dir}, "", "  ")
	_ = os.WriteFile(path, data, 0o644)
}
