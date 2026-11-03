package tasks

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseGenerated procura um array JSON mesmo quando cercado por prosa ou Markdown.
func ParseGenerated(text string, validAgents []string) ([]NewTask, error) {
	agents := map[string]bool{}
	for _, id := range validAgents {
		agents[id] = true
	}
	for i := 0; i < len(text); i++ {
		if text[i] != '[' {
			continue
		}
		var items []struct {
			Title    string   `json:"title"`
			Done     string   `json:"done"`
			Assignee string   `json:"assignee"`
			Priority Priority `json:"priority"`
		}
		decoder := json.NewDecoder(strings.NewReader(text[i:]))
		if decoder.Decode(&items) != nil {
			continue
		}
		out := []NewTask{}
		for _, item := range items {
			if strings.TrimSpace(item.Title) == "" {
				continue
			}
			if !agents[item.Assignee] {
				item.Assignee = ""
			}
			if item.Priority != High {
				item.Priority = Normal
			}
			out = append(out, NewTask{Title: item.Title, Done: item.Done, Assignee: item.Assignee, Priority: item.Priority})
		}
		return out, nil
	}
	return nil, fmt.Errorf("nenhum array JSON válido de tarefas encontrado")
}
