// Package importer reads cards projects without modifying their files.
package importer

import (
	"bytes"
	"encoding/json"
	"github.com/exadmax/TeamAgentOrchestrator/internal/i18n"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/exadmax/TeamAgentOrchestrator/internal/tasks"
	"github.com/exadmax/TeamAgentOrchestrator/internal/team"
)

const ownersPath = "docs/equipe/owners.json"

var faces = []string{"av_h1", "av_f1", "av_h2", "av_f2", "av_h3", "av_f3", "av_h4", "av_f4"}
var emblems = map[string]string{"produtor": "gestao", "arquiteto": "planejamento", "revisor": "revisao", "qa": "revisao", "guardiao-canon": "pesquisa", "pipeline-arte": "visual", "level-designer": "criacao", "audio": "producao", "manutencao": "dados"}

type Report struct {
	Agents   int      `json:"agents"`
	Tasks    int      `json:"tasks"`
	Sprints  int      `json:"sprints"`
	Warnings []string `json:"warnings"`
}

type role struct {
	ID    string
	Tool  string   `json:"tool"`
	Model string   `json:"model"`
	Owns  []string `json:"owns"`
	Read  []string `json:"read"`
}

// Detect reports whether the owners file exists.
func Detect(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ownersPath))
	return err == nil && !info.IsDir()
}

// readRoles decodes object tokens to preserve the source role order.
func readRoles(data []byte) ([]role, error) {
	var root struct {
		Roles json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(root.Roles))
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if token != json.Delim('{') {
		return nil, i18n.Err("importer.roles_object")
	}
	roles := []role{}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var r role
		if err := dec.Decode(&r); err != nil {
			return nil, err
		}
		r.ID = key.(string)
		roles = append(roles, r)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return roles, nil
}

func agentName(id string) string {
	words := strings.Fields(strings.ReplaceAll(id, "-", " "))
	for i, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func parseCard(data []byte) (map[string]any, error) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, i18n.Err("importer.front_matter_missing")
	}
	fields := map[string]any{}
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "---" {
			if text(fields, "id") == "" || text(fields, "title") == "" {
				return nil, i18n.Err("importer.required_fields")
			}
			return fields, nil
		}
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, i18n.Err("importer.invalid_front_matter_line", line)
		}
		value = strings.TrimSpace(value)
		var v any
		if json.Unmarshal([]byte(value), &v) != nil {
			v = value
		}
		fields[strings.TrimSpace(key)] = v
	}
	return nil, i18n.Err("importer.front_matter_unclosed")
}

func text(fields map[string]any, key string) string {
	s, _ := fields[key].(string)
	return s
}

// strList reads a YAML list of strings (deps: ["T030"]); anything else is empty.
func strList(value any) []string {
	list, _ := value.([]any)
	out := []string{}
	for _, v := range list {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func question(value any) string {
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return ""
	}
	last := list[len(list)-1]
	if s, ok := last.(string); ok {
		return s
	}
	if obj, ok := last.(map[string]any); ok {
		if s, ok := obj["q"].(string); ok {
			return s
		}
		return text(obj, "text")
	}
	return ""
}

// Import reads an ordered team and a task board from a cards project.
func Import(dir, projectName string, now time.Time) (*team.Team, *tasks.Board, Report, error) {
	report := Report{Warnings: []string{}}
	data, err := os.ReadFile(filepath.Join(dir, ownersPath))
	if err != nil {
		return nil, nil, report, i18n.Err("importer.read_team", err)
	}
	roles, err := readRoles(data)
	if err != nil {
		return nil, nil, report, i18n.Err("importer.invalid_team", err)
	}
	t := &team.Team{Name: projectName, Agents: []team.Agent{}}
	known := map[string]bool{}
	for _, r := range roles {
		known[r.ID] = true
	}
	for i, r := range roles {
		tier := map[string]string{"barato": "junior", "medio": "medium", "forte": "senior"}[r.Model]
		if tier == "" {
			tier = "medium"
		}
		emblem := emblems[r.ID]
		if emblem == "" {
			emblem = "codigo"
		}
		instructions := []string{}
		if len(r.Read) > 0 {
			instructions = append(instructions, i18n.T("importer.read_first", strings.Join(r.Read, ", ")))
		}
		if len(r.Owns) > 0 {
			instructions = append(instructions, i18n.T("importer.allowed_files", strings.Join(r.Owns, ", ")))
		}
		parent := ""
		if r.ID != "produtor" && known["produtor"] {
			parent = "produtor"
		}
		t.Agents = append(t.Agents, team.Agent{ID: r.ID, Name: agentName(r.ID), Role: i18n.T("importer.role", r.ID), Client: r.Tool, Tier: tier, Emblem: emblem, Face: faces[i%len(faces)], Parent: parent, Instructions: strings.Join(instructions, "\n")})
	}
	if err := t.Validate(); err != nil {
		return nil, nil, report, err
	}
	b := &tasks.Board{Tasks: []tasks.Task{}, Sprints: []tasks.Sprint{}}
	files, err := filepath.Glob(filepath.Join(dir, "docs/tasks/*.md"))
	if err != nil {
		return nil, nil, report, i18n.Err("importer.list_cards", err)
	}
	seen := map[string]bool{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, nil, report, i18n.Err("importer.read_card", err)
		}
		fields, err := parseCard(data)
		if err != nil {
			report.Warnings = append(report.Warnings, i18n.T("importer.skipped_card", filepath.Base(file), err))
			continue
		}
		id := text(fields, "id")
		if seen[id] {
			report.Warnings = append(report.Warnings, i18n.T("importer.duplicate_card", filepath.Base(file), id))
			continue
		}
		seen[id] = true
		task := tasks.Task{ID: id, Title: text(fields, "title"), Done: text(fields, "done"), Assignee: text(fields, "role"), Sprint: text(fields, "sprint"), Status: tasks.Status(text(fields, "status")), Priority: tasks.Normal, Origin: "import:docs/tasks/" + filepath.Base(file), Summary: text(fields, "summary"), Branch: text(fields, "branch"), Question: question(fields["questions"]), Deps: strList(fields["deps"]), Created: now, Updated: now}
		if !known[task.Assignee] {
			report.Warnings = append(report.Warnings, i18n.T("importer.unknown_role", id, task.Assignee))
			task.Assignee = ""
		}
		reviewer := text(fields, "reviewer")
		if reviewer != "" {
			for _, r := range roles {
				if r.Tool == reviewer {
					if task.Reviewer == "" || r.ID == "revisor" {
						task.Reviewer = r.ID
					}
				}
			}
		}
		switch task.Status {
		case tasks.Proposed, tasks.Ready, tasks.InProgress, tasks.Review, tasks.Done:
		case "blocked":
			task.Status = tasks.InProgress
		default:
			report.Warnings = append(report.Warnings, i18n.T("importer.unknown_status", id, task.Status))
			task.Status = tasks.Ready
		}
		if text(fields, "priority") == "high" {
			task.Priority = tasks.High
		}
		if opened, err := time.Parse("2006-01-02", text(fields, "opened")); err == nil {
			task.Created = opened
		}
		if verdicts, ok := fields["verdicts"].([]any); ok {
			for _, v := range verdicts {
				if s, ok := v.(string); ok {
					task.Verdicts = append(task.Verdicts, s)
				}
			}
		}
		if len(task.Verdicts) > 0 && task.Status == tasks.Review {
			task.Approved = strings.HasSuffix(task.Verdicts[len(task.Verdicts)-1], ":approve")
		}
		b.Tasks = append(b.Tasks, task)
	}
	sort.Slice(b.Tasks, func(i, j int) bool { return b.Tasks[i].ID < b.Tasks[j].ID })
	open := map[string]bool{}
	active := ""
	for _, task := range b.Tasks {
		if task.Sprint == "" {
			continue
		}
		open[task.Sprint] = open[task.Sprint] || task.Status != tasks.Done
		if task.Status != tasks.Done && task.Sprint > active {
			active = task.Sprint
		}
	}
	ids := []string{}
	for id := range open {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		sprint := tasks.Sprint{ID: id, Title: id, Status: tasks.Planned, Created: now}
		if !open[id] {
			sprint.Status = tasks.Closed
			sprint.Started = &now
			sprint.Closed = &now
		} else if id == active {
			sprint.Status = tasks.Active
			sprint.Started = &now
		}
		b.Sprints = append(b.Sprints, sprint)
	}
	report.Agents, report.Tasks, report.Sprints = len(t.Agents), len(b.Tasks), len(b.Sprints)
	return t, b, report, nil
}
