package codex

import (
	"encoding/json"
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"testing"
)

func TestDesktopThreadSettingsNativeShapeSuppliesObservedTurnModel(t *testing.T) {
	tr := NewTranslator("inst-1")
	// Shape captured from the shared desktop app server; effort is not reasoningEffort.
	r, err := tr.ObserveServer([]byte(`{"method":"thread/settings/updated","params":{"threadId":"thread-1","threadSettings":{"model":"gpt-6-astra","effort":"ultra","modelProvider":"openai"}}}`))
	if err != nil || len(r.Events) != 1 {
		t.Fatalf("settings: %v %#v", err, r)
	}
	got := r.Events[0].ThreadSettings
	if got == nil || got.Model != "gpt-6-astra" || got.ReasoningEffort != "ultra" {
		t.Fatalf("lost live settings: %#v", got)
	}
	r, err = tr.ObserveServer([]byte(`{"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"turn-1"}}}`))
	if err != nil || len(r.Events) != 1 {
		t.Fatalf("turn: %v %#v", err, r)
	}
	if r.Events[0].Model != "gpt-6-astra" || r.Events[0].ReasoningEffort != "ultra" {
		t.Fatalf("observed settings missing without resume policy: %#v", r.Events[0])
	}
	r, _ = tr.ObserveServer([]byte(`{"method":"turn/started","params":{"threadId":"other-thread","turn":{"id":"other-turn"}}}`))
	if r.Events[0].Model != "" || r.Events[0].ReasoningEffort != "" {
		t.Fatal("another thread inherited model evidence")
	}
}

func TestSharedTurnFollowsLiveSettingsInsteadOfCachedPolicyAndTemplate(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		tr := NewTranslator("inst-1")
		tr.SetSharedAppServerMode(true)
		sharedSubscriptionReady(t, tr, "sub", "thread-1", `[]`)
		tr.turnStartByThread["thread-1"] = map[string]any{"model": "old-model", "effort": "low", "collaborationMode": map[string]any{"mode": "default", "settings": map[string]any{"model": "old-model", "reasoning_effort": "low"}}}
		command := agentproto.Command{Kind: agentproto.CommandPromptSend, CommandID: "send", Target: agentproto.Target{ThreadID: "thread-1"}, CodexResume: &agentproto.CodexResumePolicy{Mode: agentproto.CodexResumePreserveThreadSettings, ModelMode: agentproto.CodexThreadValuePreservedObserved, Model: "old-model", ReasoningMode: agentproto.CodexThreadValuePreservedObserved, ReasoningEffort: "low"}}
		if explicit {
			command.Overrides = agentproto.PromptOverrides{Model: "gpt-6-astra", ReasoningEffort: "ultra"}
		}
		frames, err := tr.TranslateCommand(command)
		if err != nil || len(frames) != 1 {
			t.Fatalf("translate: %v %#v", err, frames)
		}
		var frame struct {
			Method string
			Params map[string]any
		}
		if err := json.Unmarshal(frames[0], &frame); err != nil {
			t.Fatal(err)
		}
		if frame.Method != "turn/start" {
			t.Fatalf("unexpected method %s", frame.Method)
		}
		if explicit {
			if frame.Params["model"] != "gpt-6-astra" || frame.Params["effort"] != "ultra" {
				t.Fatalf("explicit choice lost: %#v", frame.Params)
			}
		} else {
			for _, key := range []string{"model", "effort", "collaborationMode"} {
				if frame.Params[key] != nil {
					t.Fatalf("replayed stale %s: %#v", key, frame.Params)
				}
			}
		}
	}
}

func TestDesktopExplicitNullEffortDoesNotClaimPreviousUltra(t *testing.T) {
	tr := NewTranslator("inst-1")
	tr.ObserveServer([]byte(`{"method":"thread/settings/updated","params":{"threadId":"t","threadSettings":{"model":"gpt-6-astra","effort":"ultra"}}}`))
	settings, _ := tr.ObserveServer([]byte(`{"method":"thread/settings/updated","params":{"threadId":"t","threadSettings":{"model":"gpt-6-astra","effort":null}}}`))
	if len(settings.Events) != 1 || settings.Events[0].Metadata["reasoningEffortCleared"] != true {
		t.Fatalf("missing clear evidence: %#v", settings)
	}
	turn, _ := tr.ObserveServer([]byte(`{"method":"turn/started","params":{"threadId":"t","turn":{"id":"turn"}}}`))
	if turn.Events[0].ReasoningEffort != "" {
		t.Fatal("null effort retained a stale ultra claim")
	}
}

func TestSharedSubscriptionCarriesResumeResponseModel(t *testing.T) {
	tr := NewTranslator("inst-1")
	tr.SetSharedAppServerMode(true)
	resume := sharedSubscriptionBegin(t, tr, "subscribe", "thread-1")
	raw, _ := json.Marshal(map[string]any{"id": resume.ID, "result": map[string]any{"thread": map[string]any{"id": "thread-1", "cwd": "/repo"}, "model": "gpt-6-astra", "reasoningEffort": "ultra"}})
	r, err := tr.ObserveServer(raw)
	if err != nil {
		t.Fatal(err)
	}
	list := sharedSubscriptionOnlyFrame(t, r.OutboundToCodex, "thread/turns/list")
	raw, _ = json.Marshal(map[string]any{"id": list.ID, "result": map[string]any{"data": []any{}}})
	r, err = tr.ObserveServer(raw)
	if err != nil {
		t.Fatal(err)
	}
	e := sharedSubscriptionEvent(t, r, "subscribe", "thread-1", "ready")
	if e.Model != "gpt-6-astra" || e.ReasoningEffort != "ultra" {
		t.Fatalf("lost resume evidence: %#v", e)
	}
}
