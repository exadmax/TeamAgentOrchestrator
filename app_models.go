package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/exadmax/TeamAgentOrchestrator/internal/clients"
	"github.com/exadmax/TeamAgentOrchestrator/internal/failure"
	"github.com/exadmax/TeamAgentOrchestrator/internal/store"
	"github.com/exadmax/TeamAgentOrchestrator/internal/team"
)

// maxSwitches bounds how many times one task run hops to another model or the reserve after a quota failure.
const maxSwitches = 3

// OverrideView is one model swap, as the "Modelos e cotas" window lists it.
type OverrideView struct {
	Key    string     `json:"key"`
	Client string     `json:"client"`
	Task   string     `json:"task"`
	Model  string     `json:"model"`
	Until  *time.Time `json:"until"`
}

// ClientModels is the model list of one client, for the swap form.
type ClientModels struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Models []string `json:"models"`
}

// ModelsView is the content of the "Modelos e cotas" window.
type ModelsView struct {
	Auto      bool           `json:"auto"`
	Clocks    []store.Clock  `json:"clocks"`
	Overrides []OverrideView `json:"overrides"`
	Clients   []ClientModels `json:"clients"`
}

// Models returns the quota clocks, the model swaps and the auto-switch flag.
func (a *App) Models() ModelsView {
	a.mu.Lock()
	defer a.mu.Unlock()
	v := ModelsView{Clocks: []store.Clock{}, Overrides: []OverrideView{}, Clients: []ClientModels{}}
	for _, c := range a.catalog {
		v.Clients = append(v.Clients, ClientModels{ID: c.ID, Name: c.Name, Models: append([]string{}, c.Models...)})
	}
	if a.store == nil {
		return v
	}
	v.Auto, v.Clocks = a.store.AutoSwitch(), a.store.ActiveClocks()
	for k, ov := range a.store.Overrides() {
		row := OverrideView{Key: k, Client: k, Model: ov.Model, Until: ov.Until}
		if task, client, ok := strings.Cut(k, ":"); ok {
			row.Task, row.Client = task, client
		}
		v.Overrides = append(v.Overrides, row)
	}
	sort.Slice(v.Overrides, func(i, j int) bool { return v.Overrides[i].Key < v.Overrides[j].Key })
	return v
}

// SetModelOverride swaps the model of a client (task "" = every task) for minutes (0 = until removed).
func (a *App) SetModelOverride(client, model string, minutes int, task string) error {
	if strings.TrimSpace(model) == "" {
		return errors.New("escolha o modelo")
	}
	if _, ok := clients.Find(a.catalog, client); !ok {
		return fmt.Errorf("cliente %q desconhecido", client)
	}
	return a.models(func(st *store.Store) error {
		return st.SetOverride(client, model, time.Duration(minutes)*time.Minute, task)
	})
}

// ClearModelOverride removes one swap (its key as Models lists it).
func (a *App) ClearModelOverride(key string) error {
	return a.models(func(st *store.Store) error { return st.ClearKey(key) })
}

// SetAutoSwitch turns the automatic model/reserve switch on quota failures on or off.
func (a *App) SetAutoSwitch(on bool) error {
	return a.models(func(st *store.Store) error { return st.SetAuto(on) })
}

// ClearClock says a model has quota again before its clock ran out.
func (a *App) ClearClock(client, model string) error {
	return a.models(func(st *store.Store) error { return st.SetClock(client, model, 0, "liberado", "manual") })
}

func (a *App) models(f func(st *store.Store) error) error {
	a.mu.Lock()
	st := a.store
	a.mu.Unlock()
	if st == nil {
		return errors.New("nenhum projeto aberto")
	}
	if err := f(st); err != nil {
		return err
	}
	a.emit("models", a.Models())
	a.emit("snapshot", a.Snapshot())
	return nil
}

// watchClocks announces quotas that came back. It runs for the life of the app.
func (a *App) watchClocks(stop <-chan struct{}) {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
		}
		a.mu.Lock()
		st := a.store
		a.mu.Unlock()
		if st == nil {
			continue
		}
		back := st.PopExpired()
		for _, c := range back {
			a.push(Line{Agent: c.Client, Kind: "info", Text: fmt.Sprintf("a cota de %s/%s voltou", c.Client, c.Model)})
		}
		if len(back) > 0 {
			a.emit("models", a.Models())
			a.emit("snapshot", a.Snapshot())
		}
	}
}

// route is who actually runs a session and with which model.
type route struct {
	agent team.Agent
	model string
	note  string // why it is not the planned agent or model ("" = as planned)
}

// routeLocked picks ag's model (task swap, client swap, own model, tier) and, when that model is out of
// quota and auto-switch is on, another free model of the same client or else the agent's reserve.
func (a *App) routeLocked(ag team.Agent, task string) (route, error) {
	model := a.modelLocked(ag, task)
	until, clocked := a.store.ClockUntil(ag.Client, model)
	if !clocked {
		return route{agent: ag, model: model}, nil
	}
	why := fmt.Sprintf("%s/%s sem cota até %s", ag.Client, orDefault(model), until.Format("15:04"))
	if !a.store.AutoSwitch() {
		return route{}, fmt.Errorf("%s: troque o modelo ou ligue a troca automática (Ferramentas → Modelos e cotas)", why)
	}
	if c, ok := clients.Find(a.catalog, ag.Client); ok {
		if m := a.freeModelLocked(c, model); m != "" {
			return route{agent: ag, model: m, note: why + "; usando " + m}, nil
		}
	}
	if fb, ok := a.team.Get(ag.Fallback); ok && !fb.Paused && a.runs[fb.ID] == nil {
		fm := a.modelLocked(fb, task)
		if _, busy := a.store.ClockUntil(fb.Client, fm); !busy {
			return route{agent: fb, model: fm, note: why + "; usando a reserva " + fb.Name}, nil
		}
	}
	return route{}, fmt.Errorf("%s e não há outro modelo nem reserva livre", why)
}

// freeModelLocked returns the first model of c after current (wrapping around) without a quota clock.
func (a *App) freeModelLocked(c clients.Client, current string) string {
	start := 0
	for i, m := range c.Models {
		if m == current {
			start = i + 1
		}
	}
	for i := range c.Models {
		m := c.Models[(start+i)%len(c.Models)]
		if m == current {
			continue
		}
		if _, busy := a.store.ClockUntil(c.ID, m); !busy {
			return m
		}
	}
	return ""
}

// switchable reports whether a failed session is worth retrying on another model or the reserve.
func (a *App) switchable(k failure.Kind) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return (k == failure.Quota || k == failure.Model) && a.store != nil && a.store.AutoSwitch()
}

func orDefault(model string) string {
	if model == "" {
		return "padrão"
	}
	return model
}
