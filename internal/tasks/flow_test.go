package tasks

import "testing"

// The flow finishes the card closest to done before starting another, so agents never pile up.
func TestNextFinishesOneCardAtATime(t *testing.T) {
	b := &Board{}
	a := mustTask(t, b, NewTask{Assignee: "x"})
	c := mustTask(t, b, NewTask{Assignee: "x"})
	if s := b.Next("", true); s.Action != ActRun || s.Task.ID != a.ID {
		t.Fatalf("first step = %+v", s)
	}
	check(t, b.Start(a.ID, "x", fixed))
	check(t, b.Submit(a.ID, "x", "ok", fixed))
	if s := b.Next("", true); s.Action != ActReview || s.Task.ID != a.ID {
		t.Fatalf("review must come before the next card: %+v", s)
	}
	check(t, b.Review(a.ID, "r", true, "", fixed))
	if s := b.Next("", true); s.Action != ActRun || s.Task.ID != c.ID {
		t.Fatalf("after done the next card goes: %+v", s)
	}
	check(t, b.Start(c.ID, "x", fixed))
	check(t, b.Submit(c.ID, "x", "ok", fixed))
	check(t, b.Review(c.ID, "r", true, "", fixed))
	if s := b.Next("", true); s.Action != ActIdle {
		t.Fatalf("all done: %+v", s)
	}
}

func TestNextStopsOnDirectorAndMerge(t *testing.T) {
	b := &Board{}
	a := mustTask(t, b, NewTask{})
	mustTask(t, b, NewTask{})
	mustTask(t, b, NewTask{Proposed: true})
	check(t, b.Ask(a.ID, "x", "which db?", fixed))
	if s := b.Next("", true); s.Action != ActAsk || s.Task.ID != a.ID {
		t.Fatalf("a question pauses the flow: %+v", s)
	}
	check(t, b.Answer(a.ID, "user", "sqlite", fixed))
	check(t, b.Start(a.ID, "x", fixed))
	check(t, b.SetGit(a.ID, "tao/t001", "", "x", fixed))
	check(t, b.Submit(a.ID, "x", "ok", fixed))
	check(t, b.Review(a.ID, "r", true, "", fixed))
	if s := b.Next("", true); s.Action != ActMerge {
		t.Fatalf("approved branch merges: %+v", s)
	}
	if s := b.Next("", false); s.Action != ActAsk {
		t.Fatalf("without auto merge the director merges: %+v", s)
	}
}

func TestDepsAndEscalation(t *testing.T) {
	b := &Board{}
	a := mustTask(t, b, NewTask{})
	c := mustTask(t, b, NewTask{Priority: High})
	check(t, b.SetDeps(c.ID, []string{a.ID}, "user", fixed))
	if e := b.SetDeps(a.ID, []string{c.ID}, "user", fixed); e == nil {
		t.Fatal("cycle accepted")
	}
	if e := b.Start(c.ID, "x", fixed); e == nil {
		t.Fatal("started with an open dependency")
	}
	if s := b.Next("", true); s.Task.ID != a.ID {
		t.Fatalf("high priority card waits for its dependency: %+v", s)
	}
	check(t, b.Start(a.ID, "x", fixed))
	for i := 0; i < MaxReviewRounds; i++ {
		check(t, b.Submit(a.ID, "x", "ok", fixed))
		check(t, b.Review(a.ID, "r", false, "", fixed))
	}
	if s := b.Next("", true); s.Action != ActEscalate {
		t.Fatalf("endless review loop must go to the director: %+v", s)
	}
	check(t, b.Ask(a.ID, "user", "decide", fixed))
	check(t, b.Answer(a.ID, "user", "go on", fixed))
	if s := b.Next("", true); s.Action != ActRun {
		t.Fatalf("after the answer the work resumes: %+v", s)
	}
}
