package orchestrator

import (
	"encoding/json"
	"github.com/YChange01/codex-feishu-link/internal/adapter/codex"
	"github.com/YChange01/codex-feishu-link/internal/adapter/feishu"
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
	"strings"
	"testing"
	"time"
)

func modelNotices(events []eventcontract.Event, code string) []eventcontract.Event {
	var out []eventcontract.Event
	for _, e := range events {
		if e.Notice != nil && e.Notice.Code == code {
			out = append(out, e)
		}
	}
	return out
}

func TestSharedNativeModelSettingsReachFeishuAndUpdateStaleSnapshot(t *testing.T) {
	now := time.Now()
	svc, cmd := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(cmd, ""))
	surface := svc.root.Surfaces["surface-1"]
	surface.Verbosity = state.SurfaceVerbosityQuiet
	thread := svc.root.Instances["inst-1"].Threads["thread-1"]
	thread.ExplicitModel = "old-model"
	thread.ExplicitReasoningEffort = "low"
	tr := codex.NewTranslator("inst-1")
	r, err := tr.ObserveServer([]byte(`{"method":"thread/settings/updated","params":{"threadId":"thread-1","threadSettings":{"model":"gpt-6-astra","effort":"ultra","modelProvider":"openai"}}}`))
	if err != nil || len(r.Events) != 1 {
		t.Fatalf("native settings: %v %#v", err, r)
	}
	events := svc.ApplyAgentEvent("inst-1", r.Events[0])
	if len(modelNotices(events, "shared_model_settings")) != 1 {
		t.Fatalf("missing changed settings notice: %#v", events)
	}
	if thread.ExplicitModel != "gpt-6-astra" || thread.ExplicitReasoningEffort != "ultra" {
		t.Fatalf("stale snapshot still wins: %#v", thread)
	}
	if len(modelNotices(svc.ApplyAgentEvent("inst-1", r.Events[0]), "shared_model_settings")) != 0 {
		t.Fatal("repeated settings spammed")
	}
	r, err = tr.ObserveServer([]byte(`{"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"turn-1"}}}`))
	if err != nil || len(r.Events) != 1 {
		t.Fatal(err)
	}
	event := r.Events[0]
	event.Initiator.Kind = agentproto.InitiatorLocalUI
	got := modelNotices(svc.ApplyAgentEvent("inst-1", event), "shared_turn_model")
	if len(got) != 1 {
		t.Fatalf("missing start model: %#v", got)
	}
	ops := feishu.NewProjector().ProjectEvent("chat-1", got[0])
	raw, _ := json.Marshal(ops)
	for _, want := range []string{"gpt-6-astra", "ultra", "本轮使用模型", "plain_text"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("card missing %q: %s", want, raw)
		}
	}
	if len(modelNotices(svc.ApplyAgentEvent("inst-1", event), "shared_turn_model")) != 0 {
		t.Fatal("duplicate start announced twice")
	}
}

func TestSharedModelNoticeUnknownAndRouting(t *testing.T) {
	now := time.Now()
	svc, cmd := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(cmd, ""))
	svc.root.Surfaces["surface-1"].PromptOverride = state.ModelConfigRecord{Model: "gpt-6-luna"}
	e := agentproto.Event{Kind: agentproto.EventTurnStarted, ThreadID: "thread-1", TurnID: "turn-1", Initiator: agentproto.Initiator{Kind: agentproto.InitiatorLocalUI}}
	got := modelNotices(svc.ApplyAgentEvent("inst-1", e), "shared_turn_model")
	if len(got) != 1 || got[0].Notice.Sections[0].Lines[0] != "未确认" {
		t.Fatalf("requested value mislabeled actual: %#v", got)
	}
	other := e
	other.ThreadID = "other-thread"
	if len(modelNotices(svc.ApplyAgentEvent("inst-1", other), "shared_turn_model")) != 0 {
		t.Fatal("wrong thread leak")
	}
	copyInst := *svc.root.Instances["inst-1"]
	copyInst.InstanceID = "other-proxy"
	svc.UpsertInstance(&copyInst)
	svc.sharedSubscriptions["other-proxy"] = &sharedThreadSubscription{ThreadID: "thread-1", Ready: true}
	if len(modelNotices(svc.ApplyAgentEvent("other-proxy", e), "shared_turn_model")) != 0 {
		t.Fatal("wrong instance leak")
	}
}

func TestSharedModelRerouteNotifiesOnceAndIgnoresOldTurn(t *testing.T) {
	now := time.Now()
	svc, cmd := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(cmd, "turn-1"))
	e := agentproto.Event{Kind: agentproto.EventTurnModelRerouted, ThreadID: "thread-1", TurnID: "turn-1", ModelReroute: &agentproto.TurnModelReroute{ThreadID: "thread-1", TurnID: "turn-1", FromModel: "gpt-6-astra", ToModel: "gpt-6-luna", Reason: "backend_reason"}}
	got := modelNotices(svc.ApplyAgentEvent("inst-1", e), "shared_model_rerouted")
	if len(got) != 1 {
		t.Fatalf("missing reroute: %#v", got)
	}
	raw, _ := json.Marshal(got[0])
	for _, want := range []string{"gpt-6-astra", "gpt-6-luna", "backend_reason"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("missing %s", want)
		}
	}
	if len(modelNotices(svc.ApplyAgentEvent("inst-1", e), "shared_model_rerouted")) != 0 {
		t.Fatal("reroute duplicate")
	}
	e.TurnID = "old-turn"
	e.ModelReroute.TurnID = "old-turn"
	e.ModelReroute.ToModel = "old-model"
	if len(modelNotices(svc.ApplyAgentEvent("inst-1", e), "shared_model_rerouted")) != 0 || svc.root.Instances["inst-1"].Threads["thread-1"].ExplicitModel != "gpt-6-luna" {
		t.Fatal("late reroute replaced current model")
	}
}

func TestSharedClearModelDispatchDoesNotFreezeDesktopModelOrPlan(t *testing.T) {
	now := time.Now()
	svc, cmd := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(cmd, ""))
	surface := svc.root.Surfaces["surface-1"]
	surface.PromptOverride = state.ModelConfigRecord{Model: "gpt-6-luna", ReasoningEffort: "low"}
	svc.ApplySurfaceAction(control.Action{Kind: control.ActionModelCommand, SurfaceSessionID: "surface-1", Text: "/model clear"})
	thread := svc.root.Instances["inst-1"].Threads["thread-1"]
	thread.ExplicitModel = "gpt-6-astra"
	thread.ExplicitReasoningEffort = "ultra"
	events := svc.ApplySurfaceAction(control.Action{Kind: control.ActionTextMessage, SurfaceSessionID: "surface-1", MessageID: "input", Text: "continue"})
	send := findAgentCommand(events, agentproto.CommandPromptSend)
	if send == nil {
		t.Fatalf("missing dispatch: %#v", events)
	}
	if send.Overrides.Model != "" || send.Overrides.ReasoningEffort != "" || send.Overrides.PlanMode != "" {
		t.Fatalf("following desktop froze settings: %#v", send.Overrides)
	}
}

func TestSharedModelEffortClearAndRequestedMismatchAreVisible(t *testing.T) {
	now := time.Now()
	svc, cmd := sharedSubscriptionFixture(t, &now)
	subscribed := sharedSubscriptionResult(cmd, "")
	subscribed.Model = "gpt-6-astra"
	subscribed.ReasoningEffort = "ultra"
	svc.ApplyAgentEvent("inst-1", subscribed)
	update := agentproto.Event{Kind: agentproto.EventThreadSettingsUpdated, ThreadID: "thread-1", ThreadSettings: &agentproto.ThreadSettingsUpdate{ThreadID: "thread-1", Model: "gpt-6-astra"}, Metadata: map[string]any{"reasoningEffortCleared": true}}
	got := modelNotices(svc.ApplyAgentEvent("inst-1", update), "shared_model_settings")
	if len(got) != 1 || got[0].Notice.Sections[1].Lines[0] != "未确认" || svc.root.Instances["inst-1"].Threads["thread-1"].ExplicitReasoningEffort != "" {
		t.Fatalf("clear retained stale effort: %#v", got)
	}
	surface := svc.root.Surfaces["surface-1"]
	surface.PromptOverride = state.ModelConfigRecord{Model: "gpt-6-astra", ReasoningEffort: "ultra"}
	events := svc.ApplySurfaceAction(control.Action{Kind: control.ActionTextMessage, SurfaceSessionID: "surface-1", MessageID: "remote", Text: "continue"})
	if findAgentCommand(events, agentproto.CommandPromptSend) == nil {
		t.Fatal("missing remote dispatch")
	}
	event := agentproto.Event{Kind: agentproto.EventTurnStarted, ThreadID: "thread-1", TurnID: "remote-turn", Model: "gpt-6-luna", ReasoningEffort: "high", Initiator: agentproto.Initiator{Kind: agentproto.InitiatorRemoteSurface, SurfaceSessionID: "surface-1"}}
	got = modelNotices(svc.ApplyAgentEvent("inst-1", event), "shared_turn_model")
	if len(got) != 1 {
		t.Fatalf("missing remote model notice: %#v", got)
	}
	raw, _ := json.Marshal(got[0])
	if !strings.Contains(string(raw), "请求与后台确认不一致") {
		t.Fatalf("silent mismatch: %s", raw)
	}
}
