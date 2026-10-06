package codex

import (
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/jsonrpcutil"
)

// A desktop client may win a same-thread race. Only the RPC response proves
// which turn was started by this connection; notification order does not.
func (t *Translator) observeSharedTurnStartResponse(pending suppressedResponseContext, message map[string]any) Result {
	delete(t.pendingRemoteTurnByThread, pending.ThreadID)
	buffered := t.sharedPendingTurnEvents
	t.sharedPendingTurnEvents = nil
	initiator := agentproto.Initiator{Kind: agentproto.InitiatorRemoteSurface, SurfaceSessionID: pending.SurfaceSessionID}
	turnID := lookupString(message, "result", "turn", "id")
	problem := jsonrpcutil.ExtractErrorMessage(message)
	if problem == "" && turnID == "" {
		problem = "Codex turn/start response has no turn id"
	}
	if problem != "" {
		return Result{Suppress: true, Events: append(buffered, agentproto.Event{
			Kind: agentproto.EventTurnCompleted, ThreadID: pending.ThreadID, CommandID: pending.CommandID,
			Status: "failed", ErrorMessage: problem, Initiator: initiator,
			TurnCompletionOrigin: agentproto.TurnCompletionOriginTurnStartRejected,
		})}
	}
	t.turnInitiators[turnID] = initiator
	started := false
	for i := range buffered {
		if buffered[i].ThreadID == pending.ThreadID && buffered[i].TurnID == turnID {
			buffered[i].Initiator, buffered[i].CommandID = initiator, pending.CommandID
			started = started || buffered[i].Kind == agentproto.EventTurnStarted
		}
	}
	if !started {
		// Use the real RPC start evidence when its notification is still pending.
		result := t.observeTurnStarted(map[string]any{"params": map[string]any{
			"threadId": pending.ThreadID, "turn": lookupAny(message, "result", "turn"),
		}})
		for i := range result.Events {
			result.Events[i].CommandID = pending.CommandID
		}
		insertAt := len(buffered)
		for i, event := range buffered {
			if event.ThreadID == pending.ThreadID && event.TurnID == turnID {
				insertAt = i
				break
			}
		}
		events := append([]agentproto.Event(nil), buffered[:insertAt]...)
		events = append(events, result.Events...)
		buffered = append(events, buffered[insertAt:]...)
	}
	return Result{Suppress: true, Events: buffered}
}
