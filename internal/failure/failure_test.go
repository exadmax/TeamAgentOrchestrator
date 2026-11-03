package failure

import (
	"strings"
	"testing"
	"time"
)

func TestClassifyPatterns(t *testing.T) {
	cases := map[Kind][]string{
		Context:   {"context length", "context window", "maximum context", "prompt is too long", "too many tokens", "input was too long", "exceeds the maximum", "exceeds the context"},
		Model:     {"model_not_found", "model not found", "modelnotfound", "ProviderModelNotFoundError", "unknown model", "invalid model", "no such model", "model x does not exist", "model x is not supported", "model x not supported"},
		Quota:     {"429", "RESOURCE_EXHAUSTED", "TerminalQuotaError", "rate-limit", "rate limit", "ratelimit", "quota", "usage limit", "limit reached", "hit your limit", "hit your usage limit", "weekly limit", "5-hour limit", "too many requests", "insufficient_quota", "insufficient funds", "insufficient credits", "credit balance", "billing", "payment required", "402", "free usage exceeded", "out of credits", "out of tokens", "exhausted your"},
		Auth:      {"401", "403", "unauthorized", "unauthorised", "invalid api key", "api key not set", "api key missing", "api key invalid", "api key is not set", "api key is missing", "api key is invalid", "not logged in", "login required", "please login", "please log in", "please signin", "please sign in", "please run /login", "not authenticated", "authentication failed", "authentication error", "authentication required"},
		Refused:   {"content policy", "content filter", "blocked by", "PROHIBITED_CONTENT", "violates a policy", "I can't help with", "I cant assist with"},
		Launch:    {"O sistema não pode encontrar o arquivo especificado.", "cannot find the file specified", "cannot find the path specified", "is not recognized as an internal or external command", "command not found"},
		Transient: {"502", "503", "504", "529", "UNAVAILABLE", "overloaded", "high demand", "bad gateway", "service unavailable", "server error", "ECONNRESET", "ECONNREFUSED", "ETIMEDOUT", "ENOTFOUND", "EAI_AGAIN", "getaddrinfo", "socket hang up", "fetch failed", "connection error", "connection reset", "connection refused", "connection closed", "network error", "network is unreachable", "time out", "timed out", "timeout", "timedout", "temporarily", "try again later", "internal error", "internal server error"},
		Unknown:   {"boom", "14290", "x401x", "15030"},
	}
	for want, texts := range cases {
		for _, text := range texts {
			t.Run(text, func(t *testing.T) {
				for _, input := range []string{text, strings.ToUpper(text)} {
					if got := Classify(1, input); got != want {
						t.Fatalf("%q: %s, esperado %s", input, got, want)
					}
				}
			})
		}
	}
	for _, tc := range []struct {
		exit int
		text string
		want Kind
	}{{0, "429", ""}, {124, "401", Timeout}, {130, "429", Interrupted}, {1, "503 quota context window", Context}, {1, "quota invalid model", Model}, {1, "401 quota", Quota}, {1, "content filter 401", Auth}, {1, "command not found content filter", Refused}, {1, "503 command not found", Launch}} {
		if got := Classify(tc.exit, tc.text); got != tc.want {
			t.Errorf("Classify(%d,%q)=%s", tc.exit, tc.text, got)
		}
	}
}
func TestParseWait(t *testing.T) {
	now := time.Date(2026, 11, 2, 14, 0, 0, 0, time.FixedZone("BRT", -3*3600))
	for _, tc := range []struct {
		text string
		want time.Duration
		hint string
	}{
		{"Please retry in 2h39m55.26s.", 2*time.Hour + 39*time.Minute + 55260*time.Millisecond, "2h39m"},
		{"try again in 2 hours 3 minutes", 2*time.Hour + 3*time.Minute, "2h3m"},
		{"resets in 4h", 4 * time.Hour, "4h0m"}, {"reset after 1 hr and 2 mins and 3 secs", time.Hour + 2*time.Minute + 3*time.Second, "1h2m"},
		{"RETRY IN 1.5 HRS 2 MIN 3 SEC", 90*time.Minute + 2*time.Minute + 3*time.Second, "1h32m"},
		{"retry in 45.2s", 45200 * time.Millisecond, "45s"}, {"retry in 3m", 3 * time.Minute, "3m"}, {"retry in 0s", 0, "0s"},
		{"5-hour limit reached ∙ resets 3pm", time.Hour, "1h0m"}, {"resets at 15:30", 90 * time.Minute, "1h30m"},
		{"resets at 13:30", 23*time.Hour + 30*time.Minute, "23h30m"}, {"resets 14:00", 24 * time.Hour, "24h0m"},
		{"try again at 12am", 10 * time.Hour, "10h0m"}, {"available again 12pm", 22 * time.Hour, "22h0m"},
		{"resets 3:15PM", 75 * time.Minute, "1h15m"}, {"available again at 16:00", 2 * time.Hour, "2h0m"},
		{"retry in 2h3m4s", 2*time.Hour + 3*time.Minute + 4*time.Second, "2h3m"}, {"retry in 2h 3months", 2 * time.Hour, "2h0m"},
	} {
		t.Run(tc.text, func(t *testing.T) {
			got, ok := ParseWait(tc.text, now)
			if !ok || got != tc.want {
				t.Fatalf("%v,%v; esperado %v", got, ok, tc.want)
			}
			if h := RetryHint(tc.text, now); h != tc.hint {
				t.Errorf("hint %q", h)
			}
		})
	}
	for _, text := range []string{"sem dica nenhuma", "retry in 2horse", "retry in 3months", "retry in 4secondsx", "retry in 5HOURSfoo", "retry in 1ms", "resets at 15:99"} {
		if d, ok := ParseWait(text, now); ok {
			t.Errorf("%q aceito: %v", text, d)
		}
		if RetryHint(text, now) != "" {
			t.Errorf("hint inesperado: %q", text)
		}
	}
}
func TestHelpers(t *testing.T) {
	for _, tc := range []struct {
		hint string
		want time.Duration
		ok   bool
	}{{"2h40m", 165 * time.Minute, true}, {"45s", 105 * time.Second, true}, {"59m", time.Hour, true}, {"1h", 65 * time.Minute, true}, {"", 0, false}, {"0s", 0, false}, {"nada", 0, false}, {"999999999999999999999h", 0, false}} {
		d, ok := HintDuration(tc.hint)
		if d != tc.want || ok != tc.ok {
			t.Errorf("HintDuration(%q)=%v,%v", tc.hint, d, ok)
		}
	}
	for _, tc := range []struct {
		d    time.Duration
		want string
	}{{125 * time.Minute, "2h05m"}, {187 * time.Second, "3m07s"}, {0, "0m00s"}, {59900 * time.Millisecond, "0m59s"}} {
		if FormatDuration(tc.d) != tc.want {
			t.Error(tc)
		}
	}
	for _, tc := range []struct {
		text string
		n    int
		ok   bool
	}{{"x\ntokens used\n12,345\n", 12345, true}, {"TOKENS USED: 12.345", 12345, true}, {"tokens used: 0", 0, true}, {"nada", 0, false}, {"tokens used: ...", 0, false}} {
		n, ok := ParseTokensUsed(tc.text)
		if n != tc.n || ok != tc.ok {
			t.Error(tc)
		}
	}
	if LastQuestion("a\nQual executável Python devo usar? Posso ler?") != "a Qual executável Python devo usar? Posso ler?" {
		t.Fatal("pergunta")
	}
	if LastQuestion("feito\nTudo certo.") != "" || LastQuestion("") != "" {
		t.Fatal("não pergunta")
	}
	if LastQuestion("ignore\na\n\nb\nc?") != "a b c?" {
		t.Fatal("últimas linhas")
	}
	if len([]rune(LastQuestion(strings.Repeat("á", 600)+"?"))) != 500 {
		t.Fatal("limite unicode")
	}
	if ErrorModel("model: gemini-3.8-flash") != "gemini-3.8-flash" || ErrorModel("model: provider/model_1.2") != "provider/model_1.2" || ErrorModel("Model: X") != "" || ErrorModel("nada") != "" {
		t.Fatal("modelo")
	}
}
func TestPolicies(t *testing.T) {
	for _, k := range []Kind{Context, Model, Quota, Auth, Refused, Launch, Transient, Timeout, Interrupted, Unknown} {
		if Help(k) == "" {
			t.Errorf("sem ajuda %s", k)
		}
		wantAuto := k == Transient || k == Timeout
		wantSwitch := wantAuto || k == Quota || k == Model || k == Refused || k == Launch
		wantAsk := wantAuto || k == Unknown
		if AutoRetry(k) != wantAuto || Switchable(k) != wantSwitch || AskRetry(k) != wantAsk {
			t.Error(k)
		}
		wantAction := k == Quota || k == Auth || k == Model || k == Context || k == Refused || k == Launch
		if (Action(k) != "") != wantAction {
			t.Error(k)
		}
	}
	if Help("") != "" || Action("") != "" {
		t.Fatal("política vazia")
	}
	if Heartbeat != 10*time.Second || IdleKill != 300*time.Second || len(RetryWaits) != 3 || RetryWaits[0] != 20*time.Second || RetryWaits[1] != time.Minute || RetryWaits[2] != 2*time.Minute || DefaultSwitch[Quota] != 24*time.Hour {
		t.Fatal("constantes")
	}
}
