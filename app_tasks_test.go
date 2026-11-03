package main

import (
	"strings"
	"testing"

	"github.com/exadmax/TeamAgentOrchestrator/internal/i18n"
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
		{"Checked the tests.\nAPPROVED", true, "", true},
		{"> **CHANGES:** missing the timeout test", false, "missing the timeout test", true},
		{"APPROVED\nthen, looking again:\nCHANGES: README is missing", false, "README is missing", true},
	} {
		approve, note, ok := parseVerdict(c.text)
		if approve != c.approve || note != c.note || ok != c.ok {
			t.Errorf("parseVerdict(%q) = %v %q %v", c.text, approve, note, ok)
		}
	}
}

// The prompt follows the language: English by default, Portuguese when chosen.
func TestTaskPromptCarriesContext(t *testing.T) {
	defer i18n.Set(i18n.EN)
	tk := tasks.Task{ID: "T007", Title: "Portar logs", Done: "go test passa", Summary: "metade feita",
		Verdicts: []string{"revisor:changes"}, History: []tasks.Event{{By: "revisor", What: "revisão: faltou teste"}}}
	for lang, phrases := range map[string][]string{
		i18n.EN: {"asked for changes", "ONE question", "Done when"},
		i18n.PT: {"pediu mudanças", "UMA pergunta", "Pronto quando"},
	} {
		_ = i18n.Set(lang)
		p := taskPrompt(tk, "answer: use slog")
		for _, want := range append([]string{"T007", "Portar logs", "go test passa", "metade feita", "faltou teste", "use slog"}, phrases...) {
			if !strings.Contains(p, want) {
				t.Errorf("%s prompt without %q:\n%s", lang, want, p)
			}
		}
	}
}

func TestOneLine(t *testing.T) {
	if got := oneLine("fiz isso\n\nResumo: portei o módulo  "); got != "Resumo: portei o módulo" {
		t.Fatalf("got %q", got)
	}
}
