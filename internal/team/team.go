// Package team holds the org chart: the agents of a project, who reports to whom,
// and which client and model each one runs on. It is stored in <project>/tao.json.
package team

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/exadmax/TeamAgentOrchestrator/internal/fsx"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// FileName is the team file at the project root (versionable, unlike .tao/).
const FileName = "tao.json"

// Emblems are the function badges of the prototype (assets/avatars/av_e_<emblem>.png).
var Emblems = []string{"gestao", "planejamento", "codigo", "revisao", "dados", "pesquisa", "visual", "criacao", "producao", "ia"}

// Tiers map a work level to the client's model list (clients.Client.Tiers).
var Tiers = []string{"junior", "medium", "senior"}

// Agent is one node of the org chart.
type Agent struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Role         string `json:"role"`             // what the agent does, shown under the name
	Emblem       string `json:"emblem"`           // one of Emblems
	Face         string `json:"face"`             // avatar id (av_h1, av_f3, av_p_robo...)
	Client       string `json:"client"`           // clients.Client.ID
	Model        string `json:"model,omitempty"`  // empty = the client's model for Tier
	Tier         string `json:"tier,omitempty"`   // junior | medium | senior
	Parent       string `json:"parent,omitempty"` // superior's ID; empty = top of the chart
	Instructions string `json:"instructions,omitempty"`
	Paused       bool   `json:"paused,omitempty"`
}

// Team is the content of tao.json.
type Team struct {
	Name   string  `json:"name"`
	Agents []Agent `json:"agents"`
}

// Load reads tao.json from the project dir. A missing file is not an error: it returns ok=false.
func Load(dir string) (t *Team, ok bool, err error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	t = &Team{}
	if err := json.Unmarshal(data, t); err != nil {
		return nil, false, fmt.Errorf("%s: %w", FileName, err)
	}
	return t, true, t.Validate()
}

// Save writes tao.json atomically.
func (t *Team) Save(dir string) error {
	if err := t.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, FileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return fsx.Replace(tmp, path)
}

// Starter is the default team of a new project: a director on Claude, an architect,
// an engineer on Codex and a reviewer on OpenCode (the reviewer is always another client).
func Starter(name string) *Team {
	return &Team{Name: name, Agents: []Agent{
		{ID: "orquestrador", Name: "Orquestrador", Role: "Diretor do projeto", Emblem: "gestao", Face: "av_p_robo", Client: "claude", Tier: "senior"},
		{ID: "arquiteto", Name: "Arquiteto", Role: "Planejamento técnico", Emblem: "planejamento", Face: "av_h2", Client: "claude", Tier: "medium", Parent: "orquestrador"},
		{ID: "engenheiro", Name: "Engenheiro", Role: "Implementação", Emblem: "codigo", Face: "av_h4", Client: "codex", Tier: "medium", Parent: "orquestrador"},
		{ID: "revisor", Name: "Revisor QA", Role: "Testes e revisão", Emblem: "revisao", Face: "av_f2", Client: "opencode", Tier: "medium", Parent: "orquestrador"},
	}}
}

// Get returns a copy of the agent with that ID.
func (t *Team) Get(id string) (Agent, bool) {
	for _, a := range t.Agents {
		if a.ID == id {
			return a, true
		}
	}
	return Agent{}, false
}

func (t *Team) index(id string) int {
	for i, a := range t.Agents {
		if a.ID == id {
			return i
		}
	}
	return -1
}

// Add appends a new agent. An empty ID is derived from the name and made unique.
func (t *Team) Add(a Agent) (Agent, error) {
	a = clean(a)
	if a.ID == "" {
		a.ID = t.uniqueID(Slug(a.Name))
	} else if t.index(a.ID) >= 0 {
		return a, fmt.Errorf("já existe um agente com id %q", a.ID)
	}
	next := &Team{Name: t.Name, Agents: append(append([]Agent{}, t.Agents...), a)}
	if err := next.Validate(); err != nil {
		return a, err
	}
	t.Agents = next.Agents
	return a, nil
}

// Update replaces the agent with the same ID.
func (t *Team) Update(a Agent) error {
	a = clean(a)
	i := t.index(a.ID)
	if i < 0 {
		return fmt.Errorf("agente %q não existe", a.ID)
	}
	next := &Team{Name: t.Name, Agents: append([]Agent{}, t.Agents...)}
	next.Agents[i] = a
	if err := next.Validate(); err != nil {
		return err
	}
	t.Agents = next.Agents
	return nil
}

// Remove deletes an agent; its subordinates move up to its superior.
func (t *Team) Remove(id string) error {
	i := t.index(id)
	if i < 0 {
		return fmt.Errorf("agente %q não existe", id)
	}
	parent := t.Agents[i].Parent
	out := make([]Agent, 0, len(t.Agents)-1)
	for _, a := range t.Agents {
		if a.ID == id {
			continue
		}
		if a.Parent == id {
			a.Parent = parent
		}
		out = append(out, a)
	}
	t.Agents = out
	return nil
}

// Children returns the direct subordinates of id ("" = the top of the chart), in file order.
func (t *Team) Children(id string) []Agent {
	var out []Agent
	for _, a := range t.Agents {
		if a.Parent == id {
			out = append(out, a)
		}
	}
	return out
}

// Validate checks IDs, names, superiors and that the chart has no cycles.
func (t *Team) Validate() error {
	seen := map[string]bool{}
	for _, a := range t.Agents {
		switch {
		case a.ID == "":
			return errors.New("agente sem id")
		case seen[a.ID]:
			return fmt.Errorf("id repetido: %q", a.ID)
		case strings.TrimSpace(a.Name) == "":
			return fmt.Errorf("agente %q sem nome", a.ID)
		case a.Client == "":
			return fmt.Errorf("agente %q sem cliente", a.ID)
		}
		seen[a.ID] = true
	}
	for _, a := range t.Agents {
		if a.Parent != "" && !seen[a.Parent] {
			return fmt.Errorf("o superior %q de %q não existe", a.Parent, a.ID)
		}
		// walk up: more steps than agents means a cycle
		cur, steps := a, 0
		for cur.Parent != "" {
			if steps++; steps > len(t.Agents) || cur.Parent == a.ID {
				return fmt.Errorf("ciclo no organograma em %q", a.ID)
			}
			cur, _ = t.Get(cur.Parent)
		}
	}
	return nil
}

func (t *Team) uniqueID(base string) string {
	if base == "" {
		base = "agente"
	}
	id := base
	for n := 2; t.index(id) >= 0; n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

func clean(a Agent) Agent {
	a.Name, a.Role, a.Model = strings.TrimSpace(a.Name), strings.TrimSpace(a.Role), strings.TrimSpace(a.Model)
	return a
}

// Slug turns a name into an ID: "Portador Py→Go" -> "portador-py-go".
func Slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range norm.NFD.String(strings.ToLower(name)) {
		switch {
		case unicode.Is(unicode.Mn, r): // drop accents
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}
