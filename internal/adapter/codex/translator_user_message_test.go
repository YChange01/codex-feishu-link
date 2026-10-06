package codex

import (
	"strings"
	"testing"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
)

func TestSharedUserMessageClientIDsMarkBothStartAndSteer(t *testing.T) {
	tr := NewTranslator("inst-shared")
	tr.SetSharedAppServerMode(true)
	sharedSubscriptionReady(t, tr, "subscribe", "thread1", `[]`)
	start := sharedTurnCorrelationSendPrompt(t, tr)
	startID := lookupString(start.Params, "clientUserMessageId")
	if !strings.HasPrefix(startID, agentproto.RemoteUserMessageClientPrefix) {
		t.Fatalf("start lacks an echoed origin marker: %q", startID)
	}
	frames, err := tr.TranslateCommand(agentproto.Command{Kind: agentproto.CommandTurnSteer, CommandID: "steer1", Target: agentproto.Target{ThreadID: "thread1", TurnID: "turn1"}, Prompt: agentproto.Prompt{Inputs: []agentproto.Input{{Type: agentproto.InputText, Text: "phone steer"}}}})
	if err != nil {
		t.Fatal(err)
	}
	steer := sharedSubscriptionOnlyFrame(t, frames, "turn/steer")
	steerID := lookupString(steer.Params, "clientUserMessageId")
	if !strings.HasPrefix(steerID, agentproto.RemoteUserMessageClientPrefix) || steerID == startID {
		t.Fatalf("steer needs its own origin marker: %q", steerID)
	}
}

func TestUserMessagePreservesClientIDAndText(t *testing.T) {
	tr := NewTranslator("inst")
	result, err := tr.ObserveServer([]byte(`{"method":"item/completed","params":{"threadId":"thread1","turnId":"turn1","item":{"type":"userMessage","id":"item1","clientId":"desktop-uuid","content":[{"type":"text","text":"first\nsecond"}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 1 || result.Events[0].ItemKind != "user_message" || result.Events[0].Metadata["clientId"] != "desktop-uuid" || result.Events[0].Metadata["text"] != "first\nsecond" {
		t.Fatalf("input content / origin lost: %#v", result)
	}
	metadata := extractItemMetadata("user_message", map[string]any{"content": []any{map[string]any{"type": "image", "url": "data:private"}}})
	if text, _ := metadata["text"].(string); text == "" || strings.Contains(text, "private") {
		t.Fatalf("missing safe attachment placeholder: %#v", metadata)
	}
}
