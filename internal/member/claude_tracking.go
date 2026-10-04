package member

import (
	"path/filepath"
	"strings"

	"github.com/thomashartm/motley/internal/agents/claude"
)

// TracksClaude checks explicitly linked conversations. Sharing a checkout alone is
// never evidence that two sessions belong to the same member.
func (m Manifest) TracksClaude(id string) bool {
	if id == "" {
		return false
	}
	if id == m.ClaudeSession {
		return true
	}
	for _, additional := range m.ClaudeSessions {
		if additional == id {
			return true
		}
	}
	return false
}

func (m Manifest) ClaudeActivityPath(dir, sessionID string) string {
	if sessionID == m.ClaudeSession && len(m.ClaudeSessions) == 0 {
		return filepath.Join(dir, m.ID+".claude.json")
	}
	return filepath.Join(dir, m.ID+"."+sessionID+".claude.json")
}

type ClaudeSessionStatus struct {
	ID, Name, Kind, Status string
	Alive                  bool
}

func (s ClaudeSessionStatus) Location() string {
	switch s.Kind {
	case "interactive":
		return "FG"
	case "background":
		return "BG"
	default:
		return "—"
	}
}

// SessionLocation describes live conversations, independently of whether they
// are busy. Per-conversation statuses explain which one is doing the work.
func (r Row) SessionLocation() string {
	fg, bg := false, false
	for _, s := range r.ClaudeStatuses {
		if s.Alive {
			fg = fg || s.Kind == "interactive"
			bg = bg || s.Kind == "background"
		}
	}
	switch {
	case fg && bg:
		return "FG+BG"
	case fg:
		return "FG"
	case bg:
		return "BG"
	default:
		return ""
	}
}

func (r *Row) refreshClaudeSessions(dir string, sessions []claude.Session) error {
	r.ClaudeStatuses = nil
	// Preserve an owned terminal's status until its primary session is found.
	primaryAlive, primaryStatus, primarySince := r.Alive, r.Status, r.Since
	r.Alive, r.Status, r.Since = false, "dead", 0
	for _, id := range append([]string{r.ClaudeSession}, r.ClaudeSessions...) {
		entry := ClaudeSessionStatus{ID: id, Status: "dead"}
		since := int64(0)
		if id == r.ClaudeSession && primaryAlive {
			entry.Alive, entry.Status = true, primaryStatus
			since = primarySince
		}
		for _, s := range sessions {
			if s.SessionID != id || !sameDirectory(s.Cwd, r.Worktree) {
				continue
			}
			entry.Name, entry.Kind, entry.Alive = s.Name, s.Kind, s.PID > 0
			entry.Status = "dead"
			if entry.Alive {
				activity, err := claude.ReadActivity(r.ClaudeActivityPath(dir, id))
				if err == nil && activity.SessionID == "" {
					activity, err = claude.ReadActivity(filepath.Join(dir, r.ID+".claude.json"))
				}
				if err != nil {
					return err
				}
				entry.Status, since = activity.Status(id, s.MotleyStatus(), s.StartedAt)
			}
			break
		}
		r.ClaudeStatuses = append(r.ClaudeStatuses, entry)
		if entry.Alive && (!r.Alive || claudeStatusPriority(entry.Status) > claudeStatusPriority(r.Status)) {
			r.Status, r.Since = entry.Status, since
		}
		r.Alive = r.Alive || entry.Alive
	}
	return nil
}

func claudeStatusPriority(status string) int {
	for priority, candidate := range strings.Fields("dead ended idle alive ready starting working question permission") {
		if candidate == status {
			return priority
		}
	}
	return 0
}
