package member

import (
	"fmt"
	"os"

	"github.com/thomashartm/motley/internal/state"
	"github.com/thomashartm/motley/internal/terminal"
)

func requireDirectStopped(m Manifest) error {
	if m.LaunchTarget != "ghostty" {
		return nil
	}
	dir, err := state.MembersDir()
	if err != nil {
		return err
	}
	alive, err := terminal.Alive(dir, m.ID)
	if err != nil {
		return err
	}
	if alive {
		return fmt.Errorf("member %s is running in Ghostty; stop the agent in its original Ghostty window, then retry", m.ID)
	}
	return nil
}

// BeginDirect serializes ownership with retire/terminate before the agent starts.
func BeginDirect(id string) (Manifest, *os.File, error) {
	dir, err := state.MembersDir()
	if err != nil {
		return Manifest{}, nil, err
	}
	lifecycle, err := state.LockSpawn(dir)
	if err != nil {
		return Manifest{}, nil, err
	}
	defer func() { _ = lifecycle.Close() }()
	m, err := Load(dir, id)
	if err != nil {
		return Manifest{}, nil, err
	}
	if m.LaunchTarget != "ghostty" {
		return m, nil, fmt.Errorf("member %s is not a direct Ghostty member", id)
	}
	if err := m.CheckCheckout(); err != nil {
		return m, nil, err
	}
	lock, err := terminal.Acquire(dir, id)
	return m, lock, err
}

func directTerminal(m Manifest) error {
	return fmt.Errorf("%s runs directly in Ghostty; use its original Ghostty window, or motley revive %s after it stops", m.ID, m.ID)
}
