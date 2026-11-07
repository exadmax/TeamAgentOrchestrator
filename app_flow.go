package main

import (
	"time"

	"github.com/exadmax/TeamAgentOrchestrator/internal/i18n"
	"github.com/exadmax/TeamAgentOrchestrator/internal/tasks"
)

// FlowView is the sequential workflow as the UI shows it: one card at a time, hand to hand
// (assignee → reviewer → merge → next card), stopping only when the director must answer.
type FlowView struct {
	On      bool   `json:"on"`
	Merge   bool   `json:"merge"`   // the flow integrates approved branches itself
	Task    string `json:"task"`    // the card the flow is on
	Action  string `json:"action"`  // what it is doing with it (tasks.Action)
	Waiting bool   `json:"waiting"` // stopped on the director
	Reason  string `json:"reason"`  // why it stopped or is waiting
}

// StartFlow turns the sequential workflow on and makes its first move.
func (a *App) StartFlow(merge bool) error {
	a.mu.Lock()
	if a.dir == "" {
		a.mu.Unlock()
		return i18n.Err("app.no_project")
	}
	a.flow = FlowView{On: true, Merge: merge}
	a.mu.Unlock()
	a.kick()
	return nil
}

// StopFlow turns it off; the session already running finishes, but nothing new starts.
func (a *App) StopFlow() {
	a.mu.Lock()
	a.flow = FlowView{}
	a.mu.Unlock()
	go func() { a.emit("board", a.Board()) }()
}

// Flow returns the state of the workflow.
func (a *App) Flow() FlowView {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.flow
}

// kick makes the next move in the background (it is called from session callbacks, which hold no lock).
func (a *App) kick() { go a.advance() }

// advance moves the flow one step. Only one card is ever in flight: while a session runs it does nothing.
func (a *App) advance() {
	a.fmu.Lock()
	defer a.fmu.Unlock()
	a.mu.Lock()
	on, merge, busy := a.flow.On, a.flow.Merge, len(a.taskRuns) > 0
	a.mu.Unlock()
	if !on || busy {
		return
	}
	var step tasks.Step
	if err := a.board(func(b *tasks.Board) error {
		sprint := ""
		if s, ok := b.ActiveSprint(); ok {
			sprint = s.ID
		}
		step = b.Next(sprint, merge)
		if step.Action == tasks.ActEscalate {
			// too many review rounds: the director decides instead of the agents looping forever
			return b.Ask(step.Task.ID, human, i18n.T("flow.escalate", step.Task.Rounds()), time.Now())
		}
		return nil
	}); err != nil {
		a.stopFlow(err.Error())
		return
	}
	id := step.Task.ID
	a.setFlow(id, string(step.Action), false, "")
	var err error
	switch step.Action {
	case tasks.ActIdle:
		a.stopFlow(i18n.T("flow.finished"))
		a.push(Line{Agent: "flow", Kind: "end", Text: i18n.T("flow.finished")})
		return
	case tasks.ActAsk, tasks.ActEscalate:
		reason := i18n.T("flow.waiting_answer", id)
		if step.Action == tasks.ActAsk && step.Task.Approved && !step.Task.Blocked() {
			reason = i18n.T("flow.waiting_merge", id)
		}
		a.setFlow(id, string(step.Action), true, reason)
		a.push(Line{Agent: "flow", Kind: "info", Text: reason})
		return
	case tasks.ActRun:
		err = a.RunTask(id)
	case tasks.ActReview:
		err = a.ReviewTask(id)
	case tasks.ActMerge:
		err = a.MergeTask(id) // MergeTask kicks the next move itself
	}
	if err != nil {
		// a step that cannot even start (no reviewer, agent paused, git error) needs the director
		a.warn(id, a.board(func(b *tasks.Board) error { return b.Ask(id, human, i18n.T("flow.step_failed", err), time.Now()) }))
		a.setFlow(id, string(step.Action), true, i18n.T("flow.waiting_answer", id))
	}
}

func (a *App) setFlow(task, action string, waiting bool, reason string) {
	a.mu.Lock()
	if a.flow.On {
		a.flow.Task, a.flow.Action, a.flow.Waiting, a.flow.Reason = task, action, waiting, reason
	}
	a.mu.Unlock()
	go func() { a.emit("board", a.Board()) }()
}

func (a *App) stopFlow(reason string) {
	a.mu.Lock()
	a.flow = FlowView{Reason: reason}
	a.mu.Unlock()
	go func() { a.emit("board", a.Board()) }()
}

// flowBusy refuses a manual start of another card while the flow has one in flight: agents must not pile up.
func (a *App) flowBusy(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.flow.On {
		return nil
	}
	for other := range a.taskRuns {
		if other != id {
			return i18n.Err("flow.busy", other)
		}
	}
	return nil
}
