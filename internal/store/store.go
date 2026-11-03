// Package store persiste o estado local do projeto, sem dependências de interface ou rede.
package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const historyLimit = 20
const directoryMode = 0755

// Serializa operações de leitura-modificação-escrita, inclusive entre Stores do mesmo diretório.
var diskMu sync.Mutex

type Store struct {
	Dir string
	Now func() time.Time
}

func Open(dir string) *Store { return &Store{Dir: dir, Now: time.Now} }
func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
func (s *Store) path(name string) string { return filepath.Join(s.Dir, name) }
func readJSON(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func writeAtomic(path string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), directoryMode); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".tao-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
func writeJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return writeAtomic(path, append(b, '\n'))
}
func remove(path string) error {
	e := os.Remove(path)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	return e
}

type Usage struct {
	Input      int     `json:"input"`
	Output     int     `json:"output"`
	CacheRead  int     `json:"cache_read"`
	CacheWrite int     `json:"cache_write"`
	CostUSD    float64 `json:"cost_usd"`
}

func (u Usage) Total() int { return u.Input + u.Output }
func (u *Usage) add(v Usage) {
	u.Input += v.Input
	u.Output += v.Output
	u.CacheRead += v.CacheRead
	u.CacheWrite += v.CacheWrite
	u.CostUSD += v.CostUSD
}

type Session struct {
	Run      string    `json:"run"`
	Agent    string    `json:"agent"`
	Client   string    `json:"client"`
	Model    string    `json:"model"`
	Mode     string    `json:"mode"`
	Task     string    `json:"task"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Exit     int       `json:"exit"`
	Error    string    `json:"error"`
	Hint     string    `json:"hint"`
	WaitSecs int       `json:"wait_secs"`
	Usage    Usage     `json:"usage"`
	Log      string    `json:"log"`
	Status   string    `json:"status"`
	Question string    `json:"question"`
	Summary  string    `json:"summary"`
	Argv     string    `json:"argv"`
}

// RecordSession acrescenta uma linha preservando o arquivo anterior em caso de falha.
func (s *Store) RecordSession(x Session) error {
	diskMu.Lock()
	defer diskMu.Unlock()
	b, e := os.ReadFile(s.path("sessions.jsonl"))
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	line, e := json.Marshal(x)
	if e != nil {
		return e
	}
	if len(b) > 0 && b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	b = append(b, line...)
	return writeAtomic(s.path("sessions.jsonl"), append(b, '\n'))
}
func (s *Store) Sessions() ([]Session, error) {
	b, e := os.ReadFile(s.path("sessions.jsonl"))
	if errors.Is(e, os.ErrNotExist) {
		return []Session{}, nil
	}
	if e != nil {
		return nil, e
	}
	rows := []Session{}
	for _, line := range bytes.Split(b, []byte{'\n'}) {
		var x Session
		if len(bytes.TrimSpace(line)) == 0 || bytes.TrimSpace(line)[0] != '{' {
			continue
		}
		if json.Unmarshal(line, &x) == nil {
			rows = append(rows, x)
		}
	}
	return rows, nil
}

type UsageRow struct {
	Key      string
	Sessions int
	Duration time.Duration
	Usage    Usage
	Model    string
	Failures map[string]int
}

func UsageBy(rows []Session, key func(Session) string, day string) []UsageRow {
	groups := map[string]*UsageRow{}
	latest := map[string]time.Time{}
	for _, x := range rows {
		if day != "" && x.Start.Format("2006-01-02") != day {
			continue
		}
		k := key(x)
		g := groups[k]
		if g == nil {
			g = &UsageRow{Key: k, Failures: map[string]int{}}
			groups[k] = g
		}
		g.Sessions++
		if !x.Start.IsZero() && !x.End.IsZero() {
			g.Duration += x.End.Sub(x.Start)
		}
		g.Usage.add(x.Usage)
		if !x.Start.Before(latest[k]) {
			g.Model = x.Model
			latest[k] = x.Start
		}
		if x.Error != "" {
			g.Failures[x.Error]++
		}
	}
	out := make([]UsageRow, 0, len(groups))
	for _, g := range groups {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Usage.Total() == out[j].Usage.Total() {
			return out[i].Key < out[j].Key
		}
		return out[i].Usage.Total() > out[j].Usage.Total()
	})
	return out
}

type Clock struct {
	Client   string    `json:"client"`
	Model    string    `json:"model"`
	Until    time.Time `json:"until"`
	Left     string    `json:"left"`
	Recorded time.Time `json:"recorded"`
	Reason   string    `json:"reason"`
	Source   string    `json:"source"`
}

func (s *Store) clocks() []Clock {
	rows := []Clock{}
	if readJSON(s.path("clocks.json"), &rows) != nil {
		return []Clock{}
	}
	return rows
}
func formatDuration(d time.Duration) string {
	n := int64(d / time.Second)
	if n >= 3600 {
		return fmt.Sprintf("%dh%02dm", n/3600, n%3600/60)
	}
	return fmt.Sprintf("%dm%02ds", n/60, n%60)
}
func replaceClock(rows []Clock, x Clock) []Clock {
	out := make([]Clock, 0, len(rows)+1)
	for _, r := range rows {
		if r.Client != x.Client || r.Model != x.Model {
			out = append(out, r)
		}
	}
	return append(out, x)
}
func makeClock(client, model string, d time.Duration, reason, source string, now time.Time) Clock {
	return Clock{Client: client, Model: model, Until: now.Add(d), Left: formatDuration(d), Recorded: now, Reason: reason, Source: source}
}
func (s *Store) SetClock(client, model string, d time.Duration, reason, source string) error {
	diskMu.Lock()
	defer diskMu.Unlock()
	return writeJSON(s.path("clocks.json"), replaceClock(s.clocks(), makeClock(client, model, d, reason, source, s.now())))
}
func (s *Store) RecordClock(client, model, errorModel string, d time.Duration, reason string, estimated bool) error {
	diskMu.Lock()
	defer diskMu.Unlock()
	real := model
	if real == "" {
		real = errorModel
	}
	if real == "" {
		real = "padrão"
	}
	source := "informado"
	if estimated {
		source = "estimado"
	}
	now := s.now()
	rows := replaceClock(s.clocks(), makeClock(client, real, d, reason, source, now))
	if model == "" && real != "padrão" {
		rows = replaceClock(rows, makeClock(client, "padrão", d, reason, "alias", now))
	}
	return writeJSON(s.path("clocks.json"), rows)
}
func (s *Store) ClockUntil(client, model string) (time.Time, bool) {
	if model == "" {
		model = "padrão"
	}
	now := s.now()
	for _, r := range s.clocks() {
		if r.Client == client && r.Model == model && r.Until.After(now) {
			return r.Until, true
		}
	}
	return time.Time{}, false
}
func (s *Store) ActiveClocks() []Clock {
	now := s.now()
	out := []Clock{}
	for _, r := range s.clocks() {
		if r.Source != "alias" && r.Until.After(now) {
			out = append(out, r)
		}
	}
	return out
}
func (s *Store) PopExpired() []Clock {
	diskMu.Lock()
	defer diskMu.Unlock()
	now := s.now()
	live, gone := []Clock{}, []Clock{}
	changed := false
	for _, r := range s.clocks() {
		if r.Until.After(now) {
			live = append(live, r)
		} else {
			changed = true
			if r.Source != "alias" {
				gone = append(gone, r)
			}
		}
	}
	if changed {
		if writeJSON(s.path("clocks.json"), live) != nil {
			return nil
		}
	}
	return gone
}
func (s *Store) ClearClocks() error {
	diskMu.Lock()
	defer diskMu.Unlock()
	return remove(s.path("clocks.json"))
}

type Override struct {
	Model string     `json:"model"`
	Until *time.Time `json:"until"`
}

func (s *Store) models() map[string]json.RawMessage {
	data := map[string]json.RawMessage{}
	if readJSON(s.path("models.json"), &data) != nil || data == nil {
		return map[string]json.RawMessage{}
	}
	return data
}
func (s *Store) Overrides() map[string]Override {
	diskMu.Lock()
	defer diskMu.Unlock()
	data := s.models()
	out := map[string]Override{}
	now := s.now()
	changed := false
	for k, b := range data {
		if strings.HasPrefix(k, "_") {
			continue
		}
		var ov Override
		if len(b) == 0 || b[0] != '{' || json.Unmarshal(b, &ov) != nil {
			continue
		}
		if ov.Until != nil && !ov.Until.After(now) {
			delete(data, k)
			changed = true
		} else {
			out[k] = ov
		}
	}
	if changed {
		_ = writeJSON(s.path("models.json"), data)
	}
	return out
}
func (s *Store) OverrideFor(client, task string) (Override, bool) {
	live := s.Overrides()
	if task != "" {
		if ov, ok := live[task+":"+client]; ok {
			return ov, true
		}
	}
	ov, ok := live[client]
	return ov, ok
}
func (s *Store) SetOverride(client, model string, d time.Duration, task string) error {
	diskMu.Lock()
	defer diskMu.Unlock()
	data := s.models()
	ov := Override{Model: model}
	if d != 0 {
		at := s.now().Add(d)
		ov.Until = &at
	}
	k := client
	if task != "" {
		k = task + ":" + client
	}
	b, e := json.Marshal(ov)
	if e != nil {
		return e
	}
	data[k] = b
	return writeJSON(s.path("models.json"), data)
}
func (s *Store) ClearKey(key string) error {
	diskMu.Lock()
	defer diskMu.Unlock()
	data := s.models()
	if _, ok := data[key]; !ok {
		return nil
	}
	delete(data, key)
	return writeJSON(s.path("models.json"), data)
}
func (s *Store) AutoSwitch() bool {
	var on bool
	_ = json.Unmarshal(s.models()["_auto"], &on)
	return on
}
func (s *Store) SetAuto(on bool) error {
	diskMu.Lock()
	defer diskMu.Unlock()
	data := s.models()
	b, _ := json.Marshal(on)
	data["_auto"] = b
	return writeJSON(s.path("models.json"), data)
}

var Phases = []string{"start", "work", "submit", "pr", "review", "verdict", "merge"}

type Attempt struct {
	Status string    `json:"status"`
	Agent  string    `json:"agent"`
	Client string    `json:"client"`
	Model  string    `json:"model"`
	Run    string    `json:"run"`
	Detail string    `json:"detail"`
	At     time.Time `json:"at"`
}
type PhaseRecord struct {
	Status       string    `json:"status"`
	At           time.Time `json:"at"`
	Attempts     int       `json:"attempts"`
	History      []Attempt `json:"history"`
	Agent        string    `json:"agent"`
	Client       string    `json:"client"`
	Model        string    `json:"model"`
	Run          string    `json:"run"`
	Detail       string    `json:"detail"`
	Verdict      string    `json:"verdict"`
	VerdictsSeen int       `json:"verdicts_seen"`
}
type TaskState struct {
	Phases map[string]*PhaseRecord `json:"phases"`
}
type PhaseInfo struct {
	Agent, Client, Model, Run, Detail, Verdict string
	VerdictsSeen                               int
}

func taskFile(task string) (string, error) {
	if task == "" || task == "." || task == ".." || strings.ContainsAny(task, "/\\:") {
		return "", fmt.Errorf("identificador de tarefa inválido: %q", task)
	}
	return filepath.Join("state", task+".json"), nil
}
func apply(st *TaskState, phase, status string, info PhaseInfo, now time.Time) {
	p := st.Phases[phase]
	if p == nil {
		p = &PhaseRecord{History: []Attempt{}}
		st.Phases[phase] = p
	}
	if info.Agent != "" {
		p.Agent = info.Agent
	}
	if info.Client != "" {
		p.Client = info.Client
	}
	if info.Model != "" {
		p.Model = info.Model
	}
	if info.Run != "" {
		p.Run = info.Run
	}
	if info.Detail != "" {
		p.Detail = info.Detail
	}
	if info.Verdict != "" {
		p.Verdict = info.Verdict
	}
	p.VerdictsSeen = info.VerdictsSeen
	p.Status = status
	p.At = now
	if status == "ok" || status == "failed" || status == "blocked" {
		p.Attempts++
		p.History = append(p.History, Attempt{Status: status, Agent: info.Agent, Client: info.Client, Model: info.Model, Run: info.Run, Detail: info.Detail, At: now})
		if len(p.History) > historyLimit {
			p.History = p.History[len(p.History)-historyLimit:]
		}
	}
	if phase == "work" && (status == "ok" || status == "pending") {
		for _, later := range Phases[2:] {
			delete(st.Phases, later)
		}
	}
}
func (s *Store) State(task string) TaskState {
	st := TaskState{Phases: map[string]*PhaseRecord{}}
	name, e := taskFile(task)
	if e != nil {
		return st
	}
	if readJSON(s.path(name), &st) == nil && st.Phases != nil {
		return st
	}
	st.Phases = map[string]*PhaseRecord{}
	rows, _ := s.Sessions()
	for _, r := range rows {
		if r.Task != task {
			continue
		}
		status := "ok"
		if r.Error != "" {
			status = "failed"
		} else if r.Status == "blocked" {
			status = "blocked"
		} else if r.Exit != 0 {
			status = "failed"
		}
		phase := "review"
		if r.Mode == "work" {
			phase = "work"
		}
		detail := r.Error
		if detail == "" {
			detail = r.Status
		}
		apply(&st, phase, status, PhaseInfo{Agent: r.Agent, Client: r.Client, Model: r.Model, Run: r.Run, Detail: detail}, s.now())
	}
	return st
}
func (s *Store) Mark(task, phase, status string, info PhaseInfo) error {
	diskMu.Lock()
	defer diskMu.Unlock()
	name, e := taskFile(task)
	if e != nil {
		return e
	}
	st := s.State(task)
	apply(&st, phase, status, info, s.now())
	return writeJSON(s.path(name), st)
}
func (s *Store) ResetState(task string) error {
	diskMu.Lock()
	defer diskMu.Unlock()
	name, e := taskFile(task)
	if e != nil {
		return e
	}
	return remove(s.path(name))
}
