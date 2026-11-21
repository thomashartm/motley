package blueprint

import (
	"path/filepath"
	"testing"
)

func TestGlobalFilesIncludesRestrictedAndInvalidTemplates(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := GlobalDir()
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "good.md"), "+++\nrepos=['other']\n+++\nGlobal")
	write(t, filepath.Join(dir, "broken.md"), "+++\n+++\n{{")
	write(t, filepath.Join(dir, "ignored.txt"), "not a template")
	files, err := GlobalFiles()
	if err != nil || len(files) != 2 {
		t.Fatal(files, err)
	}
	if files[0].Err == nil || files[0].Source != "+++\n+++\n{{" || files[1].Err != nil || !files[1].Allows("other") {
		t.Fatal(files)
	}
	if _, err := Discover("", ""); err == nil {
		t.Fatal("spawn discovery silently accepted a malformed template")
	}
}

func TestCreateGlobalRejectsInvalidAndConflictingTemplates(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := GlobalDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "existing.md")
	const original = "+++\nname='alias'\n+++\nOriginal"
	write(t, path, original)
	for _, tc := range []struct{ name, body string }{
		{"../escape", "prompt"}, {"empty", " "}, {"broken", "{{"}, {"unknown", "{{.Unknown}}"},
		{"alias", "new"}, {"existing", "new"},
	} {
		if _, err := CreateGlobal(tc.name, tc.body); err == nil {
			t.Errorf("accepted %q", tc.name)
		}
	}
	files, err := GlobalFiles()
	if err != nil || len(files) != 1 || files[0].Source != original {
		t.Fatal(files, err)
	}
}
