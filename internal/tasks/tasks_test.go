package tasks

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var fixed = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)

func mustTask(t *testing.T, b *Board, n NewTask) Task {
	t.Helper()
	if n.Title == "" {
		n.Title = "Atividade"
	}
	v, e := b.AddTask(n, "diogo", fixed)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func check(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func TestTransitions(t *testing.T) {
	cases := []struct {
		name    string
		allowed []Status
		run     func(*Board) error
	}{
		{"approve", []Status{Proposed}, func(b *Board) error { return b.Approve("T001", "a", fixed) }},
		{"start", []Status{Ready, InProgress}, func(b *Board) error { return b.Start("T001", "a", fixed) }},
		{"ask", []Status{Proposed, Ready, InProgress, Review}, func(b *Board) error { return b.Ask("T001", "a", "Como?", fixed) }},
		{"answer", []Status{Proposed, Ready, InProgress, Review}, func(b *Board) error { return b.Answer("T001", "a", "Assim", fixed) }},
		{"submit", []Status{InProgress}, func(b *Board) error { return b.Submit("T001", "a", "Resultado", fixed) }},
		{"review approve", []Status{Review}, func(b *Board) error { return b.Review("T001", "r", true, "Boa", fixed) }},
		{"review changes", []Status{Review}, func(b *Board) error { return b.Review("T001", "r", false, "Corrija", fixed) }},
		{"delegate", []Status{Proposed, Ready, InProgress, Review}, func(b *Board) error { return b.Delegate("T001", "b", "a", fixed) }},
		{"reopen", []Status{Done}, func(b *Board) error { return b.Reopen("T001", "a", "Refazer", fixed) }},
		{"archive", Statuses, func(b *Board) error { return b.Archive("T001", "a", fixed) }},
		{"priority", Statuses, func(b *Board) error { return b.SetPriority("T001", High, "a", fixed) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, state := range Statuses {
				for _, archived := range []bool{false, true} {
					b := &Board{Tasks: []Task{{ID: "T001", Title: "Teste", Status: state, Priority: Normal, Question: "Pendente", Archived: archived, Verdicts: []string{"old:approve"}}}}
					before := cloneTask(b.Tasks[0])
					want := false
					for _, s := range c.allowed {
						if s == state {
							want = true
						}
					}
					if archived && c.name != "reopen" && c.name != "archive" && c.name != "priority" {
						want = false
					}
					err := c.run(b)
					if (err == nil) != want {
						t.Fatalf("%s archived=%v: %v", state, archived, err)
					}
					if !want {
						if !reflect.DeepEqual(before, b.Tasks[0]) {
							t.Fatal("erro alterou quadro")
						}
						continue
					}
					got := b.Tasks[0]
					if !got.Updated.Equal(fixed) || len(got.History) != 1 || got.History[0].By == "" {
						t.Fatal(got)
					}
					switch c.name {
					case "approve", "reopen":
						if got.Status != Ready {
							t.Fatal(got)
						}
					case "start", "review changes":
						if got.Status != InProgress || got.Blocked() {
							t.Fatal(got)
						}
					case "answer":
						if got.Blocked() {
							t.Fatal(got)
						}
					case "submit":
						if got.Status != Review || got.Summary != "Resultado" {
							t.Fatal(got)
						}
					case "review approve":
						if got.Status != Done || got.Verdicts[1] != "r:approve" {
							t.Fatal(got)
						}
					case "delegate":
						if got.Assignee != "b" {
							t.Fatal(got)
						}
					case "archive":
						if !got.Archived {
							t.Fatal(got)
						}
					case "priority":
						if got.Priority != High {
							t.Fatal(got)
						}
					}
					if c.name == "reopen" && (got.Archived || len(got.Verdicts) != 0) {
						t.Fatal(got)
					}
				}
			}
			if c.run(&Board{}) == nil {
				t.Fatal("ID inexistente aceito")
			}
		})
	}
}

func TestTaskEditingAndHistory(t *testing.T) {
	b := &Board{}
	v := mustTask(t, b, NewTask{Proposed: true})
	if v.ID != "T001" || v.Status != Proposed || v.Priority != Normal || v.Origin != "manual" || v.History[0].What != "criada" || !v.Created.Equal(fixed) {
		t.Fatal(v)
	}
	s, e := b.NewSprint("Sprint", "Meta", fixed)
	check(t, e)
	v.Title = "Editada"
	v.Done = "Verificável"
	v.Assignee = "a"
	v.Reviewer = "r"
	v.Sprint = s.ID
	v.Priority = High
	v.Status = Done
	v.Origin = "falso"
	v.History = nil
	later := fixed.Add(time.Hour)
	check(t, b.UpdateTask(v, "diogo", later))
	got, _ := b.Task(v.ID)
	if got.Status != Proposed || got.Origin != "manual" || len(got.History) != 2 || got.Title != v.Title || got.Done != v.Done || got.Reviewer != "r" || got.Assignee != "a" || got.Sprint != s.ID || got.Priority != High || !got.Updated.Equal(later) {
		t.Fatal(got)
	}
	for i := 0; i < historyLimit+5; i++ {
		check(t, b.SetPriority(v.ID, Normal, "a", later))
	}
	got, _ = b.Task(v.ID)
	if len(got.History) != historyLimit || got.History[0].What != "prioridade alterada" {
		t.Fatal(got)
	}
	got.History[0].What = "alterado"
	got.Verdicts = append(got.Verdicts, "x")
	again, _ := b.Task(v.ID)
	if again.History[0].What == "alterado" {
		t.Fatal("leitura compartilha histórico")
	}
	if _, ok := b.Task("missing"); ok {
		t.Fatal("tarefa inexistente")
	}
	if b.UpdateTask(Task{ID: "missing"}, "a", fixed) == nil {
		t.Fatal("update inexistente")
	}
	invalid := []NewTask{{Title: " "}, {Title: "x", Priority: "bad"}, {Title: "x", Sprint: "missing"}}
	for _, n := range invalid {
		if _, e := b.AddTask(n, "a", fixed); e == nil {
			t.Fatal(n)
		}
	}
	for _, edit := range []func(*Task){func(v *Task) { v.Title = " " }, func(v *Task) { v.Priority = "bad" }, func(v *Task) { v.Sprint = "missing" }} {
		v, _ := b.Task("T001")
		edit(&v)
		if b.UpdateTask(v, "a", fixed) == nil {
			t.Fatal(v)
		}
	}
	if b.SetPriority("T001", "bad", "a", fixed) == nil || b.Ask("T001", "a", " ", fixed) == nil || b.Answer("T001", "a", "sem pergunta", fixed) == nil {
		t.Fatal("validação ausente")
	}
	b.Tasks = append(b.Tasks, Task{ID: "T099"}, Task{ID: "Tbad"})
	if v := mustTask(t, b, NewTask{}); v.ID != "T100" {
		t.Fatal(v.ID)
	}
	labels := []string{"Proposta", "A fazer", "Fazendo", "Em revisão", "Concluída"}
	for i, s := range Statuses {
		if s.Label() != labels[i] {
			t.Fatal(s)
		}
	}
}

func TestSprints(t *testing.T) {
	for _, carry := range []bool{false, true} {
		t.Run(map[bool]string{false: "backlog", true: "carry"}[carry], func(t *testing.T) {
			b := &Board{}
			if _, e := b.NewSprint(" ", "", fixed); e == nil {
				t.Fatal("título vazio")
			}
			a, e := b.NewSprint("Primeira", "Meta", fixed)
			check(t, e)
			c, e := b.NewSprint("Segunda", "", fixed)
			check(t, e)
			if a.ID != "S1" || c.ID != "S2" || a.Status != Planned {
				t.Fatal(a, c)
			}
			if _, ok := b.ActiveSprint(); ok {
				t.Fatal("ativa prematura")
			}
			if b.CloseSprint(a.ID, nil, "", fixed) == nil {
				t.Fatal("fechou planejada")
			}
			check(t, b.StartSprint(a.ID, fixed))
			if b.StartSprint(c.ID, fixed) == nil || b.StartSprint(a.ID, fixed) == nil {
				t.Fatal("duas ativas")
			}
			active, ok := b.ActiveSprint()
			if !ok || active.ID != a.ID || !active.Started.Equal(fixed) {
				t.Fatal(active)
			}
			for _, state := range []Status{Ready, Done, Review} {
				v := mustTask(t, b, NewTask{Sprint: a.ID})
				b.Tasks[len(b.Tasks)-1].Status = state
				if state == Review {
					check(t, b.Archive(v.ID, "a", fixed))
				}
			}
			before := b.Tasks[0]
			for _, dest := range []string{a.ID, "missing"} {
				if b.CloseSprint(a.ID, nil, dest, fixed) == nil {
					t.Fatal(dest)
				}
				if !reflect.DeepEqual(before, b.Tasks[0]) {
					t.Fatal("mutação parcial")
				}
			}
			dest := ""
			if carry {
				dest = c.ID
			}
			later := fixed.Add(time.Hour)
			check(t, b.CloseSprint(a.ID, []string{"Aprendizado"}, dest, later))
			if b.Tasks[0].Sprint != dest || !b.Tasks[0].Updated.Equal(later) || b.Tasks[1].Sprint != a.ID || b.Tasks[2].Sprint != a.ID {
				t.Fatal(b.Tasks)
			}
			closed, _ := b.Sprint(a.ID)
			if closed.Status != Closed || !closed.Closed.Equal(later) || closed.Retro[0] != "Aprendizado" {
				t.Fatal(closed)
			}
			if b.StartSprint(a.ID, fixed) == nil || b.CloseSprint(a.ID, nil, "", fixed) == nil {
				t.Fatal("sprint fechada aceita")
			}
			if _, e := b.AddTask(NewTask{Title: "x", Sprint: a.ID}, "a", fixed); e == nil {
				t.Fatal("tarefa em sprint fechada")
			}
			check(t, b.StartSprint(c.ID, fixed))
			if b.CloseSprint(c.ID, nil, a.ID, fixed) == nil {
				t.Fatal("carry fechado")
			}
			check(t, b.CloseSprint(c.ID, nil, "", fixed))
			if b.StartSprint("missing", fixed) == nil || b.CloseSprint("missing", nil, "", fixed) == nil {
				t.Fatal("sprint inexistente")
			}
			if _, ok := b.Sprint("missing"); ok {
				t.Fatal("sprint inexistente encontrada")
			}
			b.Sprints = append(b.Sprints, Sprint{ID: "S20"}, Sprint{ID: "Sbad"})
			s, e := b.NewSprint("Nova", "", fixed)
			check(t, e)
			if s.ID != "S21" {
				t.Fatal(s.ID)
			}
		})
	}
}

func TestProgressAndOpen(t *testing.T) {
	b := &Board{Tasks: []Task{
		{ID: "T005", Status: Ready, Priority: Normal},
		{ID: "T004", Status: Ready, Priority: Normal, Question: "?"},
		{ID: "T003", Status: Ready, Priority: High},
		{ID: "T002", Status: Ready, Priority: High, Question: "?"},
		{ID: "T001", Status: Done},
		{ID: "T006", Status: Ready, Archived: true, Question: "?"},
		{ID: "T007", Status: Done, Sprint: "S1"},
	}}
	p := b.Progress("")
	if p.Done != 1 || p.Total != 5 || p.Pct != 20 || !reflect.DeepEqual(p.Blocked, []string{"T002", "T004"}) {
		t.Fatal(p)
	}
	if p = b.Progress("S1"); p.Total != 1 || p.Pct != 100 {
		t.Fatal(p)
	}
	if p = b.Progress("missing"); p.Total != 0 || p.Pct != 0 {
		t.Fatal(p)
	}
	ids := []string{}
	for _, v := range b.Open() {
		ids = append(ids, v.ID)
	}
	if !reflect.DeepEqual(ids, []string{"T002", "T003", "T004", "T005"}) {
		t.Fatal(ids)
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	b, e := Load(dir)
	check(t, e)
	if len(b.Tasks) != 0 || len(b.Sprints) != 0 {
		t.Fatal(b)
	}
	mustTask(t, b, NewTask{Title: "Ação"})
	_, e = b.NewSprint("Sprint", "Meta", fixed)
	check(t, e)
	check(t, b.Save(dir))
	check(t, b.Save(dir))
	raw, e := os.ReadFile(filepath.Join(dir, FileName))
	check(t, e)
	if !strings.HasSuffix(string(raw), "\n") || !strings.Contains(string(raw), "\n  \"sprints\": [") {
		t.Fatal(string(raw))
	}
	loaded, e := Load(dir)
	check(t, e)
	if !reflect.DeepEqual(b, loaded) {
		t.Fatal(loaded, b)
	}
	entries, e := os.ReadDir(dir)
	check(t, e)
	if len(entries) != 1 {
		t.Fatal(entries)
	}
	check(t, os.WriteFile(filepath.Join(dir, FileName), []byte("invalid"), 0600))
	if _, e = Load(dir); e == nil {
		t.Fatal("JSON inválido")
	}
	if b.Save(filepath.Join(dir, "missing")) == nil {
		t.Fatal("diretório inexistente")
	}
	blocked := t.TempDir()
	check(t, os.Mkdir(filepath.Join(blocked, FileName), 0700))
	if b.Save(blocked) == nil {
		t.Fatal("rename deveria falhar")
	}
	if _, e = Load(blocked); e == nil {
		t.Fatal("leitura deveria falhar")
	}
}

func TestParseGenerated(t *testing.T) {
	array := `[{"title":"Implementar [parte]","done":"Teste passa","assignee":"a","priority":"high"},{"title":"Outra","assignee":"unknown"},{"title":" "}]`
	for _, text := range []string{array, "antes\n```json\n" + array + "\n```\ndepois", "lixo antes " + array + " lixo depois", "[inválido] depois " + array} {
		got, e := ParseGenerated(text, []string{"a"})
		check(t, e)
		if len(got) != 2 || got[0].Assignee != "a" || got[0].Priority != High || got[1].Assignee != "" || got[1].Priority != Normal || got[0].Done != "Teste passa" {
			t.Fatal(got)
		}
	}
	for _, text := range []string{"nada", "{}", "[broken]", `[{"title":3}]`} {
		if _, e := ParseGenerated(text, nil); e == nil {
			t.Fatal(text)
		}
	}
	got, e := ParseGenerated(`[{"done":"sem título"}]`, nil)
	check(t, e)
	if len(got) != 0 {
		t.Fatal(got)
	}
	got, e = ParseGenerated("[]", nil)
	check(t, e)
	if len(got) != 0 {
		t.Fatal(got)
	}
}

func TestGitReviewAndMerge(t *testing.T) {
	b := &Board{Tasks: []Task{{ID: "T001", Status: Review}}}
	check(t, b.SetGit("T001", "feature/T001", "https://example.test/pr/1", "a", fixed))
	task := b.Tasks[0]
	if task.Branch != "feature/T001" || task.PR != "https://example.test/pr/1" || len(task.History) != 2 || task.History[0].What != "branch feature/T001" || task.History[1].What != "PR https://example.test/pr/1" {
		t.Fatal(task)
	}
	before := cloneTask(task)
	check(t, b.SetGit("T001", "", "", "a", fixed.Add(time.Hour)))
	check(t, b.SetGit("T001", task.Branch, task.PR, "a", fixed.Add(time.Hour)))
	if !reflect.DeepEqual(before, b.Tasks[0]) {
		t.Fatal("unchanged references mutated task")
	}
	if b.Merged("T001", "a", fixed) == nil {
		t.Fatal("merged without approval")
	}
	check(t, b.Review("T001", "r", true, "", fixed))
	task = b.Tasks[0]
	if task.Status != Review || !task.Approved || task.History[len(task.History)-1].What != "revisão: aprovada; falta o merge" {
		t.Fatal(task)
	}
	check(t, b.Merged("T001", "a", fixed))
	task = b.Tasks[0]
	if task.Status != Done || task.History[len(task.History)-1].What != "merge feito" {
		t.Fatal(task)
	}
	check(t, b.Reopen("T001", "a", "Refazer", fixed))
	if b.Tasks[0].Approved {
		t.Fatal("reopen kept approval")
	}
	b.Tasks[0].Approved = true
	check(t, b.Start("T001", "a", fixed))
	if b.Tasks[0].Approved {
		t.Fatal("start kept approval")
	}
	b.Tasks[0].Status = Review
	b.Tasks[0].Approved = true
	check(t, b.Review("T001", "r", false, "Corrigir", fixed))
	if b.Tasks[0].Approved || b.Tasks[0].Status != InProgress {
		t.Fatal(b.Tasks[0])
	}
	check(t, b.SetGit("T001", "", "https://example.test/pr/2", "a", fixed))
	if b.Tasks[0].Branch != "feature/T001" || b.Tasks[0].PR != "https://example.test/pr/2" {
		t.Fatal(b.Tasks[0])
	}
	if b.SetGit("missing", "x", "", "a", fixed) == nil || b.Merged("missing", "a", fixed) == nil {
		t.Fatal("missing id accepted")
	}
}

func TestMergedRequiresReview(t *testing.T) {
	for _, status := range Statuses {
		for _, approved := range []bool{false, true} {
			for _, archived := range []bool{false, true} {
				b := &Board{Tasks: []Task{{ID: "T001", Status: status, Approved: approved, Archived: archived}}}
				before := cloneTask(b.Tasks[0])
				err := b.Merged("T001", "a", fixed)
				want := status == Review && approved && !archived
				if (err == nil) != want {
					t.Fatalf("%s approved=%v archived=%v: %v", status, approved, archived, err)
				}
				if !want && !reflect.DeepEqual(before, b.Tasks[0]) {
					t.Fatal("rejected merge mutated task")
				}
			}
		}
	}
}

func TestGitFieldsRoundTrip(t *testing.T) {
	b := &Board{Tasks: []Task{{ID: "T001", Status: Review, Branch: "feature", PR: "https://example.test/pr/1", Approved: true}}}
	dir := t.TempDir()
	check(t, b.Save(dir))
	got, err := Load(dir)
	check(t, err)
	if !reflect.DeepEqual(b, got) {
		t.Fatal(got)
	}
}

func TestGitFieldsOmittedWhenEmpty(t *testing.T) {
	b := &Board{Tasks: []Task{{ID: "T001"}}}
	dir := t.TempDir()
	check(t, b.Save(dir))
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	check(t, err)
	for _, key := range []string{`"branch"`, `"pr"`, `"approved"`} {
		if strings.Contains(string(data), key) {
			t.Fatalf("empty field persisted: %s", data)
		}
	}
}
