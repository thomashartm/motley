package member

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIdentityInfoPersistsAndCanBeCleared(t *testing.T) {
	dir := ghState(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte("#!/bin/sh\necho 'no sessions'\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	saveTest(t, dir, Manifest{ID: "info-fixture", Name: "Original", Ticket: "42"})
	legacy, err := Load(dir, "info-fixture")
	if err != nil || legacy.Info != "" {
		t.Fatalf("legacy manifest: %+v %v", legacy, err)
	}
	name, info := "New name", "  Diagnose delayed payments  "
	if err := EditIdentity(legacy.ID, IdentityEdit{Name: &name, Info: &info}); err != nil {
		t.Fatal(err)
	}
	saved, err := Load(dir, legacy.ID)
	if err != nil || saved.Name != name || saved.Info != "Diagnose delayed payments" || saved.Ticket != "42" {
		t.Fatalf("identity not persisted: %+v %v", saved, err)
	}
	name = "Renamed again"
	if err := EditIdentity(legacy.ID, IdentityEdit{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if saved, err := Load(dir, legacy.ID); err != nil || saved.Info != "Diagnose delayed payments" {
		t.Fatalf("unrelated edit lost info: %+v %v", saved, err)
	}
	info = "bad\x1b[31mtext"
	if err := EditIdentity(legacy.ID, IdentityEdit{Info: &info}); err == nil {
		t.Fatal("accepted control characters")
	}
	info = ""
	if err := EditIdentity(legacy.ID, IdentityEdit{Info: &info}); err != nil {
		t.Fatal(err)
	}
	if saved, err := Load(dir, legacy.ID); err != nil || saved.Info != "" || saved.Name != name {
		t.Fatalf("could not clear info: %+v %v", saved, err)
	}
}
