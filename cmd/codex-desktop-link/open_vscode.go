package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/YChange01/codex-feishu-link/internal/execlaunch"
)

type desktopOpenRequest struct {
	Version       int       `json:"version"`
	ID            string    `json:"id"`
	CWD           string    `json:"cwd"`
	ThreadID      string    `json:"threadId"`
	CreatedAt     time.Time `json:"createdAt"`
	DaemonEpoch   string    `json:"daemonEpoch"`
	CLIExecutable string    `json:"cliExecutable"`
}

type desktopOpenResult struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	ThreadID string `json:"threadId"`
	CWD      string `json:"cwd"`
	Error    string `json:"error"`
}

func openVSCode(ctx context.Context, args []string, homeDir func() (string, error), getenv func(string) string, stderr io.Writer) error {
	flags := flag.NewFlagSet("open-vscode", flag.ContinueOnError)
	flags.SetOutput(stderr)
	cwd := flags.String("cwd", "", "selected thread's project directory")
	thread := flags.String("thread", "", "selected Codex thread id")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`).MatchString(*thread) || !filepath.IsAbs(*cwd) {
		return fmt.Errorf("open-vscode requires an absolute --cwd and a valid --thread")
	}
	project, err := filepath.EvalSymlinks(*cwd)
	if err != nil {
		return err
	}
	info, err := os.Stat(project)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("selected project directory is unavailable")
	}
	home, err := homeDir()
	if err != nil {
		return err
	}
	if err := validateSharedEnvironment(home, getenv); err != nil {
		return err
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("VS Code desktop opening is currently configured for macOS only")
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	socket := filepath.Join(home, ".codex", "app-server-control", "app-server-control.sock")
	if err := ensureDesktopDaemon(ctx, home, socket, stderr); err != nil {
		return err
	}
	socketPath, err := filepath.EvalSymlinks(socket)
	if err != nil {
		return err
	}
	socketInfo, err := os.Stat(socket)
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	launcher := filepath.Join(filepath.Dir(executable), "codex-vscode-link")
	if _, err := os.Stat(launcher); err != nil {
		return fmt.Errorf("shared VS Code launcher is missing: %w", err)
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	request := desktopOpenRequest{Version: 1, ID: hex.EncodeToString(random[:]), CWD: project, ThreadID: *thread,
		CreatedAt: time.Now().UTC(), DaemonEpoch: fmt.Sprintf("%s:%d", socketPath, socketInfo.ModTime().UnixNano()), CLIExecutable: launcher}
	dir := filepath.Join(home, ".codex", "feishu-desktop", "requests")
	return deliverDesktopOpen(ctx, dir, request, 1500*time.Millisecond, 100*time.Millisecond, func(ctx context.Context) error {
		code := "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
		folder := (&url.URL{Scheme: "file", Path: project}).String()
		cmd := execlaunch.CommandContext(ctx, code, "--new-window", "--folder-uri", folder)
		cmd.Stdout, cmd.Stderr = stderr, stderr
		return cmd.Run()
	})
}

// An existing matching VS Code window claims the request before we launch a
// new one. The private file mailbox avoids URI authorization prompts and
// routing the selected conversation into the wrong VS Code project.
func deliverDesktopOpen(ctx context.Context, dir string, request desktopOpenRequest, grace, interval time.Duration, launch func(context.Context) error) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, request.ID+".json")
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path+".tmp", data, 0600); err != nil {
		return err
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		_ = os.Remove(path + ".tmp")
		return err
	}
	defer os.Remove(path)
	defer os.Remove(path + ".tmp")
	defer os.Remove(filepath.Join(dir, request.ID+".claim"))
	resultPath := filepath.Join(dir, request.ID+".result.json")
	defer os.Remove(resultPath)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	deadline := time.Now().Add(grace)
	launched := false
	for {
		if raw, err := os.ReadFile(resultPath); err == nil {
			var result desktopOpenResult
			if err := json.Unmarshal(raw, &result); err != nil {
				return fmt.Errorf("invalid VS Code acknowledgement: %w", err)
			}
			if result.ID != request.ID || result.ThreadID != request.ThreadID || result.CWD != request.CWD {
				return fmt.Errorf("VS Code acknowledged a different project or conversation")
			}
			if result.Status != "opened" {
				return fmt.Errorf("VS Code could not open the selected conversation: %s", result.Error)
			}
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
		if !launched && !time.Now().Before(deadline) {
			if _, err := os.Stat(filepath.Join(dir, request.ID+".claim")); os.IsNotExist(err) {
				if err := launch(ctx); err != nil {
					return fmt.Errorf("launch VS Code project: %w", err)
				}
			}
			launched = true
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for VS Code to open the selected conversation (check Codex Feishu Relay Desktop extension): %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
