// Package worktree implements the worktree operations needed by members.
package worktree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/thomashartm/motley/internal/gitx"
)

// Base prefers origin's recorded default, then local main or master.
func Base(repo string) (string, error) {
	if ref, err := gitx.Output(repo, "symbolic-ref", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimPrefix(ref, "refs/remotes/origin/"), nil
	}
	for _, branch := range []string{"main", "master"} {
		if _, err := gitx.Output(repo, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
			return branch, nil
		}
	}
	return "", fmt.Errorf("%s has neither a local main nor master branch", repo)
}

// CheckFeatureBranch applies to linked worktrees, never to their base checkout.
func CheckFeatureBranch(repo, branch string) error {
	if branch == "" || Protected(branch) {
		return fmt.Errorf("branch %q is not a feature branch; linked worktrees must not use the default or a protected branch", branch)
	}
	base, err := Base(repo)
	if err != nil {
		return fmt.Errorf("cannot verify the repository default branch: %w", err)
	}
	if branch == base {
		return fmt.Errorf("branch %q is the default branch; linked worktrees require a feature branch", branch)
	}
	return nil
}

func CheckNew(repo, branch, path string) error {
	resolved, err := gitx.Output(repo, "check-ref-format", "--branch", branch)
	if err != nil || resolved != branch {
		return fmt.Errorf("invalid branch name %q", branch)
	}
	if err := CheckFeatureBranch(repo, branch); err != nil {
		return err
	}
	if _, err := gitx.Output(repo, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		return fmt.Errorf("branch %q already exists; spawn requires a new branch", branch)
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("worktree path already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Create leaves any created worktree in place on failure, so work is never lost.
func Create(repo, branch, base, path string, output io.Writer) error {
	if err := CheckNew(repo, branch, path); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(output, "Fetching origin/"+base+"…")
	if err := gitx.Run(repo, output, "fetch", "origin", base); err != nil {
		if err := gitx.Run(repo, output, "fetch", "origin"); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(output, "Creating worktree…")
	if err := gitx.Run(repo, output, "worktree", "add", "-b", branch, path, "origin/"+base); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(output, "Pushing branch…")
	if err := gitx.Run(path, output, "push", "-u", "origin", branch); err != nil {
		return fmt.Errorf("worktree retained at %s; push failed: %w", path, err)
	}
	_, _ = fmt.Fprintln(output, "Copying local artifacts…")
	if err := CopyArtifacts(repo, path, output); err != nil {
		return fmt.Errorf("worktree retained at %s; artifact copy failed: %w", path, err)
	}
	return nil
}
