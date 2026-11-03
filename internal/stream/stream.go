// Package stream turns the JSON lines printed by each agent CLI into one common Event type,
// so the runner and the UI never need to know which client produced a line.
package stream

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Kind is the type of a normalized event.
type Kind string

const (
	Init   Kind = "init"   // session started: model and session id
	Text   Kind = "text"   // assistant text
	Tool   Kind = "tool"   // the agent called a tool (shell, edit, read...)
	Usage  Kind = "usage"  // token usage for a step
	Result Kind = "result" // the run finished (may carry usage and the final text)
	Error  Kind = "error"  // the CLI reported an error
	Info   Kind = "info"   // anything worth showing that is not one of the above (rate limit status...)
	Raw    Kind = "raw"    // a line that is not JSON (stderr, banners)
)

// Tokens is the token usage of one event. Input counts every prompt token, cached ones included;
// CacheRead and CacheWrite are the cached subsets.
type Tokens struct {
	Input      int     `json:"input"`
	Output     int     `json:"output"`
	CacheRead  int     `json:"cache_read"`
	CacheWrite int     `json:"cache_write"`
	CostUSD    float64 `json:"cost_usd"`
}

// Add sums two usages.
func (t Tokens) Add(o Tokens) Tokens {
	return Tokens{t.Input + o.Input, t.Output + o.Output, t.CacheRead + o.CacheRead, t.CacheWrite + o.CacheWrite, t.CostUSD + o.CostUSD}
}

// Event is one normalized line of agent output.
type Event struct {
	Kind    Kind    `json:"kind"`
	Text    string  `json:"text,omitempty"`
	Tool    string  `json:"tool,omitempty"`
	Model   string  `json:"model,omitempty"`
	Session string  `json:"session,omitempty"`
	Usage   *Tokens `json:"usage,omitempty"`
	IsError bool    `json:"is_error,omitempty"`
}

// maxToolText keeps tool summaries short enough for a console line.
const maxToolText = 160

// codexStdinBanner is printed by `codex exec` on stderr even when stdin is empty: noise for the console.
var codexStdinBanner = []byte("Reading additional input from stdin...")

// Parse normalizes one output line of the given client ("claude", "codex", "opencode", "gemini").
// Lines that are not JSON come back as a single Raw event; JSON lines with nothing to show come back empty.
func Parse(client string, line []byte) []Event {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}
	if line[0] != '{' {
		if bytes.Equal(line, codexStdinBanner) {
			return nil
		}
		return []Event{{Kind: Raw, Text: string(line)}}
	}
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return []Event{{Kind: Raw, Text: string(line)}}
	}
	switch client {
	case "claude":
		return parseClaude(m)
	case "codex":
		return parseCodex(m)
	case "opencode":
		return parseOpenCode(m)
	case "gemini":
		return parseGemini(m)
	}
	return []Event{{Kind: Raw, Text: string(line)}}
}

// ---------------------------------------------------------------- Claude Code (stream-json)

func parseClaude(m map[string]any) []Event {
	switch str(m, "type") {
	case "system":
		if str(m, "subtype") == "init" {
			return []Event{{Kind: Init, Model: str(m, "model"), Session: str(m, "session_id")}}
		}
	case "assistant":
		var out []Event
		msg := obj(m, "message")
		for _, c := range arr(msg, "content") {
			part, _ := c.(map[string]any)
			switch str(part, "type") {
			case "text":
				if t := strings.TrimSpace(str(part, "text")); t != "" {
					out = append(out, Event{Kind: Text, Text: t, Model: str(msg, "model")})
				}
			case "tool_use":
				out = append(out, Event{Kind: Tool, Tool: str(part, "name"), Text: summarize(obj(part, "input"))})
			}
		}
		return out
	case "user":
		for _, c := range arr(obj(m, "message"), "content") {
			part, _ := c.(map[string]any)
			// A failed tool call is part of the agent's work (it reads the error and moves on),
			// not a failure of the CLI: it stays a tool event.
			if str(part, "type") == "tool_result" && boolean(part, "is_error") {
				return []Event{{Kind: Tool, Tool: "erro da ferramenta", Text: clip(contentText(part["content"])), IsError: true}}
			}
		}
	case "result":
		u := obj(m, "usage")
		read, write := num(u, "cache_read_input_tokens"), num(u, "cache_creation_input_tokens")
		tok := Tokens{Input: num(u, "input_tokens") + read + write, Output: num(u, "output_tokens"),
			CacheRead: read, CacheWrite: write, CostUSD: float(m, "total_cost_usd")}
		ev := Event{Kind: Result, Text: str(m, "result"), Session: str(m, "session_id"), Usage: &tok,
			IsError: boolean(m, "is_error") || str(m, "subtype") != "success"}
		return []Event{ev}
	case "rate_limit_event":
		info := obj(m, "rate_limit_info")
		if s := str(info, "status"); s != "" && s != "allowed" {
			return []Event{{Kind: Info, Text: "limite de uso: " + s + " (" + str(info, "rateLimitType") + ")"}}
		}
	}
	return nil
}

// ---------------------------------------------------------------- Codex CLI (exec --json)

func parseCodex(m map[string]any) []Event {
	switch str(m, "type") {
	case "thread.started":
		return []Event{{Kind: Init, Session: str(m, "thread_id")}}
	case "item.completed":
		item := obj(m, "item")
		switch str(item, "type") {
		case "agent_message":
			return []Event{{Kind: Text, Text: strings.TrimSpace(str(item, "text"))}}
		case "command_execution":
			ev := Event{Kind: Tool, Tool: "shell", Text: clip(str(item, "command"))}
			if code, ok := item["exit_code"].(float64); ok && code != 0 {
				ev.IsError = true
			}
			return []Event{ev}
		case "file_change":
			var paths []string
			for _, c := range arr(item, "changes") {
				if ch, ok := c.(map[string]any); ok {
					paths = append(paths, str(ch, "path"))
				}
			}
			return []Event{{Kind: Tool, Tool: "edit", Text: clip(strings.Join(paths, ", "))}}
		case "mcp_tool_call":
			return []Event{{Kind: Tool, Tool: str(item, "server") + "." + str(item, "tool")}}
		case "web_search":
			return []Event{{Kind: Tool, Tool: "web_search", Text: clip(str(item, "query"))}}
		case "error":
			return []Event{{Kind: Error, Text: str(item, "message"), IsError: true}}
		}
	case "turn.completed":
		u := obj(m, "usage")
		tok := Tokens{Input: num(u, "input_tokens"), Output: num(u, "output_tokens"), CacheRead: num(u, "cached_input_tokens"),
			CacheWrite: num(u, "cache_write_input_tokens")}
		return []Event{{Kind: Result, Usage: &tok}}
	case "turn.failed":
		return []Event{{Kind: Error, Text: str(obj(m, "error"), "message"), IsError: true}}
	case "error":
		return []Event{{Kind: Error, Text: str(m, "message"), IsError: true}}
	}
	return nil
}

// ---------------------------------------------------------------- OpenCode (run --format json)

func parseOpenCode(m map[string]any) []Event {
	part := obj(m, "part")
	session := str(m, "sessionID")
	switch str(m, "type") {
	case "text":
		if t := strings.TrimSpace(str(part, "text")); t != "" {
			return []Event{{Kind: Text, Text: t, Session: session}}
		}
	case "tool_use":
		state := obj(part, "state")
		text := str(state, "title")
		if text == "" {
			text = summarize(obj(state, "input"))
		}
		return []Event{{Kind: Tool, Tool: str(part, "tool"), Text: clip(text), IsError: str(state, "status") == "error"}}
	case "step_finish":
		t := obj(part, "tokens")
		cache := obj(t, "cache")
		read, write := num(cache, "read"), num(cache, "write")
		tok := Tokens{Input: num(t, "input") + read + write, Output: num(t, "output") + num(t, "reasoning"),
			CacheRead: read, CacheWrite: write, CostUSD: float(part, "cost")}
		return []Event{{Kind: Usage, Usage: &tok, Session: session}}
	case "error":
		e := obj(m, "error")
		msg := str(obj(e, "data"), "message")
		if msg == "" {
			msg = str(e, "name")
		}
		return []Event{{Kind: Error, Text: msg, IsError: true}}
	}
	return nil
}

// ---------------------------------------------------------------- Gemini CLI (--output-format stream-json)

func parseGemini(m map[string]any) []Event {
	switch str(m, "type") {
	case "init":
		return []Event{{Kind: Init, Model: str(m, "model"), Session: str(m, "session_id")}}
	case "message":
		if str(m, "role") == "assistant" {
			if t := strings.TrimSpace(str(m, "content")); t != "" {
				return []Event{{Kind: Text, Text: t}}
			}
		}
	case "tool_use":
		return []Event{{Kind: Tool, Tool: str(m, "tool_name"), Text: summarize(obj(m, "parameters"))}}
	case "error":
		return []Event{{Kind: Error, Text: str(m, "message"), IsError: true}}
	case "result":
		s := obj(m, "stats")
		tok := Tokens{Input: num(s, "input_tokens"), Output: num(s, "output_tokens"), CacheRead: num(s, "cached")}
		return []Event{{Kind: Result, Usage: &tok, IsError: str(m, "status") == "error"}}
	}
	return nil
}

// ---------------------------------------------------------------- helpers

func obj(m map[string]any, k string) map[string]any {
	v, _ := m[k].(map[string]any)
	return v
}

func arr(m map[string]any, k string) []any {
	v, _ := m[k].([]any)
	return v
}

func str(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return v
}

func num(m map[string]any, k string) int {
	v, _ := m[k].(float64)
	return int(v)
}

func float(m map[string]any, k string) float64 {
	v, _ := m[k].(float64)
	return v
}

func boolean(m map[string]any, k string) bool {
	v, _ := m[k].(bool)
	return v
}

// summarize picks the most telling field of a tool input (command, path, pattern...).
func summarize(in map[string]any) string {
	for _, k := range []string{"command", "file_path", "filePath", "path", "pattern", "url", "query", "description", "prompt"} {
		if s := str(in, k); s != "" {
			return clip(s)
		}
	}
	return ""
}

func contentText(v any) string {
	switch c := v.(type) {
	case string:
		return c
	case []any:
		var b strings.Builder
		for _, x := range c {
			if p, ok := x.(map[string]any); ok {
				b.WriteString(str(p, "text"))
			}
		}
		return b.String()
	}
	return ""
}

func clip(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > maxToolText {
		return string(r[:maxToolText-1]) + "…"
	}
	return s
}
