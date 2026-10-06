package daemon

import (
	"testing"
	"time"

	"github.com/YChange01/codex-feishu-link/internal/app/daemon/goalinterlockstore"
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/orchestrator"
	relayruntime "github.com/YChange01/codex-feishu-link/internal/runtime"
)

func TestGoalInterlockStateRestoreAndPersist(t *testing.T) {
	stateDir := t.TempDir()
	store := goalinterlockstore.NewStore(goalinterlockstore.StatePath(stateDir))
	lease := orchestrator.GoalInterlockLease{
		InstanceID:       "inst-1",
		ThreadID:         "thread-1",
		Phase:            orchestrator.GoalInterlockDraining,
		Objective:        "ship it",
		TriggerSurfaceID: "surface-1",
	}
	if err := store.ReplaceAll([]orchestrator.GoalInterlockLease{lease}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	app := New(":0", ":0", nil, agentproto.ServerIdentity{})
	app.SetHeadlessRuntime(HeadlessRuntimeConfig{Paths: relayruntime.Paths{StateDir: stateDir}})

	restored := app.service.GoalInterlockLeases()
	if len(restored) != 1 || restored[0].InstanceID != "inst-1" || restored[0].ThreadID != "thread-1" ||
		restored[0].Phase != orchestrator.GoalInterlockDraining {
		t.Fatalf("expected restored lease, got %#v", restored)
	}

	app.service.RestoreGoalInterlockLeases(nil)
	app.persistGoalInterlockStateLocked(time.Unix(2, 0))
	loaded, err := goalinterlockstore.LoadStore(goalinterlockstore.StatePath(stateDir))
	if err != nil {
		t.Fatalf("LoadStore: %v", err)
	}
	if leases := loaded.Leases(); len(leases) != 0 {
		t.Fatalf("expected cleared persisted leases, got %#v", leases)
	}
}
