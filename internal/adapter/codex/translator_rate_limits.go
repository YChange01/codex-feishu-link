package codex

import (
	"encoding/json"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/jsonrpcutil"
)

func (t *Translator) translateRateLimitsRead(command agentproto.Command) ([][]byte, error) {
	requestID := t.NextRequest("rate-limits-read")
	t.pendingRateLimitsRead[requestID] = pendingRateLimitsRead{
		CommandID: command.CommandID,
		SurfaceID: command.Origin.Surface,
	}
	bytes, err := json.Marshal(map[string]any{"id": requestID, "method": "account/rateLimits/read"})
	if err != nil {
		return nil, err
	}
	return [][]byte{append(bytes, '\n')}, nil
}

func (t *Translator) observeRateLimitsReadResponse(requestID string, message map[string]any) (Result, bool) {
	pending, exists := t.pendingRateLimitsRead[requestID]
	if !exists {
		return Result{}, false
	}
	delete(t.pendingRateLimitsRead, requestID)
	update := &agentproto.CapabilityStateUpdate{
		Method: "account/rateLimits/read", RateLimitsReadComplete: true,
	}
	if errText := jsonrpcutil.ExtractErrorMessage(message); errText != "" {
		update.RateLimitsReadError = errText
	} else if result, _ := message["result"].(map[string]any); result == nil {
		update.RateLimitsReadError = "Codex response missing result"
	} else {
		update.RateLimits = extractSparseRateLimits(firstNonNil(result["rateLimitsByLimitId"], result["rateLimits"]))
		if credits, ok := result["rateLimitResetCredits"].(map[string]any); ok {
			if count, ok := numberFromAny(credits["availableCount"]); ok {
				available := int(count)
				update.RateLimitResetCreditsAvailable = &available
			}
		}
	}
	event := agentproto.Event{
		Kind: agentproto.EventCapabilityStateUpdated, CommandID: pending.CommandID,
		CapabilityState: update,
		Metadata:        map[string]any{"surfaceSessionId": pending.SurfaceID},
	}
	return Result{Suppress: true, Events: []agentproto.Event{event}}, true
}

func numberFromAny(value any) (float64, bool) {
	switch value := value.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
