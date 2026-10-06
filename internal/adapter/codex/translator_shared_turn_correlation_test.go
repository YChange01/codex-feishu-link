package codex

import (
	"fmt"
	"strings"
	"testing"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
)

func TestSharedTurnCorrelationRejectedRemoteStartPreservesDesktopTurn(t *testing.T) {
	tr := NewTranslator("inst-shared")
	tr.SetSharedAppServerMode(true)
	sharedSubscriptionReady(t, tr, "sub1", "thread1", `[]`)
	request := sharedTurnCorrelationSendPrompt(t, tr)

	desktop := sharedSubscriptionObserve(t, tr, `{"method":"turn/started","params":{"threadId":"thread1","turn":{"id":"desktop-turn"}}}`)
	if len(desktop.Events) != 0 {
		t.Fatal("uncorrelated start escaped before the RPC response")
	}
	rejected := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"error":{"code":-32000,"message":"another turn is already active"}}`, request.ID))
	if len(rejected.Events) != 2 {
		t.Fatalf("must release desktop start before rejecting remote command: %#v", rejected)
	}
	sharedTurnCorrelationAssertEvent(t, Result{Events: rejected.Events[:1]}, agentproto.EventTurnStarted, "desktop-turn", agentproto.InitiatorLocalUI, "")
	event := sharedTurnCorrelationAssertEvent(t, Result{Events: rejected.Events[1:]}, agentproto.EventTurnCompleted, "", agentproto.InitiatorRemoteSurface, "surface1")
	if event.CommandID != "remote1" || event.Status != "failed" || event.TurnCompletionOrigin != agentproto.TurnCompletionOriginTurnStartRejected || !strings.Contains(event.ErrorMessage, "another turn is already active") {
		t.Fatalf("rejection must belong only to the remote command: %#v", event)
	}
	for _, tt := range []struct {
		raw  string
		kind agentproto.EventKind
	}{
		{raw: `{"method":"item/agentMessage/delta","params":{"threadId":"thread1","turnId":"desktop-turn","itemId":"item1","delta":"desktop output"}}`, kind: agentproto.EventItemDelta},
		{raw: `{"method":"turn/completed","params":{"threadId":"thread1","turn":{"id":"desktop-turn","status":"completed"}}}`, kind: agentproto.EventTurnCompleted},
	} {
		observed := sharedSubscriptionObserve(t, tr, tt.raw)
		sharedTurnCorrelationAssertEvent(t, observed, tt.kind, "desktop-turn", agentproto.InitiatorLocalUI, "")
	}
}

func TestSharedTurnCorrelationCompletedNotificationsBeforeRPCStayOrdered(t *testing.T) {
	tr := NewTranslator("inst-shared")
	tr.SetSharedAppServerMode(true)
	sharedSubscriptionReady(t, tr, "sub1", "thread1", `[]`)
	request := sharedTurnCorrelationSendPrompt(t, tr)
	for _, raw := range []string{
		`{"method":"turn/started","params":{"threadId":"thread1","turn":{"id":"remote-turn"}}}`,
		`{"method":"item/agentMessage/delta","params":{"threadId":"thread1","turnId":"remote-turn","itemId":"item1","delta":"early output"}}`,
		`{"method":"turn/completed","params":{"threadId":"thread1","turn":{"id":"remote-turn","status":"completed"}}}`,
	} {
		if result := sharedSubscriptionObserve(t, tr, raw); len(result.Events) != 0 {
			t.Fatalf("unconfirmed notification escaped: %#v", result)
		}
	}
	result := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"turn":{"id":"remote-turn","status":"inProgress"}}}`, request.ID))
	if len(result.Events) != 3 {
		t.Fatalf("must release exact original lifecycle without duplicate start: %#v", result)
	}
	for i, kind := range []agentproto.EventKind{agentproto.EventTurnStarted, agentproto.EventItemDelta, agentproto.EventTurnCompleted} {
		event := sharedTurnCorrelationAssertEvent(t, Result{Events: result.Events[i : i+1]}, kind, "remote-turn", agentproto.InitiatorRemoteSurface, "surface1")
		if event.CommandID != "remote1" {
			t.Fatal("buffered event lost its command correlation")
		}
	}
}

func TestSharedTurnCorrelationEarlierDesktopLifecyclePrecedesRemoteRPCStart(t *testing.T) {
	tr := NewTranslator("inst-shared")
	tr.SetSharedAppServerMode(true)
	sharedSubscriptionReady(t, tr, "sub1", "thread1", `[]`)
	request := sharedTurnCorrelationSendPrompt(t, tr)
	for _, raw := range []string{
		`{"method":"turn/started","params":{"threadId":"thread1","turn":{"id":"desktop-turn"}}}`,
		`{"method":"turn/completed","params":{"threadId":"thread1","turn":{"id":"desktop-turn","status":"completed"}}}`,
	} {
		sharedSubscriptionObserve(t, tr, raw)
	}
	result := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"turn":{"id":"remote-turn","status":"inProgress"}}}`, request.ID))
	if len(result.Events) != 3 {
		t.Fatalf("missing ordered desktop and remote evidence: %#v", result)
	}
	sharedTurnCorrelationAssertEvent(t, Result{Events: result.Events[:1]}, agentproto.EventTurnStarted, "desktop-turn", agentproto.InitiatorLocalUI, "")
	sharedTurnCorrelationAssertEvent(t, Result{Events: result.Events[1:2]}, agentproto.EventTurnCompleted, "desktop-turn", agentproto.InitiatorLocalUI, "")
	sharedTurnCorrelationAssertEvent(t, Result{Events: result.Events[2:]}, agentproto.EventTurnStarted, "remote-turn", agentproto.InitiatorRemoteSurface, "surface1")
}

func TestSharedTurnCorrelationNewThreadBuffersEarlyOutput(t *testing.T) {
	tr := NewTranslator("inst-shared")
	tr.SetSharedAppServerMode(true)
	frames, err := tr.TranslateCommand(agentproto.Command{
		Kind: agentproto.CommandPromptSend, CommandID: "remote1", Origin: agentproto.Origin{Surface: "surface1"},
		Target: agentproto.Target{CWD: "/workspace"}, Prompt: agentproto.Prompt{Inputs: []agentproto.Input{{Type: agentproto.InputText, Text: "new task"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	create := sharedSubscriptionOnlyFrame(t, frames, "thread/start")
	created := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"thread":{"id":"thread1","cwd":"/workspace"}}}`, create.ID))
	start := sharedSubscriptionOnlyFrame(t, created.OutboundToCodex, "turn/start")
	for _, raw := range []string{
		`{"method":"turn/started","params":{"threadId":"thread1","turn":{"id":"remote-turn"}}}`,
		`{"method":"item/agentMessage/delta","params":{"threadId":"thread1","turnId":"remote-turn","itemId":"item1","delta":"first output"}}`,
	} {
		if result := sharedSubscriptionObserve(t, tr, raw); len(result.Events) != 0 {
			t.Fatalf("new thread event escaped before exact owner was known: %#v", result)
		}
	}
	result := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"turn":{"id":"remote-turn","status":"inProgress"}}}`, start.ID))
	if len(result.Events) != 2 {
		t.Fatalf("new thread early output was dropped: %#v", result)
	}
	sharedTurnCorrelationAssertEvent(t, Result{Events: result.Events[:1]}, agentproto.EventTurnStarted, "remote-turn", agentproto.InitiatorRemoteSurface, "surface1")
	sharedTurnCorrelationAssertEvent(t, Result{Events: result.Events[1:]}, agentproto.EventItemDelta, "remote-turn", agentproto.InitiatorRemoteSurface, "surface1")
}

func TestSharedTurnCorrelationRequestResolvedWithoutTurnIDStaysOrdered(t *testing.T) {
	tr := NewTranslator("inst-shared")
	tr.SetSharedAppServerMode(true)
	sharedSubscriptionReady(t, tr, "sub1", "thread1", `[]`)
	request := sharedTurnCorrelationSendPrompt(t, tr)
	for _, raw := range []string{
		`{"method":"turn/started","params":{"threadId":"thread1","turn":{"id":"remote-turn"}}}`,
		`{"method":"serverRequest/started","params":{"threadId":"thread1","turnId":"remote-turn","request":{"id":"approval1","type":"approval","title":"Run command?","command":"git status"}}}`,
		`{"method":"serverRequest/resolved","params":{"threadId":"thread1","requestId":"approval1"}}`,
	} {
		if result := sharedSubscriptionObserve(t, tr, raw); len(result.Events) != 0 {
			t.Fatalf("request lifecycle escaped before RPC correlation: %#v", result)
		}
	}
	result := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"turn":{"id":"remote-turn","status":"inProgress"}}}`, request.ID))
	if len(result.Events) != 3 || result.Events[1].Kind != agentproto.EventRequestStarted || result.Events[2].Kind != agentproto.EventRequestResolved || result.Events[2].TurnID != "" || result.Events[2].RequestID != "approval1" {
		t.Fatalf("resolved request must follow its creation even without turnId: %#v", result)
	}
}

func TestSharedTurnCorrelationUnsupportedCompactAndReviewFailBeforeSending(t *testing.T) {
	for _, kind := range []agentproto.CommandKind{agentproto.CommandThreadCompactStart, agentproto.CommandReviewStart} {
		t.Run(string(kind), func(t *testing.T) {
			tr := NewTranslator("inst-shared")
			tr.SetSharedAppServerMode(true)
			sharedSubscriptionReady(t, tr, "sub1", "thread1", `[]`)
			frames, err := tr.TranslateCommand(agentproto.Command{Kind: kind, Target: agentproto.Target{ThreadID: "thread1"}})
			if err == nil || len(frames) != 0 || len(tr.pendingRemoteTurnByThread) != 0 {
				t.Fatalf("unsupported operation must fail without starting a pending turn: frames=%q err=%v", frames, err)
			}
		})
	}
}

func TestSharedTurnCorrelationSuccessfulRPCBindsOnlyReturnedTurnID(t *testing.T) {
	tr := NewTranslator("inst-shared")
	tr.SetSharedAppServerMode(true)
	sharedSubscriptionReady(t, tr, "sub1", "thread1", `[]`)
	request := sharedTurnCorrelationSendPrompt(t, tr)
	accepted := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"turn":{"id":"remote-turn","status":"inProgress"}}}`, request.ID))
	event := sharedTurnCorrelationAssertEvent(t, accepted, agentproto.EventTurnStarted, "remote-turn", agentproto.InitiatorRemoteSurface, "surface1")
	if event.CommandID != "remote1" {
		t.Fatalf("successful start lost command correlation: %#v", event)
	}

	for _, tt := range []struct {
		turnID    string
		initiator agentproto.InitiatorKind
		surfaceID string
	}{
		{turnID: "remote-turn", initiator: agentproto.InitiatorRemoteSurface, surfaceID: "surface1"},
		{turnID: "desktop-turn", initiator: agentproto.InitiatorLocalUI},
	} {
		started := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"method":"turn/started","params":{"threadId":"thread1","turn":{"id":%q}}}`, tt.turnID))
		sharedTurnCorrelationAssertEvent(t, started, agentproto.EventTurnStarted, tt.turnID, tt.initiator, tt.surfaceID)
		delta := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"method":"item/agentMessage/delta","params":{"threadId":"thread1","turnId":%q,"itemId":"item1","delta":"output"}}`, tt.turnID))
		sharedTurnCorrelationAssertEvent(t, delta, agentproto.EventItemDelta, tt.turnID, tt.initiator, tt.surfaceID)
	}
}

func sharedTurnCorrelationSendPrompt(t *testing.T, tr *Translator) sharedSubscriptionTestFrame {
	t.Helper()
	frames, err := tr.TranslateCommand(agentproto.Command{
		Kind: agentproto.CommandPromptSend, CommandID: "remote1",
		Origin: agentproto.Origin{Surface: "surface1"},
		Target: agentproto.Target{ThreadID: "thread1"},
		Prompt: agentproto.Prompt{Inputs: []agentproto.Input{{Type: agentproto.InputText, Text: "remote task"}}},
	})
	if err != nil {
		t.Fatalf("send remote prompt: %v", err)
	}
	return sharedSubscriptionOnlyFrame(t, frames, "turn/start")
}

func sharedTurnCorrelationAssertEvent(t *testing.T, result Result, kind agentproto.EventKind, turnID string, initiator agentproto.InitiatorKind, surfaceID string) agentproto.Event {
	t.Helper()
	if len(result.Events) != 1 || len(result.OutboundToCodex) != 0 {
		t.Fatalf("expected one correlated event without followup, got %#v", result)
	}
	event := result.Events[0]
	if event.Kind != kind || event.ThreadID != "thread1" || event.TurnID != turnID || event.Initiator.Kind != initiator || event.Initiator.SurfaceSessionID != surfaceID {
		t.Fatalf("incorrect turn correlation: got %#v; want kind=%s turn=%s initiator=%s surface=%s", event, kind, turnID, initiator, surfaceID)
	}
	return event
}
