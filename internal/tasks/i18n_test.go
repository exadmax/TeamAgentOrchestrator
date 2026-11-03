package tasks

import (
	"github.com/exadmax/TeamAgentOrchestrator/internal/i18n"
	"testing"
)

func TestEventTextFollowsLanguage(t *testing.T) {
	defer i18n.Set(i18n.EN)
	if err := i18n.Set(i18n.PT); err != nil {
		t.Fatal(err)
	}
	task := Task{}
	record(&task, "agent", "tasks.event.question", fixed, "Ready?")
	event := task.History[0]
	if event.Key != "tasks.event.question" || len(event.Args) != 1 || event.Args[0] != "Ready?" || event.What != "question: Ready?" {
		t.Fatalf("event = %+v", event)
	}
	if got := event.Text(); got != "pergunta: Ready?" {
		t.Fatal(got)
	}
	if err := i18n.Set(i18n.EN); err != nil {
		t.Fatal(err)
	}
	if got := event.Text(); got != "question: Ready?" {
		t.Fatal(got)
	}
	legacy := Event{What: "criada"}
	if legacy.Text() != legacy.What {
		t.Fatal("legacy history changed")
	}
	board := Board{Tasks: []Task{task}}
	dir := t.TempDir()
	check(t, board.Save(dir))
	loaded, err := Load(dir)
	check(t, err)
	if got := loaded.Tasks[0].History[0]; got.Key != event.Key || got.Text() != event.Text() {
		t.Fatalf("round trip = %+v", got)
	}
	copy, ok := loaded.Task("")
	if !ok {
		t.Fatal("missing task")
	}
	copy.History[0].Args[0] = "changed"
	if loaded.Tasks[0].History[0].Args[0] != "Ready?" {
		t.Fatal("event arguments share mutable storage")
	}
}
