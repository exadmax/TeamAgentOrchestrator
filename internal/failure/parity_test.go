package failure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testdata/parity.json is generated from the original Python orq (tools/orq/orq.py of JinChronicles):
// every row is what Python answered for a real-world message. The Go port must answer the same.
type parityRow struct {
	Text       string  `json:"text"`
	Classify   *string `json:"classify"`
	WaitSecs   *int    `json:"wait_secs"`
	Hint       *string `json:"hint"`
	Tokens     *int    `json:"tokens"`
	ErrorModel *string `json:"error_model"`
	Question   *string `json:"question"`
}

type parityFile struct {
	Now  string      `json:"now"`
	Rows []parityRow `json:"rows"`
	Fmt  []struct {
		Secs     int    `json:"secs"`
		Fmt      string `json:"fmt"`
		HintSecs *int   `json:"hint_secs"`
	} `json:"fmt"`
}

func val[T any](p *T) (T, bool) {
	var zero T
	if p == nil {
		return zero, false
	}
	return *p, true
}

func TestParityWithPython(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "parity.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pf parityFile
	if err := json.Unmarshal(data, &pf); err != nil {
		t.Fatal(err)
	}
	now, err := time.ParseInLocation("2006-01-02T15:04:05", pf.Now, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range pf.Rows {
		if r.Question != nil {
			if got := LastQuestion(r.Text); got != *r.Question {
				t.Errorf("LastQuestion(%q) = %q, python %q", r.Text, got, *r.Question)
			}
			continue
		}
		if want, ok := val(r.Classify); ok {
			if got := string(Classify(1, r.Text)); got != want {
				t.Errorf("Classify(%q) = %q, python %q", r.Text, got, want)
			}
		}
		gotWait, ok := ParseWait(r.Text, now)
		if want, has := val(r.WaitSecs); has != ok || (ok && int(gotWait.Round(time.Second).Seconds()) != want) {
			t.Errorf("ParseWait(%q) = %v,%v, python %v", r.Text, gotWait, ok, r.WaitSecs)
		}
		if want, _ := val(r.Hint); RetryHint(r.Text, now) != want {
			t.Errorf("RetryHint(%q) = %q, python %q", r.Text, RetryHint(r.Text, now), want)
		}
		gotTok, ok := ParseTokensUsed(r.Text)
		if want, has := val(r.Tokens); has != ok || (ok && gotTok != want) {
			t.Errorf("ParseTokensUsed(%q) = %d,%v, python %v", r.Text, gotTok, ok, r.Tokens)
		}
		if want, _ := val(r.ErrorModel); ErrorModel(r.Text) != want {
			t.Errorf("ErrorModel(%q) = %q, python %q", r.Text, ErrorModel(r.Text), want)
		}
	}
	for _, f := range pf.Fmt {
		d := time.Duration(f.Secs) * time.Second
		if got := FormatDuration(d); got != f.Fmt {
			t.Errorf("FormatDuration(%ds) = %q, python %q", f.Secs, got, f.Fmt)
		}
		got, ok := HintDuration(RetryHint(fmt.Sprintf("retry in %ds", f.Secs), now))
		if want, has := val(f.HintSecs); has != ok || (ok && int(got.Seconds()) != want) {
			t.Errorf("HintDuration(%ds) = %v,%v, python %v", f.Secs, got, ok, f.HintSecs)
		}
	}
}
