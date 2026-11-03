package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestEndToEnd runs a real agent CLI. It spends a little quota, so it only runs with TAO_E2E=1
// (TAO_E2E_CLIENT picks the client, default opencode).
func TestEndToEnd(t *testing.T) {
	if os.Getenv("TAO_E2E") == "" {
		t.Skip("set TAO_E2E=1 to run a real CLI")
	}
	client := os.Getenv("TAO_E2E_CLIENT")
	if client == "" {
		client = "opencode"
	}
	a := NewApp()
	if err := a.open(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	var target string
	for _, ag := range a.Snapshot().Agents {
		if ag.Client == client {
			target = ag.ID
			break
		}
	}
	if err := a.RunAgent(target, "Responda apenas com a palavra: pronto"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.ServiceShutdown() }) // never leave the CLI (and what it spawned) running
	busy := func() bool {
		for _, v := range a.Snapshot().Agents {
			if v.ID == target {
				return v.Status == StatusActive
			}
		}
		return false
	}
	deadline := time.Now().Add(3 * time.Minute)
	for busy() {
		if time.Now().After(deadline) {
			t.Fatal("timeout")
		}
		time.Sleep(500 * time.Millisecond)
	}
	lines := a.Console(target)
	var all []string
	for _, l := range lines {
		all = append(all, l.Kind+": "+l.Text)
	}
	t.Log("\n" + strings.Join(all, "\n"))
	last := lines[len(lines)-1]
	if last.Kind != "end" || last.Error {
		t.Fatalf("run failed: %+v", last)
	}
	rows := a.Tokens("")
	if len(rows) != 1 || rows[0].Sessions != 1 {
		t.Fatalf("tokens = %+v", rows)
	}
	t.Logf("tokens: %+v", rows[0])
}
