package tui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// repositoryView scrolls by repository, accounting for wrapped paths and group
// headings so the selected repository remains visible when moving down the list.
func (f *spawnForm) repositoryView(width, height int) []string {
	matches := f.matches()
	if len(matches) == 0 {
		return []string{"No matching repositories."}
	}
	choice := min(f.choice, len(matches)-1)
	start := choice
	for start > 0 {
		if len(f.repositoryRows(matches[start-1:choice+1], start-1, width)) > height {
			break
		}
		start--
	}
	rows := f.repositoryRows(matches[start:], start, width)
	return rows[:min(len(rows), height)]
}

func (f *spawnForm) repositoryRows(repos []string, start, width int) []string {
	var rows []string
	previousRoot := ""
	for i, path := range repos {
		root := filepath.Dir(path)
		if root != previousRoot {
			if len(rows) > 0 {
				rows = append(rows, "")
			}
			rows = append(rows, strings.Split(ansi.Wrap(clean(root), max(1, width), ""), "\n")...)
			previousRoot = root
		}
		label := clean(filepath.Base(path) + " (" + path + ")")
		wrapped := strings.Split(ansi.Wrap(label, max(1, width-4), ""), "\n")
		for j, line := range wrapped {
			prefix := "    "
			if start+i == f.choice && j == 0 {
				prefix = "  > "
			}
			rows = append(rows, prefix+line)
		}
	}
	return rows
}
