package codex

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
)

func TestSharedSubscriptionRejectsChildRestartRestore(t *testing.T) {
	tr := NewTranslator("inst-shared")
	tr.SetSharedAppServerMode(true)
	tr.currentThreadID = "desktop-thread"
	tr.knownThreadCWD["desktop-thread"] = "/workspace"
	frame, id, ok, err := tr.BuildChildRestartRestoreFrame("restart-shared")
	if err == nil || ok || len(frame) != 0 || id != "" || len(tr.pendingChildRestartRestore) != 0 {
		t.Fatalf("shared restart must fail before a configured resume: frame=%s id=%q ok=%t err=%v", frame, id, ok, err)
	}
}

func TestSharedSubscriptionObservesActiveTurnWithoutStartingOne(t *testing.T) {
	tr := NewTranslator("inst-1")
	tr.SetSharedAppServerMode(true)
	frames, err := tr.TranslateCommand(agentproto.Command{
		Kind:      agentproto.CommandThreadSubscribe,
		CommandID: "sub1",
		Target:    agentproto.Target{ThreadID: "thread-1", CWD: "/other-repo"},
		Overrides: agentproto.PromptOverrides{Model: "other-model", ReasoningEffort: "high"},
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	resume := sharedSubscriptionOnlyFrame(t, frames, "thread/resume")
	if want := (map[string]any{"threadId": "thread-1", "excludeTurns": true}); !reflect.DeepEqual(resume.Params, want) {
		t.Fatalf("subscription must preserve the desktop thread configuration: got %#v, want %#v", resume.Params, want)
	}
	resumed := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"thread":{"id":"thread-1","cwd":"/repo","status":{"type":"active"}}}}`, resume.ID))
	if !resumed.Suppress || len(resumed.Events) != 0 {
		t.Fatalf("resume must wait for the active-turn snapshot: %#v", resumed)
	}
	list := sharedSubscriptionOnlyFrame(t, resumed.OutboundToCodex, "thread/turns/list")
	wantList := map[string]any{"threadId": "thread-1", "limit": float64(2), "sortDirection": "desc", "itemsView": "notLoaded"}
	if !reflect.DeepEqual(list.Params, wantList) {
		t.Fatalf("unexpected lightweight turn lookup: got %#v, want %#v", list.Params, wantList)
	}
	result := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"data":[{"id":"turn-1","status":"inProgress"},{"id":"turn-0","status":"completed"}]}}`, list.ID))
	event := sharedSubscriptionEvent(t, result, "sub1", "thread-1", "ready")
	if event.TurnID != "turn-1" {
		t.Fatalf("expected existing desktop turn, got %#v", event)
	}
}

func TestSharedSubscriptionScopesNotificationsAndPreservesStandaloneInitiators(t *testing.T) {
	for _, shared := range []bool{true, false} {
		t.Run(fmt.Sprintf("shared=%t", shared), func(t *testing.T) {
			tr := NewTranslator("inst-1")
			tr.SetSharedAppServerMode(shared)
			if shared {
				sharedSubscriptionReady(t, tr, "sub1", "thread-1", `[]`)
			}
			result := sharedSubscriptionObserve(t, tr, `{"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"desktop-turn"}}}`)
			wantInitiator := agentproto.InitiatorUnknown
			if shared {
				wantInitiator = agentproto.InitiatorLocalUI
			}
			if len(result.Events) != 1 || result.Events[0].Kind != agentproto.EventTurnStarted || result.Events[0].Initiator.Kind != wantInitiator {
				t.Fatalf("unexpected desktop turn ownership: %#v", result.Events)
			}
			for _, raw := range []string{
				`{"method":"thread/started","params":{"thread":{"id":"other-thread","cwd":"/other"}}}`,
				`{"method":"turn/started","params":{"threadId":"other-thread","turn":{"id":"other-turn"}}}`,
				`{"method":"item/agentMessage/delta","params":{"threadId":"other-thread","turnId":"other-turn","itemId":"item-1","delta":"private output"}}`,
				`{"method":"turn/completed","params":{"threadId":"other-thread","turn":{"id":"other-turn","status":"completed"}}}`,
			} {
				observed := sharedSubscriptionObserve(t, tr, raw)
				if shared && len(observed.Events) != 0 {
					t.Fatalf("unselected thread leaked into shared connection: %#v", observed.Events)
				}
				if !shared && len(observed.Events) == 0 {
					t.Fatalf("standalone translator unexpectedly filtered notification: %s", raw)
				}
			}
			if shared {
				sharedSubscriptionAssertDirectPrompt(t, tr, "thread-1")
			}
		})
	}
}

func TestSharedSubscriptionSwitchIgnoresLateResponsesAndOldThreadEvents(t *testing.T) {
	for _, pendingStage := range []string{"resume", "turns-list"} {
		t.Run(pendingStage, func(t *testing.T) {
			tr := NewTranslator("inst-1")
			tr.SetSharedAppServerMode(true)
			oldResume := sharedSubscriptionBegin(t, tr, "sub-old", "thread-old")
			lateResponse := fmt.Sprintf(`{"id":%q,"result":{"thread":{"id":"thread-old","cwd":"/old"}}}`, oldResume.ID)
			if pendingStage == "turns-list" {
				result := sharedSubscriptionObserve(t, tr, lateResponse)
				oldList := sharedSubscriptionOnlyFrame(t, result.OutboundToCodex, "thread/turns/list")
				lateResponse = fmt.Sprintf(`{"id":%q,"result":{"data":[{"id":"old-turn","status":"inProgress"}]}}`, oldList.ID)
			}
			frames, err := tr.TranslateCommand(agentproto.Command{Kind: agentproto.CommandThreadSubscribe, CommandID: "sub-new", Target: agentproto.Target{ThreadID: "thread-new"}})
			if err != nil {
				t.Fatalf("switch subscription: %v", err)
			}
			if len(frames) != 2 {
				t.Fatalf("expected unsubscribe then resume, got %q", frames)
			}
			unsubscribe := sharedSubscriptionOnlyFrame(t, frames[:1], "thread/unsubscribe")
			if !reflect.DeepEqual(unsubscribe.Params, map[string]any{"threadId": "thread-old"}) {
				t.Fatalf("must unsubscribe previous thread: %#v", unsubscribe.Params)
			}
			resume := sharedSubscriptionOnlyFrame(t, frames[1:], "thread/resume")
			if resume.Params["threadId"] != "thread-new" {
				t.Fatalf("wrong new subscription: %#v", resume.Params)
			}
			for _, raw := range []string{
				lateResponse,
				fmt.Sprintf(`{"id":%q,"result":{}}`, unsubscribe.ID),
				`{"method":"thread/started","params":{"thread":{"id":"thread-old","cwd":"/old"}}}`,
				`{"method":"turn/started","params":{"threadId":"thread-old","turn":{"id":"old-turn"}}}`,
			} {
				result := sharedSubscriptionObserve(t, tr, raw)
				if len(result.Events) != 0 || len(result.OutboundToCodex) != 0 {
					t.Fatalf("old subscription affected new selection: %#v", result)
				}
			}
			resumed := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"thread":{"id":"thread-new","cwd":"/new"}}}`, resume.ID))
			list := sharedSubscriptionOnlyFrame(t, resumed.OutboundToCodex, "thread/turns/list")
			ready := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"data":[]}}`, list.ID))
			sharedSubscriptionEvent(t, ready, "sub-new", "thread-new", "ready")
			sharedSubscriptionAssertDirectPrompt(t, tr, "thread-new")
		})
	}
}

func TestSharedSubscriptionReportsResumeAndTurnLookupFailures(t *testing.T) {
	for _, tt := range []struct {
		name      string
		readTurns bool
		response  string
		wantError string
	}{
		{name: "resume rejected", response: `"error":{"code":-32000,"message":"resume denied"}`, wantError: "resume denied"},
		{name: "resume wrong thread", response: `"result":{"thread":{"id":"other-thread"}}`, wantError: "thread"},
		{name: "turn lookup rejected", readTurns: true, response: `"error":{"code":-32000,"message":"turn lookup failed"}`, wantError: "turn lookup failed"},
		{name: "turn lookup missing data", readTurns: true, response: `"result":{}`, wantError: "data"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewTranslator("inst-1")
			tr.SetSharedAppServerMode(true)
			request := sharedSubscriptionBegin(t, tr, "sub1", "thread-1")
			if tt.readTurns {
				resumed := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"thread":{"id":"thread-1","cwd":"/repo"}}}`, request.ID))
				request = sharedSubscriptionOnlyFrame(t, resumed.OutboundToCodex, "thread/turns/list")
			}
			failed := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,%s}`, request.ID, tt.response))
			event := sharedSubscriptionEvent(t, failed, "sub1", "thread-1", "failed")
			if event.TurnID != "" || !strings.Contains(event.ErrorMessage, tt.wantError) {
				t.Fatalf("unexpected failure details: %#v", event)
			}
		})
	}
}

func TestSharedSubscriptionIdlePromptReusesDesktopThread(t *testing.T) {
	tr := NewTranslator("inst-1")
	tr.SetSharedAppServerMode(true)
	event := sharedSubscriptionReady(t, tr, "sub1", "thread-1", `[{"id":"finished-turn","status":"completed"}]`)
	if event.TurnID != "" {
		t.Fatalf("completed history must not become an active turn: %#v", event)
	}
	request := sharedSubscriptionAssertDirectPrompt(t, tr, "thread-1")
	sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"turn":{"id":"remote-turn","status":"inProgress"}}}`, request.ID))
	started := sharedSubscriptionObserve(t, tr, `{"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"remote-turn"}}}`)
	if len(started.Events) != 1 || started.Events[0].Initiator.Kind != agentproto.InitiatorRemoteSurface || started.Events[0].Initiator.SurfaceSessionID != "feishu-chat" {
		t.Fatalf("mobile prompt lost its remote initiator: %#v", started.Events)
	}
}

func TestSharedSubscriptionNewPromptAdoptsCreatedThreadAndRemoteInitiator(t *testing.T) {
	for _, previousThread := range []string{"", "old-thread"} {
		t.Run("previous="+previousThread, func(t *testing.T) {
			tr := NewTranslator("inst-1")
			tr.SetSharedAppServerMode(true)
			if previousThread != "" {
				sharedSubscriptionReady(t, tr, "sub-old", previousThread, `[]`)
			}
			frames, err := tr.TranslateCommand(agentproto.Command{
				Kind: agentproto.CommandPromptSend, CommandID: "new-prompt",
				Origin: agentproto.Origin{Surface: "feishu-new-chat"},
				Target: agentproto.Target{CWD: "/repo"},
				Prompt: agentproto.Prompt{Inputs: []agentproto.Input{{Type: agentproto.InputText, Text: "new task"}}},
			})
			if err != nil {
				t.Fatalf("new prompt: %v", err)
			}
			start := sharedSubscriptionOnlyFrame(t, frames, "thread/start")
			unrelated := sharedSubscriptionObserve(t, tr, `{"method":"thread/started","params":{"thread":{"id":"unrelated-thread","cwd":"/repo"}}}`)
			if len(unrelated.Events) != 0 {
				t.Fatalf("unrelated notification must not select the created thread: %#v", unrelated.Events)
			}
			created := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"thread":{"id":"created-thread","cwd":"/repo","status":{"type":"idle"}}}}`, start.ID))
			if !created.Suppress || len(created.Events) != 2 {
				t.Fatalf("expected subscription and discovery for exact create response: %#v", created)
			}
			event := created.Events[0]
			if event.Kind != agentproto.EventThreadSubscribed || event.Status != "created" || event.ThreadID != "created-thread" || event.CommandID != "new-prompt" {
				t.Fatalf("created thread not correlated with original prompt: %#v", event)
			}
			if created.Events[1].Kind != agentproto.EventThreadDiscovered || created.Events[1].ThreadID != "created-thread" {
				t.Fatalf("created thread discovery missing: %#v", created.Events)
			}
			followups := created.OutboundToCodex
			if previousThread != "" {
				if len(followups) != 2 {
					t.Fatalf("expected old unsubscribe and new turn/start: %q", followups)
				}
				unsubscribe := sharedSubscriptionOnlyFrame(t, followups[:1], "thread/unsubscribe")
				if unsubscribe.Params["threadId"] != previousThread {
					t.Fatalf("wrong old subscription released: %#v", unsubscribe.Params)
				}
				followups = followups[1:]
			}
			turn := sharedSubscriptionOnlyFrame(t, followups, "turn/start")
			if turn.Params["threadId"] != "created-thread" {
				t.Fatalf("new prompt targeted wrong thread: %#v", turn.Params)
			}
			sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"turn":{"id":"created-turn","status":"inProgress"}}}`, turn.ID))
			started := sharedSubscriptionObserve(t, tr, `{"method":"turn/started","params":{"threadId":"created-thread","turn":{"id":"created-turn"}}}`)
			if len(started.Events) != 1 || started.Events[0].Initiator.Kind != agentproto.InitiatorRemoteSurface || started.Events[0].Initiator.SurfaceSessionID != "feishu-new-chat" {
				t.Fatalf("created turn lost remote ownership: %#v", started.Events)
			}
			old := sharedSubscriptionObserve(t, tr, `{"method":"turn/started","params":{"threadId":"old-thread","turn":{"id":"old-turn"}}}`)
			if len(old.Events) != 0 {
				t.Fatalf("old thread event leaked after creation: %#v", old.Events)
			}
			sharedSubscriptionAssertDirectPrompt(t, tr, "created-thread")
		})
	}
}

func TestSharedSubscriptionDoesNotOverwriteNewerTurnLifecycleWithSnapshot(t *testing.T) {
	for _, tt := range []struct {
		name         string
		notification string
		turns        string
	}{
		{name: "turn started during lookup", notification: `{"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"new-turn"}}}`, turns: `[]`},
		{name: "turn completed during lookup", notification: `{"method":"turn/completed","params":{"threadId":"thread-1","turn":{"id":"old-turn","status":"completed"}}}`, turns: `[{"id":"old-turn","status":"inProgress"}]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewTranslator("inst-1")
			tr.SetSharedAppServerMode(true)
			resume := sharedSubscriptionBegin(t, tr, "sub1", "thread-1")
			resumed := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"thread":{"id":"thread-1","cwd":"/repo"}}}`, resume.ID))
			list := sharedSubscriptionOnlyFrame(t, resumed.OutboundToCodex, "thread/turns/list")
			observed := sharedSubscriptionObserve(t, tr, tt.notification)
			if len(observed.Events) != 1 {
				t.Fatalf("live lifecycle must remain visible while subscribing: %#v", observed)
			}
			ready := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"data":%s}}`, list.ID, tt.turns))
			event := sharedSubscriptionEvent(t, ready, "sub1", "thread-1", "ready")
			if event.TurnID != "" || event.Metadata["turnSnapshotStale"] != true {
				t.Fatalf("stale lookup must not replace newer live turn state: %#v", event)
			}
		})
	}
}

type sharedSubscriptionTestFrame struct {
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
}

func sharedSubscriptionOnlyFrame(t *testing.T, frames [][]byte, method string) sharedSubscriptionTestFrame {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("expected only %s, got %q", method, frames)
	}
	var frame sharedSubscriptionTestFrame
	if err := json.Unmarshal(frames[0], &frame); err != nil {
		t.Fatalf("decode native command: %v", err)
	}
	if frame.ID == "" || frame.Method != method {
		t.Fatalf("expected %s with request id, got %#v", method, frame)
	}
	return frame
}

func sharedSubscriptionObserve(t *testing.T, tr *Translator, raw string) Result {
	t.Helper()
	result, err := tr.ObserveServer([]byte(raw))
	if err != nil {
		t.Fatalf("observe server: %v", err)
	}
	return result
}

func sharedSubscriptionBegin(t *testing.T, tr *Translator, commandID, threadID string) sharedSubscriptionTestFrame {
	t.Helper()
	frames, err := tr.TranslateCommand(agentproto.Command{Kind: agentproto.CommandThreadSubscribe, CommandID: commandID, Target: agentproto.Target{ThreadID: threadID}})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	return sharedSubscriptionOnlyFrame(t, frames, "thread/resume")
}

func sharedSubscriptionReady(t *testing.T, tr *Translator, commandID, threadID, turns string) agentproto.Event {
	t.Helper()
	resume := sharedSubscriptionBegin(t, tr, commandID, threadID)
	resumed := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"thread":{"id":%q,"cwd":"/repo","status":{"type":"idle"}}}}`, resume.ID, threadID))
	list := sharedSubscriptionOnlyFrame(t, resumed.OutboundToCodex, "thread/turns/list")
	ready := sharedSubscriptionObserve(t, tr, fmt.Sprintf(`{"id":%q,"result":{"data":%s}}`, list.ID, turns))
	return sharedSubscriptionEvent(t, ready, commandID, threadID, "ready")
}

func sharedSubscriptionEvent(t *testing.T, result Result, commandID, threadID, status string) agentproto.Event {
	t.Helper()
	if !result.Suppress || len(result.OutboundToCodex) != 0 || len(result.Events) != 1 {
		t.Fatalf("expected one subscription event without executing a turn: %#v", result)
	}
	event := result.Events[0]
	if event.Kind != agentproto.EventThreadSubscribed || event.CommandID != commandID || event.ThreadID != threadID || event.Status != status {
		t.Fatalf("unexpected subscription event: %#v", event)
	}
	return event
}

func sharedSubscriptionAssertDirectPrompt(t *testing.T, tr *Translator, threadID string) sharedSubscriptionTestFrame {
	t.Helper()
	frames, err := tr.TranslateCommand(agentproto.Command{
		Kind:   agentproto.CommandPromptSend,
		Origin: agentproto.Origin{Surface: "feishu-chat"},
		Target: agentproto.Target{ThreadID: threadID},
		Prompt: agentproto.Prompt{Inputs: []agentproto.Input{{Type: agentproto.InputText, Text: "continue"}}},
	})
	if err != nil {
		t.Fatalf("send prompt to subscribed thread: %v", err)
	}
	frame := sharedSubscriptionOnlyFrame(t, frames, "turn/start")
	if frame.Params["threadId"] != threadID {
		t.Fatalf("prompt did not reuse selected desktop thread: %#v", frame.Params)
	}
	return frame
}
