package member

import (
	"fmt"

	"github.com/thomashartm/motley/internal/agents/claude"
	"github.com/thomashartm/motley/internal/state"
	"github.com/thomashartm/motley/internal/tmux"
)

// ReplacementSessions offers other conversations in the saved checkout and the
// member's own primary conversation after a workspace change.
func ReplacementSessions(id string) ([]ImportCandidate, error) {
	dir, err := state.MembersDir()
	if err != nil {
		return nil, err
	}
	m, err := Load(dir, id)
	if err != nil {
		return nil, err
	}
	if m.ClaudeSession == "" {
		return nil, fmt.Errorf("switching tracked sessions requires an imported Claude member")
	}
	sessions, err := DiscoverImports("claude")
	if err != nil {
		return nil, err
	}
	var candidates []ImportCandidate
	for _, s := range sessions {
		if s.ReimportID == m.ID || (s.ReimportID == "" && sameDirectory(m.Worktree, s.Cwd)) {
			candidates = append(candidates, s)
		}
	}
	if len(m.ClaudeSessions) > 0 {
		live, err := claude.Sessions()
		if err != nil {
			return nil, err
		}
		for _, s := range live {
			if s.PID > 0 && s.SessionID != m.ClaudeSession && m.TracksClaude(s.SessionID) && sameDirectory(m.Worktree, s.Cwd) {
				candidates = append(candidates, ImportCandidate{Agent: "claude", SessionID: s.SessionID, Name: s.Name, Cwd: s.Cwd, Status: s.MotleyStatus(), Kind: s.Kind})
			}
		}
	}
	return candidates, nil
}

func AdditionalSessions(id string) ([]ImportCandidate, error) {
	dir, err := state.MembersDir()
	if err != nil {
		return nil, err
	}
	m, err := Load(dir, id)
	if err != nil {
		return nil, err
	}
	candidates, err := ReplacementSessions(id)
	if err != nil {
		return nil, err
	}
	var result []ImportCandidate
	for _, s := range candidates {
		if s.ReimportID == "" && !m.TracksClaude(s.SessionID) {
			result = append(result, s)
		}
	}
	return result, nil
}

// SwitchClaudeSession changes only the tracked conversation. Both processes
// keep running; any former Motley terminal is released, never killed.
func SwitchClaudeSession(id, sessionID string) (Manifest, error) {
	return trackClaudeSession(id, sessionID, false)
}

// AddClaudeSession monitors another explicitly selected conversation without
// changing the member's primary terminal or restarting either agent.
func AddClaudeSession(id, sessionID string) (Manifest, error) {
	return trackClaudeSession(id, sessionID, true)
}

func trackClaudeSession(id, sessionID string, additional bool) (Manifest, error) {
	if err := CheckID(sessionID); err != nil {
		return Manifest{}, err
	}
	dir, err := state.MembersDir()
	if err != nil {
		return Manifest{}, err
	}
	lock, err := state.LockSpawn(dir)
	if err != nil {
		return Manifest{}, err
	}
	defer func() { _ = lock.Close() }()
	m, err := Load(dir, id)
	if err != nil {
		return Manifest{}, err
	}
	candidates, err := ReplacementSessions(id)
	if err != nil {
		return Manifest{}, err
	}
	found := false
	for _, s := range candidates {
		if s.SessionID == sessionID {
			if s.ReimportID == m.ID && !additional {
				return m.reimportClaude(dir, s)
			}
			found = true
		}
	}
	if !found {
		return Manifest{}, fmt.Errorf("the selected Claude session is no longer available in this checkout or is already tracked")
	}
	if additional {
		if m.TracksClaude(sessionID) {
			return Manifest{}, fmt.Errorf("session is already tracked by this member")
		}
		m.ClaudeSessions = append(m.ClaudeSessions, sessionID)
		return m, saveManifest(dir, m)
	}
	if err := tmux.Release(id); err != nil {
		return Manifest{}, err
	}
	if len(m.ClaudeSessions) > 0 {
		next := []string{m.ClaudeSession}
		for _, id := range m.ClaudeSessions {
			if id != sessionID {
				next = append(next, id)
			}
		}
		m.ClaudeSessions = next
	}
	m.ClaudeSession = sessionID
	if err := saveManifest(dir, m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}
