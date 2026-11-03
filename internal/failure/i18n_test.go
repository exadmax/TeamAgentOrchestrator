package failure

import (
	"github.com/exadmax/TeamAgentOrchestrator/internal/i18n"
	"testing"
)

func TestHelpFollowsLanguage(t *testing.T) {
	defer i18n.Set(i18n.EN)
	if err := i18n.Set(i18n.PT); err != nil {
		t.Fatal(err)
	}
	if got := Help(Quota); got != "Cota, limite de uso ou créditos do modelo esgotados." {
		t.Fatal(got)
	}
	if err := i18n.Set(i18n.EN); err != nil {
		t.Fatal(err)
	}
	if got := Help(Quota); got != "The model quota, usage limit or credits are exhausted." {
		t.Fatal(got)
	}
	if Action(Timeout) != "" || Help(Kind("missing")) != "" {
		t.Fatal("unknown policies should remain empty")
	}
}
