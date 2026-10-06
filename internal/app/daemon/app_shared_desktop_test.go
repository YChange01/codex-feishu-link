package daemon

import (
	"context"
	"errors"
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
	"runtime"
	"strings"
	"testing"
	"time"
)

func sharedDesktopTestApp() (*App, control.DaemonCommand) {
	app := New(":0", ":0", nil, agentproto.ServerIdentity{})
	app.service.UpsertInstance(&state.InstanceRecord{InstanceID: "desktop", Online: true, Capabilities: agentproto.Capabilities{SharedAppServer: true}, Threads: map[string]*state.ThreadRecord{"thread": {ThreadID: "thread", CWD: "/exact/project"}}})
	app.service.ApplySurfaceAction(control.Action{Kind: control.ActionStatus, SurfaceSessionID: "surface"})
	surface := app.service.Surface("surface")
	surface.AttachedInstanceID, surface.SelectedThreadID = "desktop", "thread"
	app.headlessRuntime.BaseEnv = []string{"CODEX_FEISHU_RELAY_SHARED_APP_SERVER=1", "CODEX_FEISHU_RELAY_DESKTOP_OPEN=1"}
	return app, control.DaemonCommand{Kind: control.DaemonCommandOpenSharedDesktop, SurfaceSessionID: "surface", InstanceID: "desktop", ThreadID: "thread", ThreadCWD: "/exact/project"}
}

func TestSharedDesktopRejectsStaleRouteAndWrongProject(t *testing.T) {
	app, command := sharedDesktopTestApp()
	if !app.sharedDesktopTargetMatches(command) {
		t.Fatal("exact target rejected")
	}
	command.ThreadCWD = "/wrong"
	if app.sharedDesktopTargetMatches(command) {
		t.Fatal("wrong project accepted")
	}
	app.handleSharedDesktopOpenLocked(command)
	if len(app.daemonAsyncRuntime.sharedDesktop) != 0 {
		t.Fatal("invalid route started job")
	}
	command.ThreadCWD = "/exact/project"
	app.service.Surface("surface").SelectedThreadID = "new-selection"
	if app.sharedDesktopTargetMatches(command) {
		t.Fatal("stale thread accepted")
	}
}

func TestSharedDesktopOptOutAndFailureNotice(t *testing.T) {
	app, command := sharedDesktopTestApp()
	app.headlessRuntime.BaseEnv = []string{"CODEX_FEISHU_RELAY_SHARED_APP_SERVER=1"}
	app.handleSharedDesktopOpenLocked(command)
	if len(app.daemonAsyncRuntime.sharedDesktop) != 0 {
		t.Fatal("opt out started job")
	}
	events := sharedDesktopFailure(command, errors.New("helper failed"))
	if len(events) != 1 || events[0].SurfaceSessionID != "surface" || events[0].Notice.Code != "shared_desktop_open_failed" {
		t.Fatal("failure must be visible to requesting surface")
	}
}

func TestSharedDesktopAsyncFailureAllowsRetry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses system false command")
	}
	app, command := sharedDesktopTestApp()
	app.headlessRuntime.CodexRealBinary = "/usr/bin/false"
	app.mu.Lock()
	app.handleSharedDesktopOpenLocked(command)
	app.handleSharedDesktopOpenLocked(command)
	if len(app.daemonAsyncRuntime.sharedDesktop) != 1 {
		t.Fatal("expected single flight")
	}
	app.mu.Unlock()
	deadline := time.Now().Add(3 * time.Second)
	for {
		app.mu.Lock()
		remaining := len(app.daemonAsyncRuntime.sharedDesktop)
		app.mu.Unlock()
		if remaining == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("failed helper must release retry gate")
		}
		time.Sleep(10 * time.Millisecond)
	}
	app.mu.Lock()
	app.handleSharedDesktopOpenLocked(command)
	if len(app.daemonAsyncRuntime.sharedDesktop) != 1 {
		t.Fatal("failed open could not be retried")
	}
	app.shuttingDown = true
	app.cancelSharedDesktopJobsLocked()
	app.mu.Unlock()
}

func TestSharedDesktopRouteChangeCancelsPendingNavigation(t *testing.T) {
	app, command := sharedDesktopTestApp()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.daemonAsyncRuntime.sharedDesktop = map[string]*sharedDesktopJob{"key": {command: command, cancel: cancel}}
	app.service.Surface("surface").SelectedThreadID = "other"
	app.reconcileSharedDesktopJobsLocked(time.Now())
	select {
	case <-ctx.Done():
	default:
		t.Fatal("obsolete navigation not canceled")
	}
	if len(app.daemonAsyncRuntime.sharedDesktop) != 0 {
		t.Fatal("obsolete job retained")
	}
}

func TestSharedDesktopSuccessfulNavigationExpires(t *testing.T) {
	app, command := sharedDesktopTestApp()
	now := time.Now()
	app.daemonAsyncRuntime.sharedDesktop = map[string]*sharedDesktopJob{"key": {command: command, cancel: func() {}, doneAt: now}}
	app.reconcileSharedDesktopJobsLocked(now.Add(4 * time.Second))
	if len(app.daemonAsyncRuntime.sharedDesktop) != 1 {
		t.Fatal("dedup expired early")
	}
	app.reconcileSharedDesktopJobsLocked(now.Add(5 * time.Second))
	if len(app.daemonAsyncRuntime.sharedDesktop) != 0 {
		t.Fatal("successful navigation prevents explicit reopening")
	}
}

func TestSharedDesktopErrorDetailIsBoundedAndPlain(t *testing.T) {
	detail := sharedDesktopErrorDetail(errors.New("exit status 1"), strings.Repeat("长", 1200)+"<at id=all>**failure**")
	if len([]rune(detail)) > 1020 || !strings.Contains(detail, "failure") {
		t.Fatal("stderr tail not preserved or bounded")
	}
	_, command := sharedDesktopTestApp()
	notice := sharedDesktopFailure(command, errors.New(detail))[0].Notice
	if strings.Contains(notice.Text, "failure") || len(notice.Sections) != 2 {
		t.Fatal("dynamic error must use structured plain text")
	}
}
