package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/thomashartm/motley/internal/state"
)

// Activity keeps the parent and its children separate: a parent Stop or idle
// notification does not finish work that is still running in a subagent.
// The report writer serializes updates under the member's event-log lock.
type Activity struct {
	Shells    int                    `json:"shells,omitempty"`
	Monitors  int                    `json:"monitors,omitempty"`
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
	if value, ok := e.Detail["shells"]; ok {
		a.Shells, _ = strconv.Atoi(value)
	}
	if value, ok := e.Detail["monitors"]; ok {
		a.Monitors, _ = strconv.Atoi(value)
	}
	id := e.Detail["agent_id"]
	switch {
	case e.Event == "SessionEnd" || e.Event == "AgentExit":
		a.Children = nil
		a.Shells, a.Monitors = 0, 0
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
		current.Status = "waiting"
		current.Summary = "Subagents are waiting for processes"
		if a.Parent.Status == "working" {
			current.Status = "working"
		}
		for _, child := range a.Children {
			if child.Status != "waiting" {
				current.Status = "working"
				break
			}
		}
		if current.Status == "working" {
			current.Summary = "Subagents are working"
		}
		current.Detail = nil
	}
	if (current.Status == "ready" || current.Status == "idle") && (a.Shells > 0 || a.Monitors > 0) {
		current.Status = "background"
	}
	if current.Status == "" {
		current.Status = "working"
	}
	if a.Shells > 0 || a.Monitors > 0 || current.Detail["shells"] != "" || current.Detail["monitors"] != "" {
		detail := make(map[string]string, len(current.Detail)+2)
		for k, v := range current.Detail {
			detail[k] = v
		}
		delete(detail, "shells")
		delete(detail, "monitors")
		if a.Shells > 0 || a.Monitors > 0 {
			detail["shells"], detail["monitors"] = strconv.Itoa(a.Shells), strconv.Itoa(a.Monitors)
		}
		current.Detail = detail
	}
	current.Agent, current.AgentSessionID = "claude", e.AgentSessionID
	current.Event, current.TS = e.Event, e.TS
	if current.Status != a.Current.Status || a.Since == 0 {
		a.Since = e.TS.Unix()
	}
	a.Current = current
	return current
}

// Status merges a live discovery result with hook evidence. A waiting discovery
// result wins; idle may describe a parent with active children or an armed monitor.
func (a Activity) Status(sessionID, live string, startedAt int64) (string, int64) {
	if a.SessionID != sessionID || a.Current.TS.IsZero() || a.Current.TS.Before(time.UnixMilli(startedAt)) {
		return live, 0
	}
	if live == "idle" && a.Current.Status == "background" {
		if a.Monitors > 0 {
			return "background", a.Since
		}
		return a.Parent.Status, 0
	}
	if live == "working" && (a.Current.Status == "waiting" || a.Current.Status == "background") {
		return a.Current.Status, a.Since
	}
	if live == "idle" && (len(a.Children) > 0 || a.Current.Status == "ready" || a.Current.Status == "question" || a.Current.Status == "permission") {
		return a.Current.Status, a.Since
	}
	if live == "working" && len(a.Children) == 0 && (a.Current.Status == "ready" || a.Current.Status == "idle") {
		return "background", 0
	}
	if live == a.Current.Status {
		return live, a.Since
	}
	return live, 0
}

// ProcessSummary describes the last hook snapshot, not an OS process inventory.
func (a Activity) ProcessSummary(sessionID, live string, startedAt int64) string {
	if a.SessionID != sessionID || a.Current.TS.IsZero() || a.Current.TS.Before(time.UnixMilli(startedAt)) {
		return ""
	}
	shells := a.Shells
	if live == "idle" {
		shells = 0
	}
	if shells == 0 && a.Monitors == 0 {
		return ""
	}
	return fmt.Sprintf("Last reported: %d shell(s), %d monitor(s)", shells, a.Monitors)
}
