package member

import (
	"errors"
	"fmt"
	"time"

	"github.com/thomashartm/motley/internal/agents/codex"
	"github.com/thomashartm/motley/internal/state"
	"github.com/thomashartm/motley/internal/tmux"
)

func refreshCodex(rows []Row) error {
	bySocket := map[string][]string{}
	for _, r := range rows {
		if r.CodexSession != "" {
			bySocket[r.CodexSocket] = append(bySocket[r.CodexSocket], r.CodexSession)
		}
	}
	var result error
	for socket, ids := range bySocket {
		sessions, discoveryErr := codex.Snapshot(socket, ids)
		result = errors.Join(result, discoveryErr)
		for i := range rows {
			r := &rows[i]
			if r.CodexSession == "" || r.CodexSocket != socket {
				continue
			}
			r.External = !r.Alive
			current := codex.Session{}
			for _, s := range sessions {
				if s.ID == r.CodexSession {
					current = s
					break
				}
			}
			var err error
			if discoveryErr == nil {
				current, err = followCodex(r, current, sessions, rows)
			}
			if err != nil || current.ID == "" || (discoveryErr != nil && current.MotleyStatus() != "working") {
				r.Alive, r.Status, r.Since = true, "unknown", 0
				if err == nil && discoveryErr == nil {
					err = fmt.Errorf("tracked Codex conversation is unavailable")
				}
				if err != nil {
					result = errors.Join(result, fmt.Errorf("%s: %w", r.ID, err))
				}
				continue
			}
			r.Alive, r.Status, r.Since, r.Seen = current.Loaded(), current.MotleyStatus(), 0, time.Now().Unix()
			if !sameDirectory(current.Cwd, r.Worktree) {
				r.Alive, r.Status = true, "moved"
				result = errors.Join(result, fmt.Errorf("%s: codex session directory changed", r.ID))
			}
		}
	}
	return result
}

func followCodex(r *Row, current codex.Session, sessions []codex.Session, rows []Row) (codex.Session, error) {
	// Keep the current active conversation when several conversations share a checkout.
	if current.MotleyStatus() == "working" {
		return current, nil
	}
	var working, active, loaded []codex.Session
	for _, s := range sessions {
		if (s.Status.Type != "active" && s.Status.Type != "idle") || !sameDirectory(s.Cwd, r.Worktree) {
			continue
		}
		tracked := false
		for _, other := range rows {
			tracked = tracked || (other.ID != r.ID && other.CodexSession == s.ID)
		}
		if tracked {
			continue
		}
		loaded = append(loaded, s)
		if s.Status.Type == "active" {
			active = append(active, s)
		}
		if s.MotleyStatus() == "working" {
			working = append(working, s)
		}
	}
	candidates := working
	if len(candidates) == 0 {
		if current.Status.Type == "active" {
			return current, nil
		}
		candidates = active
	}
	if len(candidates) == 0 {
		if current.Loaded() {
			return current, nil
		}
		candidates = loaded
	}
	if len(candidates) > 1 {
		return current, fmt.Errorf("multiple live Codex conversations in this worktree; cannot choose an active conversation")
	}
	if len(candidates) == 0 || candidates[0].ID == current.ID {
		return current, nil
	}
	next := candidates[0]
	if err := rebindCodex(r, current, next); err != nil {
		return current, err
	}
	return next, nil
}

func rebindCodex(r *Row, previous, next codex.Session) error {
	if err := CheckID(next.ID); err != nil {
		return err
	}
	dir, err := state.MembersDir()
	if err != nil {
		return err
	}
	lock, err := state.LockSpawn(dir)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	m, err := Load(dir, r.ID)
	if err != nil {
		return err
	}
	if m.CodexSession != r.CodexSession || m.CodexSocket != r.CodexSocket || !sameDirectory(m.Worktree, next.Cwd) {
		return fmt.Errorf("codex binding changed during refresh; retrying next refresh")
	}
	members, err := loadAll(dir)
	if err != nil {
		return err
	}
	for _, other := range members {
		if other.ID != m.ID && other.CodexSession == next.ID {
			return fmt.Errorf("active Codex conversation is already tracked by %s", other.ID)
		}
	}
	// Release the old client so Open and Reply cannot target its former conversation.
	if err := tmux.Release(m.ID); err != nil {
		return err
	}
	m.CodexSession = next.ID
	if m.Name == previous.Name && next.Name != "" {
		m.Name = next.Name
	}
	if err := saveManifest(dir, m); err != nil {
		return err
	}
	r.Manifest, r.External = m, true
	return nil
}
