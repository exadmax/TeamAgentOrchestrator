package team

import (
	"testing"
)

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Portador Py→Go": "portador-py-go",
		"Revisor QA":     "revisor-qa",
		"Função Ação!":   "funcao-acao",
		"  ":             "",
	} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAddUniqueID(t *testing.T) {
	tm := Starter("demo")
	a, err := tm.Add(Agent{Name: "Revisor QA", Client: "codex", Parent: "orquestrador"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "revisor-qa" {
		t.Fatalf("id = %q", a.ID)
	}
	b, _ := tm.Add(Agent{Name: "Revisor QA", Client: "codex"})
	if b.ID != "revisor-qa-2" {
		t.Fatalf("id = %q", b.ID)
	}
	if _, err := tm.Add(Agent{ID: "arquiteto", Name: "x", Client: "claude"}); err == nil {
		t.Fatal("duplicate id accepted")
	}
}

func TestValidateRejectsBadParentAndCycle(t *testing.T) {
	tm := Starter("demo")
	if _, err := tm.Add(Agent{Name: "Órfão", Client: "claude", Parent: "ninguem"}); err == nil {
		t.Fatal("missing parent accepted")
	}
	a, _ := tm.Get("orquestrador")
	a.Parent = "arquiteto" // arquiteto reports to orquestrador: cycle
	if err := tm.Update(a); err == nil {
		t.Fatal("cycle accepted")
	}
	if got, _ := tm.Get("orquestrador"); got.Parent != "" {
		t.Fatal("failed update changed the team")
	}
}

func TestRemoveMovesChildrenUp(t *testing.T) {
	tm := Starter("demo")
	tm.Add(Agent{ID: "estagiario", Name: "Estagiário", Client: "opencode", Parent: "arquiteto"})
	if err := tm.Remove("arquiteto"); err != nil {
		t.Fatal(err)
	}
	if a, _ := tm.Get("estagiario"); a.Parent != "orquestrador" {
		t.Fatalf("parent = %q", a.Parent)
	}
	if len(tm.Children("orquestrador")) != 3 {
		t.Fatalf("children = %+v", tm.Children("orquestrador"))
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	if _, ok, err := Load(dir); ok || err != nil {
		t.Fatalf("missing file: ok=%v err=%v", ok, err)
	}
	tm := Starter("demo")
	if err := tm.Save(dir); err != nil {
		t.Fatal(err)
	}
	got, ok, err := Load(dir)
	if err != nil || !ok || len(got.Agents) != 4 || got.Name != "demo" {
		t.Fatalf("load = %+v ok=%v err=%v", got, ok, err)
	}
}
