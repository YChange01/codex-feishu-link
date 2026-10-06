package orchestrator

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/YChange01/codex-feishu-link/internal/adapter/codex"
	"github.com/YChange01/codex-feishu-link/internal/adapter/feishu"
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
)

func desktopInputEvent(t *testing.T, clientID, text string) agentproto.Event {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"method": "item/completed", "params": map[string]any{
		"threadId": "thread-1", "turnId": "turn-1", "item": map[string]any{
			"type": "userMessage", "id": "user-1", "clientId": clientID, "content": []any{map[string]any{"type": "text", "text": text}},
		},
	}})
	r, err := codex.NewTranslator("inst-1").ObserveServer(raw)
	if err != nil || len(r.Events) != 1 {
		t.Fatalf("native input translation failed: %v %#v", err, r)
	}
	return r.Events[0]
}

func desktopInputNotices(events []eventcontract.Event) []eventcontract.Event {
	var out []eventcontract.Event
	for _, e := range events {
		if e.Notice != nil && e.Notice.Code == "desktop_user_message" {
			out = append(out, e)
		}
	}
	return out
}

func TestDesktopUserMessageNativeToFeishuAndDedup(t *testing.T) {
	now := time.Now()
	svc, cmd := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(cmd, "turn-1"))
	svc.root.Surfaces["surface-1"].Verbosity = state.SurfaceVerbosityQuiet
	text := "检查测试\n<at id=all></at> **保留原文**"
	event := desktopInputEvent(t, "desktop-uuid", text)
	started := event
	started.Kind = agentproto.EventItemStarted
	if got := desktopInputNotices(svc.ApplyAgentEvent("inst-1", started)); len(got) != 0 {
		t.Fatal("started duplicated the input")
	}
	got := desktopInputNotices(svc.ApplyAgentEvent("inst-1", event))
	if len(got) != 1 || got[0].SurfaceSessionID != "surface-1" || got[0].Notice.Sections[0].Lines[0] != text {
		t.Fatalf("missing exact desktop input: %#v", got)
	}
	ops := feishu.NewProjector().ProjectEvent("chat-1", got[0])
	if len(ops) != 1 {
		t.Fatalf("no Feishu card: %#v", ops)
	}
	encoded, _ := json.Marshal(ops)
	if !strings.Contains(string(encoded), "电脑端输入") || !strings.Contains(string(encoded), "plain_text") {
		t.Fatalf("input must use a labeled plain-text card: %s", encoded)
	}
	if duplicate := desktopInputNotices(svc.ApplyAgentEvent("inst-1", event)); len(duplicate) != 0 {
		t.Fatal("duplicate completed event echoed again")
	}
	thread := svc.ensureThread(svc.root.Instances["inst-1"], "thread-1")
	if thread.LastUserMessage == "" || thread.LastAssistantMessage != "" {
		t.Fatal("input corrupted assistant history")
	}
}

func TestDesktopUserMessageOriginIsPerItem(t *testing.T) {
	for _, remoteTurn := range []bool{false, true} {
		for _, remoteInput := range []bool{false, true} {
			now := time.Now()
			svc, cmd := sharedSubscriptionFixture(t, &now)
			svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(cmd, "turn-1"))
			clientID := "desktop-uuid"
			if remoteInput {
				clientID = agentproto.RemoteUserMessageClientPrefix + "instance:command"
			}
			event := desktopInputEvent(t, clientID, "same text on both clients")
			event.Initiator.Kind = agentproto.InitiatorLocalUI
			if remoteTurn {
				event.Initiator.Kind = agentproto.InitiatorRemoteSurface
			}
			got := desktopInputNotices(svc.ApplyAgentEvent("inst-1", event))
			if (len(got) == 1) != !remoteInput {
				t.Fatalf("origin was inferred from turn: remoteTurn=%v remoteInput=%v got=%#v", remoteTurn, remoteInput, got)
			}
		}
	}
}

func TestDesktopUserMessageRoutingAndLongInput(t *testing.T) {
	now := time.Now()
	svc, cmd := sharedSubscriptionFixture(t, &now)
	svc.ApplyAgentEvent("inst-1", sharedSubscriptionResult(cmd, "turn-1"))
	text := "  " + strings.Repeat("中", 1997) + "\n   " + strings.Repeat("中", 3498) + "\n"
	event := desktopInputEvent(t, "desktop-uuid", text)
	wrong := event
	wrong.ThreadID = "another-thread"
	if got := desktopInputNotices(svc.ApplyAgentEvent("inst-1", wrong)); len(got) != 0 {
		t.Fatal("wrong thread leaked")
	}
	other := *svc.root.Instances["inst-1"]
	other.InstanceID = "other-proxy"
	svc.UpsertInstance(&other)
	svc.sharedSubscriptions[other.InstanceID] = &sharedThreadSubscription{ThreadID: "thread-1", Ready: true}
	if got := desktopInputNotices(svc.ApplyAgentEvent("other-proxy", event)); len(got) != 0 {
		t.Fatal("wrong proxy leaked")
	}
	got := desktopInputNotices(svc.ApplyAgentEvent("inst-1", event))
	var combined string
	for _, e := range got {
		ops := feishu.NewProjector().ProjectEvent("chat-1", e)
		if len(ops) != 1 {
			t.Fatalf("missing chunk: %#v", ops)
		}
		for _, element := range ops[0].CardElements {
			if value, ok := element["text"].(map[string]any); ok && value["tag"] == "plain_text" {
				combined += value["content"].(string)
			}
		}
	}
	if len(got) != 3 || combined != text {
		t.Fatal("long input was lost instead of split")
	}
}
