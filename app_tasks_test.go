package main

import (
	"strings"
	"testing"

	"github.com/exadmax/TeamAgentOrchestrator/internal/tasks"
)

func TestParseVerdict(t *testing.T) {
	for _, c := range []struct {
		text    string
		approve bool
		note    string
		ok      bool
	}{
		{"Conferi os testes.\nAPROVADO", true, "", true},
		{"**APROVADO** - tudo certo", true, "tudo certo", true},
		{"MUDANÇAS: falta o teste de timeout", false, "falta o teste de timeout", true},
		{"mudancas - renomear a função", false, "renomear a função", true},
		{"APROVADO\n...depois de olhar melhor:\nMUDANÇAS: faltou o README", false, "faltou o README", true},
		{"Parece bom, mas não sei.", false, "", false},
	} {
		approve, note, ok := parseVerdict(c.text)
		if approve != c.approve || note != c.note || ok != c.ok {
			t.Errorf("parseVerdict(%q) = %v %q %v", c.text, approve, note, ok)
		}
	}
}

func TestTaskPromptCarriesContext(t *testing.T) {
	p := taskPrompt(tasks.Task{ID: "T007", Title: "Portar logs", Done: "go test passa", Summary: "metade feita",
		Verdicts: []string{"revisor:changes"}, History: []tasks.Event{{By: "revisor", What: "revisão: faltou teste"}}}, "Resposta do Diogo: use slog")
	for _, want := range []string{"T007", "Portar logs", "go test passa", "metade feita", "pediu mudanças", "faltou teste", "use slog", "UMA pergunta"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt sem %q:\n%s", want, p)
		}
	}
}

func TestOneLine(t *testing.T) {
	if got := oneLine("fiz isso\n\nResumo: portei o módulo  "); got != "Resumo: portei o módulo" {
		t.Fatalf("got %q", got)
	}
}
