package orchestrator

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
	"github.com/YChange01/codex-feishu-link/internal/testutil"
)

func TestTargetPickerSessionOptionsIncludesThreadsWithCrossInstancePollutedWorkspaceKey(t *testing.T) {
	now := time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	root := t.TempDir()
	workspace := filepath.Join(root, "svmpy")
	pool := filepath.Join(root, "headless-pool")

	// An instance serving the real workspace.
	svc.UpsertInstance(&state.InstanceRecord{
		InstanceID:    "inst-svm",
		DisplayName:   "svmpy",
		WorkspaceRoot: workspace,
		WorkspaceKey:  workspace,
		ShortName:     "svmpy",
		Source:        "headless",
		Managed:       true,
		Online:        true,
		Threads: map[string]*state.ThreadRecord{
			"thread-svm": {
				ThreadID:     "thread-svm",
				Name:         "svmnew0910",
				CWD:          workspace,
				WorkspaceKey: pool, // Cross-instance polluted WorkspaceKey
				Loaded:       true,
				LastUsedAt:   now,
			},
		},
	})

	// Attach the surface to the real workspace.
	svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionAttachWorkspace,
		SurfaceSessionID: "surface-1",
		ChatID:           "chat-1",
		ActorUserID:      "user-1",
		WorkspaceKey:     workspace,
	})

	// Trigger ShowThreads (/use)
	events := svc.ApplySurfaceAction(control.Action{
		Kind:             control.ActionShowThreads,
		SurfaceSessionID: "surface-1",
		ChatID:           "chat-1",
		ActorUserID:      "user-1",
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 target picker event, got %#v", events)
	}
	view := targetPickerFromEvent(t, events[0])
	if !testutil.SamePath(view.SelectedWorkspaceKey, workspace) {
		t.Fatalf("expected selected workspace to be %q, got %q", workspace, view.SelectedWorkspaceKey)
	}

	// thread-svm should be present in SessionOptions despite having a polluted WorkspaceKey
	option, ok := targetPickerSessionOption(view, targetPickerThreadValue("thread-svm"))
	if !ok {
		t.Fatalf("expected thread-svm with polluted WorkspaceKey to be included in SessionOptions, got %#v", view.SessionOptions)
	}
	if option.Label != "svmpy · svmnew0910" {
		t.Fatalf("expected option label to be 'svmpy · svmnew0910', got %q", option.Label)
	}
}

func TestEventThreadsSnapshotDoesNotPolluteThreadWorkspaceKeyFromUnrelatedInstance(t *testing.T) {
	now := time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC)
	svc := newServiceForTest(&now)
	root := t.TempDir()
	workspace := filepath.Join(root, "svmpy")
	pool := filepath.Join(root, "headless-pool")

	// A managed headless instance running in the state dir (e.g. headless pool preheat)
	svc.UpsertInstance(&state.InstanceRecord{
		InstanceID:    "inst-headless-pool",
		DisplayName:   "headless-pool",
		WorkspaceRoot: pool,
		WorkspaceKey:  pool,
		ShortName:     "headless-pool",
		Source:        "headless",
		Managed:       true,
		Online:        true,
	})

	// Headless instance emits a snapshot containing threads from another workspace
	// with empty WorkspaceKey
	svc.ApplyAgentEvent("inst-headless-pool", agentproto.Event{
		Kind: agentproto.EventThreadsSnapshot,
		Threads: []agentproto.ThreadSnapshotRecord{
			{
				ThreadID: "thread-svm-snap",
				Name:     "svmnew0910",
				CWD:      workspace,
				Loaded:   true,
			},
		},
	})

	inst := svc.root.Instances["inst-headless-pool"]
	thread := inst.Threads["thread-svm-snap"]
	if thread == nil {
		t.Fatal("expected thread-svm-snap to exist in inst-headless-pool")
	}

	// The thread's WorkspaceKey should NOT be overwritten with inst.WorkspaceKey
	if testutil.SamePath(thread.WorkspaceKey, pool) {
		t.Fatalf("expected thread WorkspaceKey not to be polluted by headless-pool instance, got %q", thread.WorkspaceKey)
	}
	if !testutil.SamePath(thread.WorkspaceKey, workspace) {
		t.Fatalf("expected thread WorkspaceKey to resolve to thread CWD %q, got %q", workspace, thread.WorkspaceKey)
	}
}

func TestEventThreadsSnapshotPreservesValidExistingWorkspaceKey(t *testing.T) {
	for _, tc := range []struct {
		name          string
		instanceRoot  string
		workspaceRoot string
		cwd           string
		snapshotCWD   string
	}{
		{
			name:          "foreign instance preserves parent workspace",
			instanceRoot:  "headless-pool",
			workspaceRoot: "repo",
			cwd:           "repo/web",
			snapshotCWD:   "repo/web",
		},
		{
			name:          "parent instance preserves nested worktree",
			instanceRoot:  "repo",
			workspaceRoot: "repo/.worktrees/feature",
			cwd:           "repo/.worktrees/feature/web",
			snapshotCWD:   "repo/.worktrees/feature/web",
		},
		{
			name:          "omitted cwd preserves known workspace",
			instanceRoot:  "headless-pool",
			workspaceRoot: "repo",
			cwd:           "repo/web",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC)
			svc := newServiceForTest(&now)
			root := t.TempDir()
			instanceRoot := filepath.Join(root, tc.instanceRoot)
			workspaceRoot := filepath.Join(root, tc.workspaceRoot)
			cwd := filepath.Join(root, tc.cwd)
			svc.UpsertInstance(&state.InstanceRecord{
				InstanceID:    "inst-1",
				WorkspaceRoot: instanceRoot,
				WorkspaceKey:  instanceRoot,
				Source:        "headless",
				Managed:       true,
				Online:        true,
				Threads: map[string]*state.ThreadRecord{
					"thread-1": {ThreadID: "thread-1", WorkspaceKey: workspaceRoot, CWD: cwd, Loaded: true},
				},
			})
			snapshotCWD := ""
			if tc.snapshotCWD != "" {
				snapshotCWD = filepath.Join(root, tc.snapshotCWD)
			}
			svc.ApplyAgentEvent("inst-1", agentproto.Event{
				Kind: agentproto.EventThreadsSnapshot,
				Threads: []agentproto.ThreadSnapshotRecord{
					{ThreadID: "thread-1", CWD: snapshotCWD, Loaded: true},
				},
			})
			thread := svc.root.Instances["inst-1"].Threads["thread-1"]
			if !testutil.SamePath(thread.WorkspaceKey, workspaceRoot) {
				t.Fatalf("snapshot changed valid workspace root: got %q, want %q", thread.WorkspaceKey, workspaceRoot)
			}
			if !testutil.SamePath(threadWorkspaceKeyFromRecord(thread), workspaceRoot) {
				t.Fatalf("workspace selection lost stable root: got %q, want %q", threadWorkspaceKeyFromRecord(thread), workspaceRoot)
			}
		})
	}
}
