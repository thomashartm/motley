package worktree

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thomashartm/motley/internal/gitx"
)

// SourceBranch keeps the exact ref separate from its display and PR base name.
type SourceBranch struct {
	Ref, Name, Remote string
	Default           bool
}

func (b SourceBranch) Label() string {
	label := b.Name + " (local)"
	if b.Remote != "" {
		label = b.Remote + "/" + b.Name + " (remote)"
	}
	if b.Default {
		label += " · default"
	}
	return label
}

func (b SourceBranch) Group() string {
	if b.Default {
		return "Default branch"
	}
	if !strings.Contains(b.Name, "/") {
		return "Unprefixed branches"
	}
	return "Other branches"
}

// SourceBranches lists local and fetched remote branches without hiding forks
// or local commits behind a similarly named origin branch.
func SourceBranches(repo string) ([]SourceBranch, error) {
	out, err := gitx.Output(repo, "for-each-ref", "--format=%(refname) %(symref)", "refs/heads/", "refs/remotes/")
	if err != nil {
		return nil, err
	}
	defaultName, _ := Base(repo)
	var branches []SourceBranch
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Fields(line)
		if len(parts) != 1 { // Exclude symbolic aliases such as origin/HEAD.
			continue
		}
		b := SourceBranch{Ref: parts[0]}
		if name, ok := strings.CutPrefix(b.Ref, "refs/heads/"); ok {
			b.Name = name
		} else {
			b.Remote, b.Name, _ = strings.Cut(strings.TrimPrefix(b.Ref, "refs/remotes/"), "/")
		}
		b.Default = b.Name == defaultName && (b.Remote == "" || b.Remote == "origin")
		branches = append(branches, b)
	}
	rank := func(b SourceBranch) int {
		if b.Default {
			return 0
		}
		if !strings.Contains(b.Name, "/") {
			return 1
		}
		return 2
	}
	sort.Slice(branches, func(i, j int) bool {
		a, b := branches[i], branches[j]
		if rank(a) != rank(b) {
			return rank(a) < rank(b)
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Remote < b.Remote
	})
	return branches, nil
}

func ResolveSource(repo, ref string) (SourceBranch, error) {
	branches, err := SourceBranches(repo)
	if err != nil {
		return SourceBranch{}, err
	}
	for _, branch := range branches {
		if branch.Ref == ref {
			return branch, nil
		}
	}
	return SourceBranch{}, fmt.Errorf("source branch %q no longer exists; choose an existing branch", ref)
}
