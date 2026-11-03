package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/exadmax/TeamAgentOrchestrator/internal/stream"
)

// The test binary doubles as a fake agent CLI: with RUNNER_FAKE set it prints and exits
// instead of running the tests.
func TestMain(m *testing.M) {
	switch os.Getenv("RUNNER_FAKE") {
	case "":
		os.Exit(m.Run())
	case "ok":
		fmt.Println(`{"type":"thread.started","thread_id":"t1"}`)
		fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"pronto"}}`)
		fmt.Println(`{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":3}}`)
		os.Exit(0)
	case "fail":
		fmt.Fprintln(os.Stderr, "Error: 429 Too Many Requests, please retry in 2h")
		os.Exit(1)
	case "toolerr": // the real reviewer session that was misread as quota
		fmt.Println(`{"type":"user","message":{"content":[{"type":"tool_result","is_error":true,"content":"File does not exist."}]}}`)
		fmt.Println(`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"five_hour"}}`)
		fmt.Println(`{"type":"result","subtype":"success","is_error":false,"result":"APROVADO: ok","usage":{"input_tokens":5,"output_tokens":2}}`)
		os.Exit(0)
	case "chatty": // keeps printing forever, like an agent stuck in a loop
		for {
			fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"de novo"}}`)
			time.Sleep(200 * time.Millisecond)
		}
	case "silent":
		fmt.Println("starting")
		time.Sleep(20 * time.Second)
		os.Exit(0)
	}
}

func fake(t *testing.T, mode string) Spec {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return Spec{Client: "codex", Argv: []string{exe}, Env: []string{"RUNNER_FAKE=" + mode},
		LogPath: filepath.Join(t.TempDir(), "run.log")}
}

func TestRunOK(t *testing.T) {
	spec := fake(t, "ok")
	var evs []stream.Event
	res := Run(context.Background(), spec, Hooks{OnEvent: func(e stream.Event) { evs = append(evs, e) }})
	if res.Exit != 0 || res.Final != "pronto" || res.Session != "t1" || res.Usage.Input != 10 || res.Usage.Output != 3 {
		t.Fatalf("result = %+v", res)
	}
	if len(evs) != 3 {
		t.Fatalf("events = %+v", evs)
	}
	log, _ := os.ReadFile(spec.LogPath)
	if !strings.Contains(string(log), "agent_message") {
		t.Fatalf("log = %s", log)
	}
}

func TestRunFailKeepsTail(t *testing.T) {
	res := Run(context.Background(), fake(t, "fail"), Hooks{})
	if res.Exit != 1 || !strings.Contains(res.Tail, "429") {
		t.Fatalf("result = %+v", res)
	}
}

func TestRunIdleKill(t *testing.T) {
	spec := fake(t, "silent")
	spec.IdleKill, spec.Heartbeat = 2*time.Second, time.Second
	beats := 0
	start := time.Now()
	res := Run(context.Background(), spec, Hooks{OnIdle: func(time.Duration) { beats++ }})
	if res.Exit != ExitTimeout || !strings.Contains(res.Tail, "Timeout") {
		t.Fatalf("result = %+v", res)
	}
	if beats == 0 || time.Since(start) > 10*time.Second {
		t.Fatalf("beats = %d, took %s", beats, time.Since(start))
	}
}

func TestRunCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(500*time.Millisecond, cancel)
	res := Run(ctx, fake(t, "silent"), Hooks{})
	if res.Exit != ExitInterrupted {
		t.Fatalf("result = %+v", res)
	}
}

func TestRunLaunchError(t *testing.T) {
	res := Run(context.Background(), Spec{Client: "codex", Argv: []string{filepath.Join(t.TempDir(), "missing.exe")}}, Hooks{})
	if res.Exit != ExitLaunch || res.LaunchErr == "" {
		t.Fatalf("result = %+v", res)
	}
}

func TestRunMaxRun(t *testing.T) {
	spec := fake(t, "chatty")
	spec.MaxRun = 2 * time.Second
	res := Run(context.Background(), spec, Hooks{})
	if res.Exit != ExitTimeout || !strings.Contains(res.Tail, "maximum duration") {
		t.Fatalf("result = %+v", res)
	}
}

func TestToolErrorIsNotACLIError(t *testing.T) {
	spec := fake(t, "toolerr")
	spec.Client = "claude"
	res := Run(context.Background(), spec, Hooks{})
	if res.Exit != 0 || res.IsError || len(res.Errors) != 0 || res.Final != "APROVADO: ok" {
		t.Fatalf("result = %+v", res)
	}
}
