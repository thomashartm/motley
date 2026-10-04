package claude

import (
	"encoding/json"
	"os"
	"sort"
	"time"

	"github.com/thomashartm/motley/internal/state"
)

// Activity keeps the parent and its children separate: a parent Stop or idle
// notification does not finish work that is still running in a subagent.
// The report writer serializes updates under the member's event-log lock.
type Activity struct {
	SessionID string                 `json:"session_id"`
	Parent    state.Event            `json:"parent"`
	Children  map[string]state.Event `json:"children,omitempty"`
	Current   state.Event            `json:"current"`
	Since     int64                  `json:"since"`
}

func ReadActivity(path string) (Activity, error) {
	data, err := state.Tail(path, 256*1024)
	if os.IsNotExist(err) {
		return Activity{}, nil
	}
	var a Activity
	if err == nil && len(data) > 0 {
		err = json.Unmarshal(data, &a)
	}
	return a, err
}

func (a *Activity) Apply(e state.Event) state.Event {
	if a.SessionID != e.AgentSessionID || e.Event == "SessionStart" {
		*a = Activity{SessionID: e.AgentSessionID}
	}
	if a.Children == nil {
		a.Children = map[string]state.Event{}
	}
	if raw, ok := e.Detail["active_agents"]; ok {
		var ids []string
		if json.Unmarshal([]byte(raw), &ids) == nil {
			active := map[string]state.Event{}
			for _, id := range ids {
				child, ok := a.Children[id]
				if !ok {
					child = state.Event{Status: "working"}
				}
				active[id] = child
			}
			a.Children = active
		}
	}
	id := e.Detail["agent_id"]
	switch {
	case e.Event == "SessionEnd" || e.Event == "AgentExit":
		a.Children = nil
		a.Parent = e
	case e.Event == "SubagentStop":
		delete(a.Children, id)
	case id != "" && e.Status != "":
		a.Children[id] = e
	case e.Status != "":
		a.Parent = e
	}
	current := a.Parent
	// Explicit questions and permission requests remain visible even while
	// another child is working. A child cannot clear the parent's question.
	if current.Status != "permission" && current.Status != "question" {
		ids := make([]string, 0, len(a.Children))
		for id := range a.Children {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, status := range []string{"permission", "question"} {
			for _, id := range ids {
				child := a.Children[id]
				if child.Status == status {
					current = child
					break
				}
			}
			if current.Status == status {
				break
			}
		}
	}
	if current.Status != "permission" && current.Status != "question" && len(a.Children) > 0 {
		current.Status = "working"
		current.Summary = "Subagents are working"
		current.Detail = nil
	}
	if current.Status == "" {
		current.Status = "working"
	}
	current.Agent, current.AgentSessionID = "claude", e.AgentSessionID
	current.Event, current.TS = e.Event, e.TS
	if current.Status != a.Current.Status || a.Since == 0 {
		a.Since = e.TS.Unix()
	}
	a.Current = current
	return current
}

// Status merges a live discovery result with hook evidence. A busy or waiting
// discovery result wins; idle may describe a parent with active children.
func (a Activity) Status(sessionID, live string, startedAt int64) (string, int64) {
	if a.SessionID != sessionID || a.Current.TS.IsZero() || a.Current.TS.Before(time.UnixMilli(startedAt)) {
		return live, 0
	}
	if live == "idle" && (len(a.Children) > 0 || a.Current.Status == "ready" || a.Current.Status == "question" || a.Current.Status == "permission") {
		return a.Current.Status, a.Since
	}
	if live == a.Current.Status {
		return live, a.Since
	}
	return live, 0
}
