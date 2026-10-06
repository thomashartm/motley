package tui

import (
	"github.com/thomashartm/motley/internal/worktree"
)

func (f *spawnForm) sourceMatches() []worktree.SourceBranch {
	var matches []worktree.SourceBranch
	for _, source := range f.sources {
		if match(f.query.Value(), source.Label()) {
			matches = append(matches, source)
		}
	}
	return matches
}

func (f *spawnForm) sourceView(width, height int) []string {
	matches := f.sourceMatches()
	if len(matches) == 0 {
		return []string{"No matching source branches."}
	}
	choice := min(f.choice, len(matches)-1)
	start := choice
	for start > 0 {
		if len(f.sourceRows(matches[start-1:choice+1], start-1, width)) > height {
			break
		}
		start--
	}
	rows := f.sourceRows(matches[start:], start, width)
	// On very small terminals, keep the selected branch visible before its heading.
	if height == 1 && len(rows) > 1 {
		return rows[1:2]
	}
	return rows[:min(len(rows), height)]
}

func (f *spawnForm) sourceRows(sources []worktree.SourceBranch, start, width int) []string {
	var rows []string
	previous := ""
	for i, source := range sources {
		if group := source.Group(); group != previous {
			if len(rows) > 0 {
				rows = append(rows, "")
			}
			rows = append(rows, "["+group+"]")
			previous = group
		}
		prefix := "  "
		if start+i == f.choice {
			prefix = "> "
		}
		rows = append(rows, prefix+fit(clean(source.Label()), max(1, width-2)))
	}
	return rows
}
