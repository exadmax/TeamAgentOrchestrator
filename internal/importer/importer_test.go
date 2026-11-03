package importer

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/exadmax/TeamAgentOrchestrator/internal/tasks"
)

var fixed = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func TestImportJin(t *testing.T) {
	dir := filepath.Join("testdata", "jin")
	if !Detect(dir) || Detect(t.TempDir()) {
		t.Fatal("detection mismatch")
	}
	team, b, report, err := Import(dir, "Jin", fixed)
	if err != nil {
		t.Fatal(err)
	}
	if report.Agents != 6 || report.Tasks != 3 || report.Sprints != 2 || report.Warnings == nil || len(report.Warnings) != 0 {
		t.Fatal(report)
	}
	if team.Name != "Jin" {
		t.Fatal(team.Name)
	}
	ids := []string{"produtor", "arquiteto", "gameplay", "guardiao-canon", "manutencao", "revisor"}
	names := []string{"Produtor", "Arquiteto", "Gameplay", "Guardiao Canon", "Manutencao", "Revisor"}
	clients := []string{"claude", "claude", "codex", "gemini", "opencode", "claude"}
	tiers := []string{"senior", "senior", "medium", "junior", "junior", "medium"}
	badges := []string{"gestao", "planejamento", "codigo", "pesquisa", "dados", "revisao"}
	for i, a := range team.Agents {
		parent := "produtor"
		if i == 0 {
			parent = ""
		}
		if a.ID != ids[i] || a.Name != names[i] || a.Client != clients[i] || a.Tier != tiers[i] || a.Emblem != badges[i] || a.Parent != parent || a.Face != faces[i] || a.Role != "role "+ids[i]+" (imported)" {
			t.Fatal(a)
		}
	}
	if team.Agents[0].Instructions != "Read first: docs/equipe/papeis/produtor.md\nFiles you may edit: docs/tasks/" {
		t.Fatal(team.Agents[0])
	}
	if b.Tasks[0].ID != "T001" || b.Tasks[1].ID != "T005" || b.Tasks[2].ID != "T006" {
		t.Fatal(b.Tasks)
	}
	review := b.Tasks[1]
	if review.Status != tasks.Review || review.Reviewer != "gameplay" || !review.Approved || review.Branch != "claude/T005-vite-ts-strict-phaser-e-tela-320" || review.Summary != "Vite + TS + Phaser." || review.Assignee != "arquiteto" || review.Priority != tasks.Normal || review.Origin != "import:docs/tasks/T005.md" || review.Done == "" {
		t.Fatal(review)
	}
	if review.Created.Format("2006-01-02") != "2026-11-02" || !review.Updated.Equal(fixed) {
		t.Fatal(review)
	}
	ready := b.Tasks[2]
	if ready.Status != tasks.Ready || ready.Reviewer != "revisor" || ready.Sprint != "S02" || ready.Priority != tasks.High || ready.Question != "Como continuar?" || !ready.Created.Equal(fixed) {
		t.Fatal(ready)
	}
	if b.Tasks[0].Status != tasks.Done || b.Sprints[0].ID != "S01" || b.Sprints[0].Status != tasks.Closed || b.Sprints[0].Closed == nil || b.Sprints[0].Started == nil || b.Sprints[1].ID != "S02" || b.Sprints[1].Status != tasks.Active || b.Sprints[1].Started == nil || b.Sprints[1].Closed != nil {
		t.Fatal(b.Sprints)
	}
	dest := t.TempDir()
	if err := b.Save(dest); err != nil {
		t.Fatal(err)
	}
	loaded, err := tasks.Load(dest)
	if err != nil || !reflect.DeepEqual(b, loaded) {
		t.Fatalf("round trip: %v, %#v", err, loaded)
	}
}

func writeFile(t *testing.T, dir, name, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestImportWarningsAndSprints(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ownersPath, `{"roles":{"level-designer":{"tool":"codex","model":"custom"}}}`)
	writeFile(t, dir, "docs/tasks/bad.md", "---\nid: \"bad\"\nbroken line\n---")
	writeFile(t, dir, "docs/tasks/open.md", "---\nid: \"T002\"\ntitle: Raw title\nrole: \"missing\"\nstatus: \"unknown\"\nsprint: \"S01\"\nquestions: [\"Question?\"]\n---")
	writeFile(t, dir, "docs/tasks/blocked.md", "---\nid: \"T003\"\ntitle: \"Blocked\"\nrole: \"level-designer\"\nstatus: \"blocked\"\nsprint: \"S02\"\nquestions: [{\"text\":\"Pending?\"}]\nverdicts: [\"x:approve\"]\n---")
	team, b, report, err := Import(dir, "Other", fixed)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Warnings) != 3 || report.Tasks != 2 {
		t.Fatal(report)
	}
	a := team.Agents[0]
	if a.Name != "Level Designer" || a.Parent != "" || a.Tier != "medium" || a.Emblem != "criacao" {
		t.Fatal(a)
	}
	if b.Tasks[0].Assignee != "" || b.Tasks[0].Status != tasks.Ready || b.Tasks[0].Title != "Raw title" || b.Tasks[0].Question != "Question?" || b.Tasks[0].Reviewer != "" {
		t.Fatal(b.Tasks[0])
	}
	if b.Tasks[1].Status != tasks.InProgress || b.Tasks[1].Approved || b.Tasks[1].Question != "Pending?" {
		t.Fatal(b.Tasks[1])
	}
	if b.Sprints[0].Status != tasks.Planned || b.Sprints[0].Started != nil || b.Sprints[1].Status != tasks.Active {
		t.Fatal(b.Sprints)
	}
}

func TestImportErrors(t *testing.T) {
	dir := t.TempDir()
	if _, _, report, err := Import(dir, "Missing", fixed); err == nil || report.Warnings == nil {
		t.Fatal("missing owners must fail with initialized report")
	}
	for _, data := range []string{`{`, `{"roles":[]}`, `{"roles":{"empty":{}}}`} {
		writeFile(t, dir, ownersPath, data)
		if _, _, _, err := Import(dir, "Bad", fixed); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}

func TestParseCard(t *testing.T) {
	for _, data := range []string{"body", "---\nid: \"T1\"", "---\nid: \"T1\"\n---", "---\n: 1\n---"} {
		if _, err := parseCard([]byte(data)); err == nil {
			t.Fatalf("accepted %q", data)
		}
	}
	fields, err := parseCard([]byte("---\nid: \"T1\"\ntitle: \"Title\"\nextra: {\"x\":true}\nnumber: 3\n---\nbroken body"))
	if err != nil || !strings.Contains(text(fields, "title"), "Title") {
		t.Fatal(fields, err)
	}
	for _, value := range []any{nil, []any{}, []any{3}, []any{map[string]any{"q": 3}}} {
		if question(value) != "" {
			t.Fatal(value)
		}
	}
}

func TestImportStatusAndApproval(t *testing.T) {
	cases := []struct {
		status   string
		verdicts string
		want     tasks.Status
		approved bool
	}{
		{"proposed", "[]", tasks.Proposed, false},
		{"ready", "[]", tasks.Ready, false},
		{"in_progress", "[]", tasks.InProgress, false},
		{"review", "[\"r:approve\"]", tasks.Review, true},
		{"review", "[\"r:approve\",\"r:changes\"]", tasks.Review, false},
		{"review", "[]", tasks.Review, false},
		{"done", "[\"r:approve\"]", tasks.Done, false},
	}
	for _, c := range cases {
		t.Run(c.status+c.verdicts, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, ownersPath, `{"roles":{"dev":{"tool":"codex","model":"medio"}}}`)
			card := "---\nid: \"T009\"\ntitle: \"Test\"\nrole: \"dev\"\nstatus: \"" + c.status + "\"\nverdicts: " + c.verdicts + "\nopened: \"not-a-date\"\n---\n"
			writeFile(t, dir, "docs/tasks/card.md", card)
			_, b, report, err := Import(dir, "Test", fixed)
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Warnings) != 0 || len(b.Sprints) != 0 || len(b.Tasks) != 1 {
				t.Fatal(report, b)
			}
			task := b.Tasks[0]
			if task.Status != c.want || task.Approved != c.approved || !task.Created.Equal(fixed) {
				t.Fatal(task)
			}
		})
	}
}
