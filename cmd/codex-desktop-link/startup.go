package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/YChange01/codex-feishu-link/internal/execlaunch"
)

type desktopStarter struct {
	probe    func(context.Context) bool
	lock     func(context.Context) (func(), error)
	start    func(context.Context) error
	interval time.Duration
}

// Recheck under the process-shared lock: both robots and the IDE may reconnect
// together. A live daemon is never restarted, stopped, or replaced.
func (s desktopStarter) ensure(ctx context.Context) error {
	if s.probe(ctx) {
		return nil
	}
	unlock, err := s.lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	if s.probe(ctx) {
		return nil
	}
	if err := s.start(ctx); err != nil {
		return fmt.Errorf("start shared Codex daemon: %w", err)
	}
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		if s.probe(ctx) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for shared Codex daemon: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func ensureDesktopDaemon(ctx context.Context, home, socket string, stderr io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	starter := desktopStarter{
		probe: func(ctx context.Context) bool {
			conn, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "unix", socket)
			if err != nil {
				return false
			}
			_ = conn.Close()
			return true
		},
		lock: func(ctx context.Context) (func(), error) {
			dir := filepath.Join(home, ".codex", "feishu-desktop")
			if err := os.MkdirAll(dir, 0700); err != nil {
				return nil, err
			}
			return lockDesktopStartup(ctx, filepath.Join(dir, "startup.lock"))
		},
		start: func(ctx context.Context) error {
			fmt.Fprintln(stderr, "codex-desktop-link: shared daemon unavailable; starting it with the official daemon manager")
			cmd := execlaunch.CommandContext(ctx, realCodexBinary, "app-server", "daemon", "start")
			// Do not allow the manager's human output to corrupt JSONL stdout.
			cmd.Stdout, cmd.Stderr = stderr, stderr
			return cmd.Run()
		},
		interval: 100 * time.Millisecond,
	}
	return starter.ensure(ctx)
}
