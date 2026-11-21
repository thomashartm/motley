package blueprint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// CreateGlobal publishes a complete template without replacing an existing file.
func CreateGlobal(name, prompt string) (string, error) {
	if !namePattern.MatchString(name) {
		return "", fmt.Errorf("start the name with a letter or digit and use only letters, digits, dots, underscores or hyphens")
	}
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("enter a prompt before saving")
	}
	if err := ValidatePrompt(prompt); err != nil {
		return "", err
	}
	dir, err := GlobalDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, name+".md")
	source := []byte("+++\nschema = 1\n+++\n" + prompt)
	b, err := Parse(path, source)
	if err != nil {
		return "", err
	}
	if _, err := b.Render(Data{}); err != nil {
		return "", err
	}
	files, err := GlobalFiles()
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if f.Name == name || f.Path == path {
			return "", fmt.Errorf("template %q already exists", name)
		}
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".new-template-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.Write(source)
	closeErr := tmp.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	return path, os.Link(tmp.Name(), path)
}
