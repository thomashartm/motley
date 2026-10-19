// Package terminal tracks agents running directly in a terminal window.
package terminal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/thomashartm/motley/internal/state"
)

// Acquire holds process ownership until every inherited descriptor is closed.
// Keep the inode in place: deleting lock files would allow concurrent owners.
func Acquire(dir, id string) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(dir, id+".direct.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("member %s already runs in Ghostty: %w", id, err)
	}
	return f, nil
}

// Alive uses kernel ownership rather than a PID that might have been reused.
func Alive(dir, id string) (bool, error) {
	f, err := os.OpenFile(filepath.Join(dir, id+".direct.lock"), os.O_RDWR, 0600)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return true, nil
	}
	return false, err
}

type Status struct {
	Agent   string
	Status  string
	Context string
	Since   int64
	Seen    int64
}

func ReadStatus(dir, id string) (Status, error) {
	var s Status
	data, err := os.ReadFile(filepath.Join(dir, id+".direct.json"))
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}

func WriteStatus(dir, id string, s Status) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return state.WriteAtomic(filepath.Join(dir, id+".direct.json"), data)
}
