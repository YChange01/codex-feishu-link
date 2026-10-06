// codex-desktop-link connects the Feishu wrapper to an existing local Codex daemon.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/YChange01/codex-feishu-link/internal/execlaunch"
)

const realCodexBinary = "/usr/local/bin/codex"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.UserHomeDir, os.Getenv, runCodex))
}

type passthroughFunc func(context.Context, []string, io.Reader, io.Writer, io.Writer) error

func run(ctx context.Context, args []string, stdin io.ReadCloser, stdout io.WriteCloser, stderr io.Writer, homeDir func() (string, error), getenv func(string) string, passthrough passthroughFunc) int {
	if len(args) > 0 && args[0] == "open-vscode" {
		if err := openVSCode(ctx, args[1:], homeDir, getenv, stderr); err != nil {
			fmt.Fprintln(stderr, "codex-desktop-link:", err)
			return 1
		}
		return 0
	}
	shared, err := sharedServerInvocation(args)
	if err != nil {
		fmt.Fprintln(stderr, "codex-desktop-link:", err)
		return 2
	}
	if !shared {
		if err := passthrough(ctx, args, stdin, stdout, stderr); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return exitErr.ExitCode()
			}
			fmt.Fprintln(stderr, "codex-desktop-link: Codex command failed:", err)
			return 1
		}
		return 0
	}
	home, err := homeDir()
	if err == nil {
		err = validateSharedEnvironment(home, getenv)
	}
	if err != nil {
		fmt.Fprintln(stderr, "codex-desktop-link:", err)
		return 2
	}
	stdin, stdout, cleanup, err := interruptibleStdio(stdin, stdout)
	if err != nil {
		fmt.Fprintln(stderr, "codex-desktop-link:", err)
		return 2
	}
	defer cleanup()
	fmt.Fprintln(stderr, "codex-desktop-link: using the shared desktop Codex daemon; it will be started only if unavailable. Existing configuration is preserved.")
	socket := filepath.Join(home, ".codex", "app-server-control", "app-server-control.sock")
	if err := ensureDesktopDaemon(ctx, home, socket, stderr); err != nil {
		fmt.Fprintln(stderr, "codex-desktop-link:", err)
		return 1
	}
	err = bridge(ctx, socket, stdin, stdout, maxMessageBytes)
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(stderr, "codex-desktop-link:", err)
		return 1
	}
	return 0
}

func runCodex(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := execlaunch.CommandContext(ctx, realCodexBinary, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	return cmd.Run()
}
