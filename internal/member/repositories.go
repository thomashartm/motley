package member

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverRepos scans each entrypoint once, without descending into nested
// directories. Only main repositories are eligible for spawning worktrees.
func DiscoverRepos(roots []string) ([]string, error) {
	type repository struct{ name, path string }
	var repos []repository
	seen := map[string]bool{}
	names := map[string]map[string]bool{}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, fmt.Errorf("read repository root %s: %w", root, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			path, err := resolveRepoAt(root, entry.Name())
			if err != nil {
				continue
			}
			if names[entry.Name()] == nil {
				names[entry.Name()] = map[string]bool{}
			}
			names[entry.Name()][path] = true
			if seen[path] {
				continue
			}
			seen[path] = true
			// Keep the configured path as the selector, even for symlinks.
			selection, err := filepath.Abs(filepath.Join(root, entry.Name()))
			if err != nil {
				return nil, err
			}
			repos = append(repos, repository{entry.Name(), selection})
		}
	}
	var selections []string
	for _, repo := range repos {
		if len(names[repo.name]) > 1 {
			selections = append(selections, repo.path)
		} else {
			selections = append(selections, repo.name)
		}
	}
	sort.Strings(selections)
	return selections, nil
}

// ResolveRepo accepts a unique basename, or an absolute path directly beneath
// a configured entrypoint. Duplicate names never silently select the first root.
func ResolveRepo(roots []string, name string) (string, error) {
	if name == "" || name == "." || name == ".." || (!filepath.IsAbs(name) && filepath.Base(name) != name) {
		return "", fmt.Errorf("--repo must be a name or absolute path directly under a configured repository root")
	}
	var matches, choices []string
	var failures []error
	seen := map[string]bool{}
	for _, root := range roots {
		if filepath.IsAbs(name) {
			parent, err := filepath.Abs(root)
			if err != nil {
				return "", err
			}
			if filepath.Dir(filepath.Clean(name)) != parent {
				continue
			}
		}
		path, err := resolveRepoAt(root, filepath.Base(name))
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		matches = append(matches, path)
		choice, err := filepath.Abs(filepath.Join(root, filepath.Base(name)))
		if err != nil {
			return "", err
		}
		choices = append(choices, choice)
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		if len(failures) > 0 {
			return "", fmt.Errorf("repository %q not found in configured roots: %w", name, errors.Join(failures...))
		}
		return "", fmt.Errorf("repository %q is not directly under a configured repository root", name)
	default:
		return "", fmt.Errorf("repository %q is ambiguous; use one of these paths with --repo: %s", name, strings.Join(choices, ", "))
	}
}
