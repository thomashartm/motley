package blueprint

import (
	"os"
	"path/filepath"
)

// File retains invalid templates so the manager can still display and repair
// them. Spawn discovery remains strict about invalid or duplicate definitions.
type File struct {
	Blueprint
	Err error
}

func GlobalFiles() ([]File, error) {
	dir, err := GlobalDir()
	if err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}
	files := make([]File, 0, len(paths))
	for _, path := range paths {
		src, err := os.ReadFile(path)
		b := Blueprint{Path: path, Name: filepath.Base(path)}
		if err == nil {
			b, err = Parse(path, src)
		}
		files = append(files, File{Blueprint: b, Err: err})
	}
	return files, nil
}
