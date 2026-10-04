package member

import (
	"fmt"

	"github.com/thomashartm/motley/internal/state"
	"github.com/thomashartm/motley/internal/tmux"
)

// ReplacementSessions offers unregistered live sessions in this member's
// checkout. Choosing another checkout would invalidate its repository metadata.
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
		if sameDirectory(m.Worktree, s.Cwd) {
			candidates = append(candidates, s)
		}
	}
	return candidates, nil
}

// SwitchClaudeSession changes only the tracked conversation. Both processes
// keep running; any former Motley terminal is released, never killed.
func SwitchClaudeSession(id, sessionID string) (Manifest, error) {
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
			found = true
		}
	}
	if !found {
		return Manifest{}, fmt.Errorf("the selected Claude session is no longer available in this checkout or is already tracked")
	}
	if err := tmux.Release(id); err != nil {
		return Manifest{}, err
	}
	m.ClaudeSession = sessionID
	if err := saveManifest(dir, m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}
