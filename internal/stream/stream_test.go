package stream

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
)

// fixture parses a real capture from testdata/stream and returns every event.
func fixture(t *testing.T, client string) []Event {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "testdata", "stream", client+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<22)
	for sc.Scan() {
		out = append(out, Parse(client, sc.Bytes())...)
	}
	return out
}

func kinds(evs []Event) []Kind {
	out := make([]Kind, len(evs))
	for i, e := range evs {
		out[i] = e.Kind
	}
	return out
}

func sum(evs []Event) Tokens {
	var t Tokens
	for _, e := range evs {
		if e.Usage != nil {
			t = t.Add(*e.Usage)
		}
	}
	return t
}

func TestClaudeFixture(t *testing.T) {
	evs := fixture(t, "claude")
	if got := kinds(evs); len(got) != 3 || got[0] != Init || got[1] != Text || got[2] != Result {
		t.Fatalf("kinds = %v", got)
	}
	if evs[0].Model != "claude-haiku-5-5" || evs[1].Text != "ok" || evs[2].IsError {
		t.Fatalf("events = %+v", evs)
	}
	u := sum(evs)
	if u.Output != 4 || u.Input != 2+25343+27214 || u.CacheRead != 27214 || u.CostUSD <= 0 {
		t.Fatalf("usage = %+v", u)
	}
}

func TestCodexFixture(t *testing.T) {
	evs := fixture(t, "codex")
	if got := kinds(evs); len(got) != 3 || got[0] != Init || got[1] != Text || got[2] != Result {
		t.Fatalf("kinds = %v", got)
	}
	if u := sum(evs); u.Input != 25367 || u.Output != 5 || u.CacheRead != 13440 {
		t.Fatalf("usage = %+v", u)
	}
}

func TestOpenCodeFixture(t *testing.T) {
	evs := fixture(t, "opencode")
	if len(evs) != 1 || evs[0].Kind != Text || evs[0].Text != "ok" || evs[0].Session == "" {
		t.Fatalf("events = %+v", evs)
	}
}

func TestToolsAndErrors(t *testing.T) {
	cases := []struct {
		client, line string
		want         Event
	}{
		{"claude", `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}`,
			Event{Kind: Tool, Tool: "Bash", Text: "go test ./..."}},
		{"claude", `{"type":"user","message":{"content":[{"type":"tool_result","is_error":true,"content":"exit 1"}]}}`,
			Event{Kind: Error, Text: "exit 1", IsError: true}},
		{"claude", `{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","rateLimitType":"five_hour"}}`,
			Event{Kind: Info, Text: "limite de uso: rejected (five_hour)"}},
		{"codex", `{"type":"item.completed","item":{"type":"command_execution","command":"bash -lc ls","exit_code":2}}`,
			Event{Kind: Tool, Tool: "shell", Text: "bash -lc ls", IsError: true}},
		{"codex", `{"type":"item.completed","item":{"type":"file_change","changes":[{"path":"a.go"},{"path":"b.go"}]}}`,
			Event{Kind: Tool, Tool: "edit", Text: "a.go, b.go"}},
		{"codex", `{"type":"turn.failed","error":{"message":"You've hit your usage limit. Try again in 2 hours."}}`,
			Event{Kind: Error, Text: "You've hit your usage limit. Try again in 2 hours.", IsError: true}},
		{"codex", `{"type":"error","message":"stream disconnected"}`,
			Event{Kind: Error, Text: "stream disconnected", IsError: true}},
		{"opencode", `{"type":"tool_use","part":{"tool":"bash","state":{"status":"completed","title":"go build"}}}`,
			Event{Kind: Tool, Tool: "bash", Text: "go build"}},
		{"opencode", `{"type":"error","error":{"name":"ProviderModelNotFoundError","data":{"message":"model x not found"}}}`,
			Event{Kind: Error, Text: "model x not found", IsError: true}},
		{"gemini", `{"type":"message","role":"assistant","content":"oi"}`, Event{Kind: Text, Text: "oi"}},
		{"claude", `Error: not logged in`, Event{Kind: Raw, Text: "Error: not logged in"}},
	}
	for _, c := range cases {
		got := Parse(c.client, []byte(c.line))
		if len(got) != 1 || got[0].Kind != c.want.Kind || got[0].Text != c.want.Text || got[0].Tool != c.want.Tool ||
			got[0].IsError != c.want.IsError {
			t.Errorf("%s %s\n got  %+v\n want %+v", c.client, c.line, got, c.want)
		}
	}
}

func TestOpenCodeStepUsage(t *testing.T) {
	line := `{"type":"step_finish","sessionID":"s","part":{"cost":0.01,"tokens":{"input":100,"output":20,"reasoning":5,"cache":{"read":50,"write":10}}}}`
	got := Parse("opencode", []byte(line))
	if len(got) != 1 || got[0].Kind != Usage {
		t.Fatalf("got %+v", got)
	}
	if u := *got[0].Usage; u.Input != 160 || u.Output != 25 || u.CacheRead != 50 || u.CostUSD != 0.01 {
		t.Fatalf("usage = %+v", u)
	}
}

func TestQuietLines(t *testing.T) {
	for _, c := range []struct{ client, line string }{
		{"codex", ""}, {"codex", "   "}, {"codex", `{"type":"turn.started"}`}, {"codex", "Reading additional input from stdin..."},
		{"claude", `{"type":"system","subtype":"hook_started"}`},
	} {
		if got := Parse(c.client, []byte(c.line)); len(got) != 0 {
			t.Errorf("%s %q -> %+v", c.client, c.line, got)
		}
	}
}

func TestClipLongTool(t *testing.T) {
	long := make([]byte, 400)
	for i := range long {
		long[i] = 'a'
	}
	got := Parse("codex", []byte(`{"type":"item.completed","item":{"type":"command_execution","command":"`+string(long)+`"}}`))
	if r := []rune(got[0].Text); len(r) != maxToolText || r[len(r)-1] != '…' {
		t.Fatalf("len = %d", len(r))
	}
}
