package main

import (
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/thomashartm/motley/internal/member"
	"github.com/thomashartm/motley/internal/report"
	"github.com/thomashartm/motley/internal/state"
	"github.com/thomashartm/motley/internal/terminal"
)

func runDirectCommand() *cobra.Command {
	var resume bool
	cmd := &cobra.Command{
		Use: "run-direct <id>", Hidden: true, Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// Window creation can finish before SpawnPrepared releases its lock.
			var m member.Manifest
			var lock *os.File
			var err error
			deadline := time.Now().Add(5 * time.Second)
			for {
				m, lock, err = member.BeginDirect(args[0])
				if err == nil || !strings.Contains(err.Error(), "another member lifecycle command") || time.Now().After(deadline) {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if err != nil {
				return err
			}
			defer func() { _ = lock.Close() }()
			_ = os.Unsetenv("TMUX")
			_ = os.Unsetenv("TMUX_PANE")
			_ = os.Setenv("MOTLEY_MEMBER", m.ID)
			dir, err := state.MembersDir()
			if err != nil {
				return err
			}
			now := time.Now().Unix()
			if err := terminal.WriteStatus(dir, m.ID, terminal.Status{Agent: m.Agent, Status: m.LaunchStatus(resume), Since: now, Seen: now}); err != nil {
				return err
			}
			bin, err := os.Executable()
			if err != nil {
				return err
			}
			argv := []string{"exec-agent", m.ID}
			if resume {
				argv = append(argv, "--resume")
			}
			child := exec.Command(bin, argv...)
			child.Dir, child.Stdin, child.Stdout, child.Stderr = m.Worktree, os.Stdin, os.Stdout, os.Stderr
			// The child retains ownership even if the wrapper is killed unexpectedly.
			child.ExtraFiles = []*os.File{lock}
			signals := make(chan os.Signal, 4)
			signal.Notify(signals, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
			defer signal.Stop(signals)
			if err := child.Start(); err != nil {
				report.Exited(m.ID)
				return err
			}
			done := make(chan struct{})
			defer close(done)
			go func() {
				for {
					select {
					case sig := <-signals:
						// Ctrl-C already reaches the foreground agent; keep its runner alive.
						if sig != syscall.SIGINT {
							_ = child.Process.Signal(sig)
						}
					case <-done:
						return
					}
				}
			}()
			err = child.Wait()
			report.Exited(m.ID)
			return err
		},
	}
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume the latest recorded conversation")
	return cmd
}
