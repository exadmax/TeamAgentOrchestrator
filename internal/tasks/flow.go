package tasks

import (
	"sort"
	"strings"
	"time"

	"github.com/exadmax/TeamAgentOrchestrator/internal/i18n"
)

// MaxReviewRounds is how many "changes" verdicts a task may collect before the flow stops and asks the director.
const MaxReviewRounds = 3

// Action is what the sequential flow must do next.
type Action string

const (
	ActRun      Action = "run"      // hand the task to its assignee
	ActReview   Action = "review"   // hand the work to a reviewer on another client
	ActMerge    Action = "merge"    // integrate the approved branch
	ActAsk      Action = "ask"      // the director must answer the task's question
	ActEscalate Action = "escalate" // too many review rounds: the director must decide
	ActIdle     Action = ""         // nothing left the flow can move
)

// Step is the single next move of the flow.
type Step struct {
	Task   Task   `json:"task"`
	Action Action `json:"action"`
}

// DepsOpen lists the dependencies of t that are not done yet (unknown ids count as open).
func (b *Board) DepsOpen(t Task) []string {
	var open []string
	for _, d := range t.Deps {
		dep, ok := b.Task(d)
		if !ok || dep.Status != Done {
			open = append(open, d)
		}
	}
	return open
}

// Rounds counts the "changes" verdicts the task received.
func (t Task) Rounds() int {
	n := 0
	for _, v := range t.Verdicts {
		if strings.HasSuffix(v, ":changes") {
			n++
		}
	}
	return n
}

// stage orders tasks by how far they are: the flow always finishes what is closest to done first,
// so work never piles up half-way (one card goes hand to hand until it is closed).
func stage(t Task) int {
	switch {
	case t.Status == Review && t.Approved:
		return 0
	case t.Status == Review:
		return 1
	case t.Status == InProgress:
		return 2
	}
	return 3
}

// Next picks the one move the flow makes now in the given sprint ("" is the backlog). Proposed and
// archived tasks are left alone (a proposal waits for the director's approval), as are tasks whose
// dependencies are still open. A blocked task stops the flow: it returns ActAsk instead of skipping it,
// because the work is sequential. merge says whether the flow integrates approved branches itself.
func (b *Board) Next(sprint string, merge bool) Step {
	var cand []Task
	for _, t := range b.Tasks {
		if t.Archived || t.Sprint != sprint || t.Status == Done || t.Status == Proposed {
			continue
		}
		if t.Status == Ready && len(b.DepsOpen(t)) > 0 {
			continue
		}
		cand = append(cand, cloneTask(t))
	}
	sort.SliceStable(cand, func(i, j int) bool {
		a, c := cand[i], cand[j]
		if stage(a) != stage(c) {
			return stage(a) < stage(c)
		}
		if (a.Priority == High) != (c.Priority == High) {
			return a.Priority == High
		}
		return a.ID < c.ID
	})
	for _, t := range cand {
		switch {
		case t.Blocked():
			return Step{Task: t, Action: ActAsk}
		case t.Status == Review && t.Approved:
			if !merge {
				return Step{Task: t, Action: ActAsk}
			}
			return Step{Task: t, Action: ActMerge}
		case t.Status == Review:
			return Step{Task: t, Action: ActReview}
		case t.Rounds() > 0 && t.Rounds()%MaxReviewRounds == 0 && lastKey(t) == "tasks.event.review_changes":
			return Step{Task: t, Action: ActEscalate}
		default:
			return Step{Task: t, Action: ActRun}
		}
	}
	return Step{}
}

// SetDeps replaces the dependencies of a task, refusing itself, unknown ids and cycles.
func (b *Board) SetDeps(id string, deps []string, by string, now time.Time) error {
	return b.mutate(id, by, now, func(t *Task) (Event, error) {
		clean := []string{}
		seen := map[string]bool{}
		for _, d := range deps {
			d = strings.TrimSpace(d)
			if d == "" || seen[d] {
				continue
			}
			if d == id {
				return Event{}, i18n.Err("tasks.dep_self", id)
			}
			if _, ok := b.Task(d); !ok {
				return Event{}, i18n.Err("tasks.not_found", d)
			}
			if b.reaches(d, id) {
				return Event{}, i18n.Err("tasks.dep_cycle", id, d)
			}
			seen[d] = true
			clean = append(clean, d)
		}
		t.Deps = clean
		return Event{Key: "tasks.event.deps", Args: []string{orNone(strings.Join(clean, ", "))}}, nil
	})
}

// reaches reports whether from depends (directly or not) on to.
func (b *Board) reaches(from, to string) bool {
	seen := map[string]bool{}
	var walk func(string) bool
	walk = func(id string) bool {
		if id == to {
			return true
		}
		if seen[id] {
			return false
		}
		seen[id] = true
		t, ok := b.Task(id)
		if !ok {
			return false
		}
		for _, d := range t.Deps {
			if walk(d) {
				return true
			}
		}
		return false
	}
	return walk(from)
}

func orNone(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// lastKey is the key of the last event without its "_note" suffix: an answer by the director comes after it.
func lastKey(t Task) string {
	if len(t.History) == 0 {
		return ""
	}
	return strings.TrimSuffix(t.History[len(t.History)-1].Key, "_note")
}
