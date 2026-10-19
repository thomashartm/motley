package terminal

import "testing"

func TestOwnershipIsReleasedWithLock(t *testing.T) {
	dir := t.TempDir()
	lock, err := Acquire(dir, "test")
	if err != nil {
		t.Fatal(err)
	}
	if live, err := Alive(dir, "test"); err != nil || !live {
		t.Fatalf("live=%v err=%v", live, err)
	}
	if duplicate, err := Acquire(dir, "test"); err == nil {
		_ = duplicate.Close()
		t.Fatal("duplicate owner accepted")
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	if live, err := Alive(dir, "test"); err != nil || live {
		t.Fatalf("stale lock treated as live: %v %v", live, err)
	}
}
