// Package runner starts an agent CLI, streams its output line by line as normalized events,
// writes the raw log and kills the process when it stays silent for too long.
// Port of run_session from the Python orq.
package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/exadmax/TeamAgentOrchestrator/internal/i18n"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/exadmax/TeamAgentOrchestrator/internal/stream"
)

// Exit codes the runner sets itself, the same ones the Python orq used.
const (
	ExitTimeout     = 124 // no output for IdleKill
	ExitInterrupted = 130 // cancelled by the user
	ExitLaunch      = 127 // the executable could not start
)

const (
	tailLines   = 80      // lines kept for failure classification
	maxLineSize = 4 << 20 // a single JSON line may carry a large tool result
	pollEvery   = time.Second
	maxErrors   = 10 // error texts kept per run
)

// Spec describes one run.
type Spec struct {
	Client    string        // "claude", "codex", "opencode", "gemini": selects the stream parser
	Argv      []string      // argv[0] must already be resolved to an executable path
	Dir       string        // working directory (the project)
	Env       []string      // extra KEY=VALUE pairs on top of the current environment
	LogPath   string        // raw output log; "" = no log
	IdleKill  time.Duration // kill after this long without output; 0 = never
	Heartbeat time.Duration // call OnIdle every so often while silent; 0 = never
	MaxRun    time.Duration // kill after this long in total, even with output (runaway agents); 0 = never
}

// Hooks receive what happens during the run. Any of them may be nil.
type Hooks struct {
	OnEvent func(stream.Event)
	OnIdle  func(silent time.Duration)
}

// Result is what the run left behind.
type Result struct {
	Exit      int           `json:"exit"`
	Tail      string        `json:"tail"`  // last lines of raw output, for failure.Classify
	Final     string        `json:"final"` // last assistant text (or the result text)
	Usage     stream.Tokens `json:"usage"`
	Model     string        `json:"model"`
	Session   string        `json:"session"`
	IsError   bool          `json:"is_error"`         // the CLI itself reported an error event
	Errors    []string      `json:"errors,omitempty"` // texts of those error events, for classification
	Start     time.Time     `json:"start"`
	End       time.Time     `json:"end"`
	LaunchErr string        `json:"launch_err,omitempty"`
}

// Run executes the spec and blocks until the process exits, goes silent past IdleKill,
// or ctx is cancelled. It never returns an error: launch problems come back as ExitLaunch.
func Run(ctx context.Context, spec Spec, hooks Hooks) Result {
	res := Result{Start: time.Now()}
	finish := func(code int) Result {
		res.Exit, res.End = code, time.Now()
		return res
	}
	if len(spec.Argv) == 0 {
		res.LaunchErr = i18n.T("runner.empty_command")
		return finish(ExitLaunch)
	}

	var log io.Writer = io.Discard
	if spec.LogPath != "" {
		if err := os.MkdirAll(filepath.Dir(spec.LogPath), 0o755); err == nil {
			if f, err := os.Create(spec.LogPath); err == nil {
				defer f.Close()
				log = f
				fmt.Fprintf(f, "$ %s\n\n", strings.Join(spec.Argv, " "))
			}
		}
	}

	cmd := exec.Command(spec.Argv[0], spec.Argv[1:]...)
	cmd.Dir = spec.Dir
	cmd.Env = append(os.Environ(), spec.Env...)
	cmd.Stdin = nil // reads from the null device: CLIs that wait on stdin (codex) do not hang
	hideWindow(cmd)
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		res.LaunchErr = err.Error()
		fmt.Fprintf(log, "launch error: %v\n", err)
		return finish(ExitLaunch)
	}

	lines := make(chan string, 64)
	go func() {
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64<<10), maxLineSize)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	waited := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		pw.Close()
		waited <- err
	}()

	var tail []string
	keep := func(s string) {
		tail = append(tail, s)
		if len(tail) > tailLines {
			tail = tail[len(tail)-tailLines:]
		}
	}
	handle := func(line string) {
		fmt.Fprintln(log, line)
		keep(line)
		for _, ev := range stream.Parse(spec.Client, []byte(line)) {
			res.absorb(ev)
			if hooks.OnEvent != nil {
				hooks.OnEvent(ev)
			}
		}
	}
	stop := func(code int, msg string) int {
		keep(msg)
		fmt.Fprintln(log, msg)
		kill(cmd)
		<-waited
		return code
	}
	last, lastBeat := time.Now(), time.Time{}
	ticker := time.NewTicker(pollEvery)
	defer ticker.Stop()
	code := -1
	for code < 0 {
		select {
		case line, ok := <-lines:
			if !ok {
				lines = nil // drained; wait for the process below
				continue
			}
			last = time.Now()
			handle(line)
		case <-ticker.C:
			silent := time.Since(last)
			if spec.IdleKill > 0 && silent >= spec.IdleKill {
				code = stop(ExitTimeout, i18n.T("runner.idle_timeout", int(silent.Seconds())))
				continue
			}
			if spec.MaxRun > 0 && time.Since(res.Start) >= spec.MaxRun {
				code = stop(ExitTimeout, i18n.T("runner.max_timeout", spec.MaxRun))
				continue
			}
			if spec.Heartbeat > 0 && silent >= spec.Heartbeat && time.Since(lastBeat) >= spec.Heartbeat && hooks.OnIdle != nil {
				lastBeat = time.Now()
				hooks.OnIdle(silent)
			}
		case <-ctx.Done():
			code = stop(ExitInterrupted, i18n.T("runner.interrupted"))
		case err := <-waited:
			if lines != nil { // flush what is still buffered; nil means it was already drained
				for line := range lines {
					handle(line)
				}
			}
			code = exitCode(err)
		}
	}
	res.Tail = strings.Join(tail, "\n")
	return finish(code)
}

// absorb keeps the run summary up to date with one event.
func (r *Result) absorb(ev stream.Event) {
	if ev.Usage != nil {
		r.Usage = r.Usage.Add(*ev.Usage)
	}
	if ev.Model != "" {
		r.Model = ev.Model
	}
	if ev.Session != "" {
		r.Session = ev.Session
	}
	switch ev.Kind {
	case stream.Text:
		r.Final = ev.Text
	case stream.Result:
		if ev.Text != "" {
			r.Final = ev.Text
		}
		r.IsError = r.IsError || ev.IsError
	case stream.Error:
		r.IsError = true
		if ev.Text != "" && len(r.Errors) < maxErrors {
			r.Errors = append(r.Errors, ev.Text)
		}
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}

func kill(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = killTree(cmd)
	}
}
