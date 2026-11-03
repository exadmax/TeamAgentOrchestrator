package store

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Store, *time.Time) {
	t.Helper()
	now := time.Date(2026, 11, 2, 14, 0, 0, 0, time.FixedZone("BRT", -3*3600))
	s := Open(filepath.Join(t.TempDir(), "nested", ".tao"))
	s.Now = func() time.Time { return now }
	return s, &now
}
func must(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func TestSessionsAndUsage(t *testing.T) {
	s, now := fixture(t)
	rows, e := s.Sessions()
	must(t, e)
	if len(rows) != 0 {
		t.Fatal(rows)
	}
	first := Session{Run: "r1", Agent: "a", Client: "gemini", Model: "old", Mode: "work", Task: "T000", Start: *now, End: now.Add(109 * time.Second), Exit: 1, Error: "quota", Hint: "2h40m", WaitSecs: 9900, Usage: Usage{Input: 100, Output: 5, CacheRead: 30, CacheWrite: 20, CostUSD: 0.1}, Log: "logs/a", Status: "error", Question: "Qual chave?", Summary: "parei", Argv: "gemini"}
	second := Session{Run: "r2", Agent: "b", Client: "codex", Model: "m", Start: now.Add(-time.Hour), End: *now, Usage: Usage{Input: 12000, Output: 345, CostUSD: 1}}
	third := first
	third.Run = "r3"
	third.Model = "new"
	third.Start = now.Add(time.Minute)
	third.End = now.Add(2 * time.Minute)
	third.Error = ""
	third.Exit = 0
	for _, r := range []Session{first, second, third} {
		must(t, s.RecordSession(r))
	}
	rows, e = s.Sessions()
	must(t, e)
	if !reflect.DeepEqual(rows[0].Usage, first.Usage) || !rows[0].Start.Equal(first.Start) || rows[0].Question != first.Question {
		t.Fatal(rows)
	}
	b, e := os.ReadFile(s.path("sessions.jsonl"))
	must(t, e)
	if !strings.Contains(string(b), `"cache_read":30`) || !strings.Contains(string(b), `"wait_secs":9900`) {
		t.Fatal(string(b))
	}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if !json.Valid([]byte(line)) {
			t.Fatal("JSONL inválido")
		}
	}
	f, e := os.OpenFile(s.path("sessions.jsonl"), os.O_APPEND|os.O_WRONLY, 0600)
	must(t, e)
	_, e = f.WriteString("broken\nnull\n[]\n{bad json}\n")
	must(t, e)
	must(t, f.Close())
	rows, e = s.Sessions()
	must(t, e)
	if len(rows) != 3 {
		t.Fatal(rows)
	}
	by := UsageBy(rows, func(x Session) string { return x.Client }, "2026-11-02")
	if len(by) != 2 || by[0].Key != "codex" || by[0].Usage.Total() != 12345 || by[1].Sessions != 2 || by[1].Model != "new" || by[1].Failures["quota"] != 1 || by[1].Duration != 169*time.Second || by[1].Usage.CacheRead != 60 || by[1].Usage.CacheWrite != 40 || by[1].Usage.CostUSD != 0.2 {
		t.Fatal(by)
	}
	// O último modelo depende do horário da sessão, mesmo com entrada fora de ordem.
	reversed := UsageBy([]Session{third, first}, func(x Session) string { return x.Client }, "")
	if reversed[0].Model != "new" {
		t.Fatal(reversed)
	}
	if len(UsageBy(rows, func(x Session) string { return x.Agent }, "2026-11-03")) != 0 {
		t.Fatal("filtro de dia")
	}
	if len(UsageBy(rows, func(x Session) string { return x.Agent }, "")) != 2 {
		t.Fatal("agrupamento")
	}
	invalid := first
	invalid.Usage.CostUSD = math.NaN()
	if s.RecordSession(invalid) == nil {
		t.Fatal("NaN aceito")
	}
	after, e := os.ReadFile(s.path("sessions.jsonl"))
	must(t, e)
	if !strings.HasPrefix(string(after), string(b)) {
		t.Fatal("arquivo anterior alterado")
	}
}
func TestClocks(t *testing.T) {
	s, now := fixture(t)
	if len(s.ActiveClocks()) != 0 || len(s.PopExpired()) != 0 {
		t.Fatal("inicial")
	}
	must(t, s.RecordClock("gemini", "", "gemini-3.8-flash", time.Hour, "quota", false))
	until, ok := s.ClockUntil("gemini", "")
	if !ok || !until.Equal(now.Add(time.Hour)) {
		t.Fatal(until, ok)
	}
	active := s.ActiveClocks()
	if len(active) != 1 || active[0].Source != "informado" || active[0].Model != "gemini-3.8-flash" || active[0].Left != "1h00m" || !active[0].Recorded.Equal(*now) {
		t.Fatal(active)
	}
	must(t, s.SetClock("gemini", "gemini-3.8-flash", 2*time.Hour, "transient", "estimado"))
	if len(s.clocks()) != 2 {
		t.Fatal("duplicou relógio")
	}
	*now = now.Add(time.Hour)
	if _, ok = s.ClockUntil("gemini", ""); ok {
		t.Fatal("alias vencido")
	}
	if gone := s.PopExpired(); len(gone) != 0 {
		t.Fatal("alias devolvido", gone)
	}
	if len(s.clocks()) != 1 {
		t.Fatal(s.clocks())
	}
	*now = now.Add(time.Hour)
	if len(s.ActiveClocks()) != 0 {
		t.Fatal("ativo no instante de expiração")
	}
	gone := s.PopExpired()
	if len(gone) != 1 || gone[0].Reason != "transient" {
		t.Fatal(gone)
	}
	if len(s.PopExpired()) != 0 || len(s.clocks()) != 0 {
		t.Fatal("não removeu")
	}
	must(t, s.RecordClock("codex", "explicit", "ignored", time.Minute, "timeout", true))
	must(t, s.RecordClock("claude", "", "", time.Minute, "quota", false))
	active = s.ActiveClocks()
	if len(active) != 2 || active[0].Model != "explicit" || active[0].Source != "estimado" || active[1].Model != "padrão" {
		t.Fatal(active)
	}
	if _, ok = s.ClockUntil("other", ""); ok {
		t.Fatal("outro cliente")
	}
	must(t, s.ClearClocks())
	must(t, s.ClearClocks())
	if len(s.ActiveClocks()) != 0 {
		t.Fatal("limpeza")
	}
}
func TestOverrides(t *testing.T) {
	s, now := fixture(t)
	if s.AutoSwitch() || len(s.Overrides()) != 0 {
		t.Fatal("inicial")
	}
	must(t, s.SetAuto(true))
	must(t, s.SetOverride("codex", "global", 0, ""))
	must(t, s.SetOverride("codex", "task", time.Hour, "T002"))
	ov, ok := s.OverrideFor("codex", "T002")
	if !ok || ov.Model != "task" || ov.Until == nil {
		t.Fatal(ov)
	}
	ov, ok = s.OverrideFor("codex", "other")
	if !ok || ov.Model != "global" || ov.Until != nil {
		t.Fatal(ov)
	}
	*now = now.Add(time.Hour)
	ov, ok = s.OverrideFor("codex", "T002")
	if !ok || ov.Model != "global" {
		t.Fatal(ov)
	}
	data := s.models()
	if _, ok = data["T002:codex"]; ok {
		t.Fatal("não regravou expiração")
	}
	if !s.AutoSwitch() {
		t.Fatal("perdeu _auto")
	}
	must(t, s.SetAuto(false))
	if s.AutoSwitch() {
		t.Fatal("não desligou")
	}
	must(t, s.ClearKey("codex"))
	must(t, s.ClearKey("missing"))
	if _, ok = s.OverrideFor("codex", ""); ok {
		t.Fatal("não removeu")
	}
	must(t, s.SetOverride("gemini", "temporary", time.Second, ""))
	*now = now.Add(time.Second)
	if len(s.Overrides()) != 0 {
		t.Fatal("troca vencida")
	}
	if s.AutoSwitch() {
		t.Fatal("auto entrou em overrides")
	}
}
func TestMark(t *testing.T) {
	s, now := fixture(t)
	for _, phase := range Phases {
		must(t, s.Mark("T000", phase, "ok", PhaseInfo{Agent: "b", Client: "codex", Model: "m", Run: "r1", Detail: "done", Verdict: "approve", VerdictsSeen: 2}))
	}
	if len(s.State("T000").Phases) != 7 {
		t.Fatal("etapas")
	}
	must(t, s.Mark("T000", "work", "failed", PhaseInfo{Client: "gemini", Model: "m2", Run: "r2", Detail: "quota"}))
	st := s.State("T000")
	p := st.Phases["work"]
	if len(st.Phases) != 7 || p.Attempts != 2 || len(p.History) != 2 || p.Client != "gemini" || p.History[0].Client != "codex" || p.History[1].Status != "failed" || !p.At.Equal(*now) {
		t.Fatal(st, p)
	}
	must(t, s.Mark("T000", "work", "pending", PhaseInfo{}))
	st = s.State("T000")
	if len(st.Phases) != 2 || st.Phases["work"].Attempts != 2 {
		t.Fatal(st)
	}
	for _, status := range []string{"running", "skipped", "pending"} {
		must(t, s.Mark("T000", "review", status, PhaseInfo{}))
	}
	if p = s.State("T000").Phases["review"]; p.Attempts != 0 || len(p.History) != 0 {
		t.Fatal(p)
	}
	must(t, s.Mark("T000", "work", "blocked", PhaseInfo{Detail: "Qual chave?"}))
	if len(s.State("T000").Phases) != 3 {
		t.Fatal("blocked invalidou etapas")
	}
	must(t, s.Mark("T000", "work", "ok", PhaseInfo{Client: "opencode", Run: "r3"}))
	st = s.State("T000")
	if len(st.Phases) != 2 || st.Phases["work"].Attempts != 4 {
		t.Fatal(st)
	}
	for i := 0; i < 25; i++ {
		*now = now.Add(time.Second)
		must(t, s.Mark("T000", "review", "blocked", PhaseInfo{Run: now.Format(time.RFC3339)}))
	}
	p = s.State("T000").Phases["review"]
	if p.Attempts != 25 || len(p.History) != 20 || p.History[0].Run != now.Add(-19*time.Second).Format(time.RFC3339) {
		t.Fatal(p)
	}
	must(t, s.ResetState("T000"))
	must(t, s.ResetState("T000"))
	if len(s.State("T000").Phases) != 0 {
		t.Fatal("reset")
	}
}
func TestRebuildState(t *testing.T) {
	s, _ := fixture(t)
	for _, r := range []Session{{Task: "T000", Mode: "work", Client: "gemini", Model: "m1", Run: "r1", Exit: 1, Error: "quota"}, {Task: "T000", Mode: "work", Client: "opencode", Model: "big", Run: "r2", Exit: 1, Error: "unknown"}, {Task: "T000", Mode: "review", Exit: 0, Status: "blocked"}, {Task: "other", Mode: "work", Exit: 0}} {
		must(t, s.RecordSession(r))
	}
	st := s.State("T000")
	p := st.Phases["work"]
	if p.Status != "failed" || p.Attempts != 2 || p.Client != "opencode" || p.History[0].Client != "gemini" || st.Phases["review"].Status != "blocked" {
		t.Fatal(st)
	}
	must(t, os.MkdirAll(s.path("state"), directoryMode))
	must(t, os.WriteFile(s.path("state/T000.json"), []byte("broken"), 0600))
	if s.State("T000").Phases["work"].Attempts != 2 {
		t.Fatal("não reconstruiu")
	}
	must(t, s.Mark("T000", "work", "ok", PhaseInfo{}))
	if _, ok := s.State("T000").Phases["review"]; ok {
		t.Fatal("revisão antiga")
	}
	must(t, s.ResetState("T000"))
	if s.State("T000").Phases["work"].Attempts != 2 {
		t.Fatal("reset deve permitir reconstruir")
	}
}
func TestInvalidFilesAndWriteErrors(t *testing.T) {
	s, _ := fixture(t)
	must(t, os.MkdirAll(s.Dir, directoryMode))
	for _, name := range []string{"models.json", "clocks.json"} {
		must(t, os.WriteFile(s.path(name), []byte("invalid"), 0600))
	}
	if len(s.Overrides()) != 0 || len(s.ActiveClocks()) != 0 || s.AutoSwitch() {
		t.Fatal("arquivo inválido")
	}
	for _, task := range []string{"", "..", "../escape", "a/b", "a\\b", "C:foo"} {
		if s.Mark(task, "work", "ok", PhaseInfo{}) == nil || s.ResetState(task) == nil {
			t.Fatal(task)
		}
	}
	blocked := filepath.Join(t.TempDir(), "file")
	must(t, os.WriteFile(blocked, []byte("original"), 0600))
	bad := Open(filepath.Join(blocked, "child"))
	for _, e := range []error{bad.RecordSession(Session{}), bad.SetClock("c", "m", time.Hour, "q", "s"), bad.SetOverride("c", "m", 0, ""), bad.SetAuto(true), bad.Mark("T", "work", "ok", PhaseInfo{})} {
		if e == nil {
			t.Fatal("erro de escrita omitido")
		}
	}
	must(t, os.Mkdir(s.path("sessions.jsonl"), directoryMode))
	if _, e := s.Sessions(); e == nil {
		t.Fatal("erro de leitura omitido")
	}
	b, e := os.ReadFile(blocked)
	must(t, e)
	if string(b) != "original" {
		t.Fatal("arquivo alterado")
	}
}
func TestConcurrentUpdates(t *testing.T) {
	s, _ := fixture(t)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := s.RecordSession(Session{Client: "codex"}); e != nil {
				t.Error(e)
			}
			if e := s.Mark("T", "review", "ok", PhaseInfo{}); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	rows, e := s.Sessions()
	must(t, e)
	if len(rows) != 10 || s.State("T").Phases["review"].Attempts != 10 {
		t.Fatal("atualizações perdidas")
	}
	files, e := filepath.Glob(s.path(".tao-*"))
	must(t, e)
	if len(files) != 0 {
		t.Fatal(files)
	}
}

func TestRebuildStatusPriority(t *testing.T) {
	s, _ := fixture(t)
	for _, tc := range []struct {
		task, status, failure, want string
		exit                        int
	}{
		{"error", "blocked", "quota", "failed", 0},
		{"blocked", "blocked", "", "blocked", 1},
		{"failed", "", "", "failed", 1},
		{"ok", "", "", "ok", 0},
	} {
		must(t, s.RecordSession(Session{Task: tc.task, Mode: "work", Status: tc.status, Error: tc.failure, Exit: tc.exit}))
		if got := s.State(tc.task).Phases["work"].Status; got != tc.want {
			t.Errorf("%s: %s, esperado %s", tc.task, got, tc.want)
		}
	}
}
