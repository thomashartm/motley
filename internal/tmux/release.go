package tmux

import (
	"fmt"
	"strconv"
	"time"
)

// Release preserves a former agent terminal while freeing the member's name.
// Its old hooks can no longer address the new member through this tmux session.
func Release(id string) error {
	sessions, err := Sessions()
	if err != nil {
		return err
	}
	for _, s := range sessions {
		if s.Name != SessionName(id) {
			continue
		}
		if s.MemberID != id {
			return fmt.Errorf("session %s is not owned by this member", s.Name)
		}
		name := "untracked-" + s.Name + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		_, err := run("rename-session", "-t", "="+s.Name+":", name,
			";", "set-option", "-u", "-t", "="+name+":", "@motley_member")
		return err
	}
	return nil
}
