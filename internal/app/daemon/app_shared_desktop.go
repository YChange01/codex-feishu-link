package daemon

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
	"github.com/YChange01/codex-feishu-link/internal/execlaunch"
)

type sharedDesktopJob struct {
	cancel  context.CancelFunc
	doneAt  time.Time
	command control.DaemonCommand
}

func (a *App) sharedDesktopTargetMatches(command control.DaemonCommand) bool {
	surface := a.service.Surface(command.SurfaceSessionID)
	inst := a.service.Instance(command.InstanceID)
	if surface == nil || inst == nil || !inst.Online || !inst.Capabilities.SharedAppServer || surface.AttachedInstanceID != command.InstanceID || surface.SelectedThreadID != command.ThreadID || command.ThreadID == "" {
		return false
	}
	thread := inst.Threads[command.ThreadID]
	return thread != nil && strings.TrimSpace(thread.CWD) != "" && thread.CWD == command.ThreadCWD
}

func (a *App) handleSharedDesktopOpenLocked(command control.DaemonCommand) []eventcontract.Event {
	cfg := a.headlessRuntime
	if a.shuttingDown || !sharedCodexAppServerEnabled(cfg.BaseEnv) || envMap(cfg.BaseEnv)["CODEX_FEISHU_RELAY_DESKTOP_OPEN"] != "1" || !a.sharedDesktopTargetMatches(command) {
		return nil
	}
	key := command.InstanceID + "\x00" + command.ThreadID
	if a.daemonAsyncRuntime.sharedDesktop == nil {
		a.daemonAsyncRuntime.sharedDesktop = map[string]*sharedDesktopJob{}
	}
	if existing := a.daemonAsyncRuntime.sharedDesktop[key]; existing != nil {
		if existing.doneAt.IsZero() || time.Since(existing.doneAt) < 5*time.Second {
			return nil
		}
		delete(a.daemonAsyncRuntime.sharedDesktop, key)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	job := &sharedDesktopJob{cancel: cancel, command: command}
	a.daemonAsyncRuntime.sharedDesktop[key] = job
	binary, env := cfg.CodexRealBinary, append([]string{}, cfg.BaseEnv...)
	go func() {
		defer cancel()
		// Revalidate after handing work to the worker, before starting the helper.
		a.mu.Lock()
		valid := !a.shuttingDown && a.sharedDesktopTargetMatches(command)
		a.mu.Unlock()
		var err error
		var stderr bytes.Buffer
		if valid {
			cmd := execlaunch.CommandContext(ctx, binary, "open-vscode", "--cwd", command.ThreadCWD, "--thread", command.ThreadID)
			cmd.Env = env
			cmd.Stderr = &stderr
			cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
			cmd.WaitDelay = 3 * time.Second
			// Start under the route lock, but never wait under it.
			a.mu.Lock()
			valid = !a.shuttingDown && a.sharedDesktopTargetMatches(command)
			if valid {
				err = cmd.Start()
			}
			a.mu.Unlock()
			if valid && err == nil {
				err = cmd.Wait()
			}
		}
		a.enqueueDaemonAsyncResult(daemonAsyncResult{Apply: func(ctx context.Context, app *App) {
			if app.daemonAsyncRuntime.sharedDesktop[key] != job {
				return
			}
			if !valid || err != nil || !app.sharedDesktopTargetMatches(command) {
				delete(app.daemonAsyncRuntime.sharedDesktop, key)
				if valid && err != nil && app.sharedDesktopTargetMatches(command) {
					detail := sharedDesktopErrorDetail(err, stderr.String())
					log.Printf("shared desktop open failed: instance=%s thread=%s detail=%q", command.InstanceID, command.ThreadID, detail)
					app.handleUIEventsLocked(ctx, sharedDesktopFailure(command, fmt.Errorf("%s", detail)))
				}
				return
			}
			job.doneAt = time.Now()
		}})
	}()
	return nil
}

func sharedDesktopFailure(command control.DaemonCommand, err error) []eventcontract.Event {
	return []eventcontract.Event{{Kind: eventcontract.KindNotice, SurfaceSessionID: command.SurfaceSessionID, Notice: &control.Notice{Code: "shared_desktop_open_failed", Text: "后台会话已连接，但 VS Code 窗口打开失败。可重新选择会话重试；飞书任务仍可继续。", Sections: []control.FeishuCardTextSection{control.CommandCatalogTextSection("", "后台会话已连接，但 VS Code 窗口打开失败。可重新选择会话重试；飞书任务仍可继续。"), control.CommandCatalogTextSection("错误详情", err.Error())}}}}
}

func (a *App) cancelSharedDesktopJobsLocked() {
	for _, job := range a.daemonAsyncRuntime.sharedDesktop {
		if job != nil && job.cancel != nil {
			job.cancel()
		}
	}
}

func sharedDesktopErrorDetail(err error, stderr string) string {
	detail := []rune(strings.TrimSpace(stderr))
	if len(detail) > 1000 {
		detail = detail[len(detail)-1000:]
	}
	if len(detail) == 0 {
		return err.Error()
	}
	return err.Error() + ": " + string(detail)
}

// Cancel obsolete navigation before its helper can consume or finish a UI job.
func (a *App) reconcileSharedDesktopJobsLocked(now time.Time) {
	for key, job := range a.daemonAsyncRuntime.sharedDesktop {
		if job == nil {
			delete(a.daemonAsyncRuntime.sharedDesktop, key)
			continue
		}
		if !a.sharedDesktopTargetMatches(job.command) {
			job.cancel()
			delete(a.daemonAsyncRuntime.sharedDesktop, key)
		} else if !job.doneAt.IsZero() && !now.Before(job.doneAt.Add(5*time.Second)) {
			delete(a.daemonAsyncRuntime.sharedDesktop, key)
		}
	}
}
