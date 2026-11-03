package main

import (
	"testing"
	"time"

	"github.com/exadmax/TeamAgentOrchestrator/internal/tasks"
)

// The board operations must never block each other (they run from the UI one after another).
func TestBoardFlowDoesNotBlock(t *testing.T) {
	a := NewApp()
	if err := a.open(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		if err := a.NewSprint("Demo", "objetivo"); err != nil {
			done <- err
			return
		}
		if err := a.StartSprint("S1"); err != nil {
			done <- err
			return
		}
		if err := a.AddTask(tasks.NewTask{Title: "Fazer x", Assignee: "engenheiro", Sprint: "S1"}); err != nil {
			done <- err
			return
		}
		done <- a.ApproveTask("T001") // already ready: must fail fast, not hang
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("board operations blocked")
	}
	b := a.Board()
	if b.Active != "S1" || len(b.Tasks) != 1 || b.Tasks[0].Status != tasks.Ready {
		t.Fatalf("board = %+v", b)
	}
}
