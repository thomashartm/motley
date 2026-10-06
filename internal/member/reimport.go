package member

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thomashartm/motley/internal/agents/claude"
	"github.com/thomashartm/motley/internal/gitx"
	"github.com/thomashartm/motley/internal/tmux"
	"github.com/thomashartm/motley/internal/worktree"
)

// reimportClaude repairs the binding of the same live primary conversation.
// Callers hold the spawn lock. Identity, user metadata and history stay intact.
func (m Manifest) reimportClaude(dir string, s ImportCandidate) (Manifest, error) {
	if s.SessionID != m.ClaudeSession || s.ReimportID != m.ID {
		return Manifest{}, fmt.Errorf("reimport requires this member's primary Claude session")
	}
	if len(m.ClaudeSessions) > 0 {
		sessions, err := claude.Sessions()
		if err != nil {
			return Manifest{}, err
		}
		for _, other := range sessions {
			if other.PID > 0 && m.TracksClaude(other.SessionID) && !sameDirectory(other.Cwd, s.Cwd) {
				return Manifest{}, fmt.Errorf("tracked Claude sessions are in different workspaces; move or stop the other tracked conversations before reimporting")
			}
		}
	}
	if err := m.setImportedWorkspace(s.Cwd); err != nil {
		return Manifest{}, err
	}
	// Release stale terminal ownership; never stop or resume the conversation.
	if err := tmux.Release(m.ID); err != nil {
		return Manifest{}, err
	}
	if err := saveManifest(dir, m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func (m *Manifest) setImportedWorkspace(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("session directory must be absolute: %s", path)
	}
	cwd, err := worktree.Physical(path)
	if err != nil {
		return err
	}
	if info, err := os.Stat(cwd); err != nil || !info.IsDir() {
		return fmt.Errorf("session directory is unavailable: %s", cwd)
	}
	// Rebuild all checkout metadata, including when leaving Git entirely.
	m.Worktree, m.Repo = cwd, filepath.Base(cwd)
	m.RepoPath, m.Branch, m.Base, m.RemoteURL = "", "", "", ""
	m.GH = nil
	if rows, err := gitx.Worktrees(cwd); err == nil && len(rows) > 0 {
		m.RepoPath = rows[0].Path
		m.Branch, _ = gitx.Output(cwd, "symbolic-ref", "--short", "HEAD")
		m.Base, _ = worktree.Base(m.RepoPath)
		m.RemoteURL, _ = gitx.Output(m.RepoPath, "remote", "get-url", "origin")
	}
	return m.CheckCheckout()
}
