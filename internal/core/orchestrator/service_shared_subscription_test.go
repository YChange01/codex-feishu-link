package orchestrator

import (
	"testing"
	"time"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
)

func sharedSubscriptionFixture(t *testing.T, now *time.Time) (*Service, *agentproto.Command) {
	t.Helper()
	svc := newReplyAutoSteerServiceFixture(now)
	inst := svc.root.Instances["inst-1"]
	inst.Capabilities = agentproto.DefaultCapabilitiesForBackend(agentproto.BackendCodex)
	inst.Capabilities.SharedAppServer = true
	inst.CapabilitiesDeclared = true
	events := svc.ApplySurfaceAction(control.Action{Kind: control.ActionUseThread, SurfaceSessionID: "surface-1", ThreadID: "thread-1"})
	command := findAgentCommand(events, agentproto.CommandThreadSubscribe)
	if command == nil || command.Target.ThreadID != "thread-1" {
		t.Fatalf("selection must subscribe without sending a prompt: %#v", events)
	}
	return svc, command
}

func sharedSubscriptionResult(command *agentproto.Command, turnID string) agentproto.Event {
	return agentproto.Event{Kind: agentproto.EventThreadSubscribed, CommandID: command.CommandID, ThreadID: command.Target.ThreadID, TurnID: turnID, Status: "ready"}
}

func TestSharedSubscriptionQueuesInputUntilIdleSnapshotReady(t *testing.T) {
	now := time.Now()
	svc, command := sharedSubscriptionFixture(t, &now)
	events := svc.ApplySurfaceAction(control.Action{Kind: control.ActionTextMessage, SurfaceSessionID: "surface-1", MessageID: "msg-1", Text: "continue"})
	if findAgentCommand(events, agentproto.CommandPromptSend) != nil {
		t.Fatal("prompt raced ahead of the desktop subscription")
	}
	surface := svc.root.Surfaces["surface-1"]
	if len(surface.QueuedQueueItemIDs) != 1 || surface.ActiveQueueItemID != "" {
		t.Fatalf("input must remain queued: %#v", surface)
	}
	events = svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(command, ""))
	prompt := findAgentCommand(events, agentproto.CommandPromptSend)
	if prompt == nil || prompt.Target.ThreadID != "thread-1" {
		t.Fatalf("ready idle subscription must dispatch to the same thread: %#v", events)
	}
}

func TestSharedSubscriptionDesktopTurnCanSteerStopAndDeliverOutput(t *testing.T) {
	now := time.Now()
	svc, command := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(command, "desktop-turn"))
	inst := svc.root.Instances["inst-1"]
	if inst.ActiveTurnID != "desktop-turn" || inst.ActiveThreadID != "thread-1" {
		t.Fatalf("real desktop turn was not projected: %#v", inst)
	}
	events := svc.ApplySurfaceAction(control.Action{Kind: control.ActionTextMessage, SurfaceSessionID: "surface-1", MessageID: "supplement", Text: "check tests first"})
	steer := findAgentCommand(events, agentproto.CommandTurnSteer)
	if steer == nil || steer.Target.ThreadID != "thread-1" || steer.Target.TurnID != "desktop-turn" {
		t.Fatalf("ordinary Feishu input must steer the selected desktop turn immediately: %#v", events)
	}
	if len(steer.Prompt.Inputs) != 1 || steer.Prompt.Inputs[0].Text != "check tests first" {
		t.Fatalf("unexpected immediate steer payload: %#v", steer.Prompt.Inputs)
	}
	svc.BindPendingRemoteCommand("surface-1", "cmd-shared-steer")
	svc.HandleCommandAccepted("inst-1", agentproto.CommandAck{CommandID: "cmd-shared-steer", Accepted: true})
	if item := svc.root.Surfaces["surface-1"].QueueItems["queue-1"]; item == nil || item.Status != state.QueueItemSteered {
		t.Fatalf("accepted input should leave the queue as steered: %#v", item)
	}
	events = svc.ApplySurfaceAction(control.Action{Kind: control.ActionStop, SurfaceSessionID: "surface-1"})
	stop := findAgentCommand(events, agentproto.CommandTurnInterrupt)
	if stop == nil || stop.Target.ThreadID != "thread-1" || stop.Target.TurnID != "desktop-turn" {
		t.Fatalf("stop must interrupt only the selected desktop turn: %#v", events)
	}
	svc.ApplyAgentEvent("inst-1", agentproto.Event{Kind: agentproto.EventItemCompleted, ThreadID: "thread-1", TurnID: "desktop-turn", ItemID: "desktop-message", ItemKind: "agent_message", Metadata: map[string]any{"text": "desktop result"}})
	events = svc.ApplyAgentEvent("inst-1", agentproto.Event{Kind: agentproto.EventTurnCompleted, ThreadID: "thread-1", TurnID: "desktop-turn", Status: "completed", Initiator: agentproto.Initiator{Kind: agentproto.InitiatorLocalUI}})
	for _, event := range events {
		if event.Block != nil && event.Block.Final && event.Block.Text == "desktop result" && event.SurfaceSessionID == "surface-1" {
			return
		}
	}
	t.Fatalf("desktop output must reach the selected Feishu surface: %#v", events)
}

func TestSharedDesktopInputsArrivingDuringSteerAreSentAfterCurrentSteerAck(t *testing.T) {
	now := time.Now()
	svc, command := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(command, "desktop-turn"))
	first := svc.ApplySurfaceAction(control.Action{Kind: control.ActionTextMessage, SurfaceSessionID: "surface-1", MessageID: "first", Text: "第一条补充"})
	if findAgentCommand(first, agentproto.CommandTurnSteer) == nil {
		t.Fatalf("first Feishu input should steer immediately: %#v", first)
	}
	svc.BindPendingRemoteCommand("surface-1", "cmd-first-steer")
	second := svc.ApplySurfaceAction(control.Action{Kind: control.ActionTextMessage, SurfaceSessionID: "surface-1", MessageID: "second", Text: "第二条补充"})
	if findAgentCommand(second, agentproto.CommandTurnSteer) != nil {
		t.Fatalf("second input must wait while the first steer is pending: %#v", second)
	}
	if len(svc.root.Surfaces["surface-1"].QueuedQueueItemIDs) != 1 {
		t.Fatalf("second input should remain queued until the first steer is acknowledged")
	}
	accepted := svc.HandleCommandAccepted("inst-1", agentproto.CommandAck{CommandID: "cmd-first-steer", Accepted: true})
	next := findAgentCommand(accepted, agentproto.CommandTurnSteer)
	if next == nil || next.Target.TurnID != "desktop-turn" || len(next.Prompt.Inputs) != 1 || next.Prompt.Inputs[0].Text != "第二条补充" {
		t.Fatalf("queued Feishu input should automatically steer after the previous ack: %#v", accepted)
	}
}

func TestSharedSubscriptionIgnoresOtherThreadAndWrongProxyEvents(t *testing.T) {
	now := time.Now()
	svc, command := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(command, "desktop-turn"))
	events := svc.ApplyAgentEvent("inst-1", agentproto.Event{Kind: agentproto.EventTurnStarted, ThreadID: "other-thread", TurnID: "other-turn", Initiator: agentproto.Initiator{Kind: agentproto.InitiatorLocalUI}})
	if len(events) != 0 || svc.root.Instances["inst-1"].ActiveTurnID != "desktop-turn" {
		t.Fatalf("unselected thread overwrote the active target: %#v", events)
	}
	other := *svc.root.Instances["inst-1"]
	other.InstanceID = "other-proxy"
	svc.UpsertInstance(&other)
	svc.sharedSubscriptions[other.InstanceID] = &sharedThreadSubscription{ThreadID: "thread-1", Ready: true}
	events = svc.ApplyAgentEvent(other.InstanceID, agentproto.Event{Kind: agentproto.EventItemCompleted, ThreadID: "thread-1", TurnID: "desktop-turn", ItemID: "duplicate", ItemKind: "agent_message", Metadata: map[string]any{"text": "duplicate"}})
	if len(events) != 0 {
		t.Fatalf("another proxy must not forward duplicate output to this surface: %#v", events)
	}
	inst := svc.root.Instances["inst-1"]
	inst.ActiveThreadID = "other-thread"
	if _, _, ok := svc.interruptibleSurfaceTurn(svc.root.Surfaces["surface-1"]); ok {
		t.Fatal("stop selected an unrelated daemon turn")
	}
}

func TestSharedSubscriptionFailureTimeoutAndReconnectCanRetry(t *testing.T) {
	for _, failure := range []string{"response", "timeout", "rejected"} {
		t.Run(failure, func(t *testing.T) {
			now := time.Now()
			svc, first := sharedSubscriptionFixture(t, &now)
			svc.ApplySurfaceAction(control.Action{Kind: control.ActionTextMessage, SurfaceSessionID: "surface-1", MessageID: "queued", Text: "continue"})
			switch failure {
			case "response":
				event := sharedSubscriptionResult(first, "")
				event.Status, event.ErrorMessage = "failed", "resume failed"
				svc.ApplyAgentEvent("inst-1", event)
			case "timeout":
				now = now.Add(31 * time.Second)
				svc.Tick(now)
			case "rejected":
				svc.HandleCommandRejected("inst-1", agentproto.CommandAck{CommandID: first.CommandID, Error: "socket disconnected"})
			}
			events := svc.ApplySurfaceAction(control.Action{Kind: control.ActionStatus, SurfaceSessionID: "surface-1"})
			retry := findAgentCommand(events, agentproto.CommandThreadSubscribe)
			if retry == nil || retry.CommandID == first.CommandID {
				t.Fatalf("failed subscription needs a fresh retry: %#v", events)
			}
			if events := svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(first, "old-turn")); len(events) != 0 || svc.sharedThreadSubscriptionReady("inst-1", "thread-1") {
				t.Fatal("late response from the failed attempt became current")
			}
			svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(retry, ""))
		})
	}
	t.Run("reconnect pinned route", func(t *testing.T) {
		now := time.Now()
		svc, command := sharedSubscriptionFixture(t, &now)
		svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(command, ""))
		inst := svc.root.Instances["inst-1"]
		svc.ApplyInstanceTransportDegraded(inst.InstanceID, true)
		inst.Online = true
		svc.UpsertInstance(inst)
		events := svc.ApplyInstanceConnected(inst.InstanceID)
		if findAgentCommand(events, agentproto.CommandThreadSubscribe) == nil {
			t.Fatalf("reconnect must resubscribe the restored selection automatically: %#v", events)
		}
	})
}

func TestSharedSubscriptionNewThreadAcceptsOnlyCorrelatedCreate(t *testing.T) {
	now := time.Now()
	svc, command := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(command, ""))
	svc.ApplySurfaceAction(control.Action{Kind: control.ActionNewThread, SurfaceSessionID: "surface-1"})
	events := svc.ApplySurfaceAction(control.Action{Kind: control.ActionTextMessage, SurfaceSessionID: "surface-1", MessageID: "new-prompt", Text: "new task"})
	prompt := findAgentCommand(events, agentproto.CommandPromptSend)
	if prompt == nil {
		t.Fatalf("new thread prompt missing: %#v", events)
	}
	prompt.CommandID = "new-command"
	svc.bindPendingRemoteCommand(svc.root.Surfaces["surface-1"], prompt.CommandID)
	event := agentproto.Event{Kind: agentproto.EventThreadSubscribed, ThreadID: "new-thread", CommandID: "unrelated-command", Status: "created"}
	svc.ApplyAgentEvent("inst-1", event)
	if svc.sharedThreadSubscriptionReady("inst-1", "new-thread") {
		t.Fatal("uncorrelated created thread changed the subscription")
	}
	event.CommandID = prompt.CommandID
	svc.ApplyAgentEvent("inst-1", event)
	svc.ApplyAgentEvent("inst-1", agentproto.Event{Kind: agentproto.EventTurnStarted, ThreadID: "new-thread", TurnID: "new-turn", Initiator: agentproto.Initiator{Kind: agentproto.InitiatorRemoteSurface, SurfaceSessionID: "surface-1"}})
	if !svc.sharedThreadSubscriptionReady("inst-1", "new-thread") {
		t.Fatalf("created thread was not adopted as the ready target: %#v", svc.root.Surfaces["surface-1"])
	}
	if svc.root.Surfaces["surface-1"].QueueItems[svc.root.Surfaces["surface-1"].ActiveQueueItemID].Status != state.QueueItemRunning {
		t.Fatal("new remote turn was not accepted")
	}
	svc.ApplyAgentEvent("inst-1", agentproto.Event{Kind: agentproto.EventTurnCompleted, ThreadID: "new-thread", TurnID: "new-turn", Status: "completed", Initiator: agentproto.Initiator{Kind: agentproto.InitiatorRemoteSurface, SurfaceSessionID: "surface-1"}})
	if svc.root.Surfaces["surface-1"].SelectedThreadID != "new-thread" {
		t.Fatal("completed new thread was not committed to the surface")
	}
}

func TestSharedSubscriptionDesktopRaceCannotConsumeOrFailPendingRemoteTurn(t *testing.T) {
	now := time.Now()
	svc, subscription := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(subscription, ""))
	events := svc.ApplySurfaceAction(control.Action{Kind: control.ActionTextMessage, SurfaceSessionID: "surface-1", MessageID: "race", Text: "remote task"})
	prompt := findAgentCommand(events, agentproto.CommandPromptSend)
	if prompt == nil {
		t.Fatal("remote prompt missing")
	}
	prompt.CommandID = "race-command"
	surface := svc.root.Surfaces["surface-1"]
	svc.bindPendingRemoteCommand(surface, prompt.CommandID)
	item := surface.QueueItems[surface.ActiveQueueItemID]
	local := agentproto.Initiator{Kind: agentproto.InitiatorLocalUI}
	svc.ApplyAgentEvent("inst-1", agentproto.Event{Kind: agentproto.EventTurnStarted, ThreadID: "thread-1", TurnID: "desktop-race", Initiator: local})
	if item.Status != state.QueueItemDispatching || svc.turns.pendingRemoteBinding("inst-1") == nil {
		t.Fatal("desktop notification consumed the remote pending command")
	}
	svc.ApplyAgentEvent("inst-1", agentproto.Event{Kind: agentproto.EventItemCompleted, ThreadID: "thread-1", TurnID: "desktop-race", ItemID: "desktop-output", ItemKind: "agent_message", Initiator: local, Metadata: map[string]any{"text": "desktop survived"}})
	svc.ApplyAgentEvent("inst-1", agentproto.Event{Kind: agentproto.EventTurnCompleted, ThreadID: "thread-1", CommandID: prompt.CommandID, Status: "failed", ErrorMessage: "thread busy", Initiator: agentproto.Initiator{Kind: agentproto.InitiatorRemoteSurface, SurfaceSessionID: surface.SurfaceSessionID}, TurnCompletionOrigin: agentproto.TurnCompletionOriginTurnStartRejected})
	if svc.root.Instances["inst-1"].ActiveTurnID != "desktop-race" || item.Status != state.QueueItemFailed {
		t.Fatalf("rejection must fail only remote input: active=%q item=%s", svc.root.Instances["inst-1"].ActiveTurnID, item.Status)
	}
	if got := svc.pendingTurnTextValue("inst-1", "thread-1", "desktop-race"); got != "desktop survived" {
		t.Fatalf("remote rejection erased desktop output: %q", got)
	}
	events = svc.ApplySurfaceAction(control.Action{Kind: control.ActionStop, SurfaceSessionID: surface.SurfaceSessionID})
	stop := findAgentCommand(events, agentproto.CommandTurnInterrupt)
	if stop == nil || stop.Target.TurnID != "desktop-race" {
		t.Fatalf("stop lost the surviving desktop turn: %#v", events)
	}
}

func TestSharedSubscriptionOpensExactDesktopOnlyAfterCorrelatedReady(t *testing.T) {
	now := time.Now()
	svc, command := sharedSubscriptionFixture(t, &now)
	inst := svc.root.Instances["inst-1"]
	inst.Threads["thread-1"].CWD = "/exact/project"
	inst.WorkspaceRoot = "/wrong/workspace"
	stale := sharedSubscriptionResult(command, "")
	stale.CommandID = "stale-command"
	for _, event := range svc.ApplyAgentEvent("inst-1", stale) {
		if event.DaemonCommand != nil && event.DaemonCommand.Kind == control.DaemonCommandOpenSharedDesktop {
			t.Fatal("stale response opened desktop")
		}
	}
	found := false
	for _, event := range svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(command, "")) {
		if c := event.DaemonCommand; c != nil && c.Kind == control.DaemonCommandOpenSharedDesktop {
			found = true
			if c.ThreadCWD != "/exact/project" || c.ThreadID != "thread-1" || c.InstanceID != "inst-1" || c.SurfaceSessionID != "surface-1" {
				t.Fatalf("incorrect desktop target: %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("ready subscription did not open desktop")
	}
}
