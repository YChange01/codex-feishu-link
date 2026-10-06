package orchestrator

import (
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
	"github.com/YChange01/codex-feishu-link/internal/xutil"
)

// Input origin is per item: either client can steer a turn started by the
// other. Turn ownership must never be used to decide whether to echo input.
func (s *Service) renderDesktopUserMessage(instanceID string, event agentproto.Event) []eventcontract.Event {
	inst := s.root.Instances[instanceID]
	if inst == nil || !inst.Capabilities.SharedAppServer || event.TrafficClass == agentproto.TrafficClassInternalHelper {
		return nil
	}
	sub := s.sharedSubscriptions[instanceID]
	surface := s.threadClaimSurface(event.ThreadID)
	if sub == nil || sub.ThreadID != event.ThreadID || surface == nil || surface.AttachedInstanceID != instanceID || surface.SelectedThreadID != event.ThreadID {
		return nil
	}
	if strings.HasPrefix(xutil.MetadataString(event.Metadata, "clientId"), agentproto.RemoteUserMessageClientPrefix) {
		return nil
	}
	text, _ := event.Metadata["text"].(string)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	key := event.TurnID + ":" + event.ItemID
	for _, seen := range sub.UserMessageIDs {
		if seen == key {
			return nil
		}
	}
	// Connection-local, bounded deduplication of repeated completed events.
	sub.UserMessageIDs = append(sub.UserMessageIDs, key)
	if len(sub.UserMessageIDs) > 512 {
		sub.UserMessageIDs = sub.UserMessageIDs[len(sub.UserMessageIDs)-512:]
	}
	s.recordThreadUserMessage(inst, event.ThreadID, text)
	events := s.flushAndSealExecCommandProgressForTurn(instanceID, event.ThreadID, event.TurnID)
	// Stay below card transport limits even for long prompts / escaped text.
	remaining := []rune(text)
	for len(remaining) > 0 {
		n := min(len(remaining), 2000)
		part := string(remaining[:n])
		remaining = remaining[n:]
		events = append(events, surfaceEventFromPayload(surface, eventcontract.NoticePayload{Notice: control.Notice{
			Code: "desktop_user_message", Title: "电脑端输入",
			Sections: []control.FeishuCardTextSection{{Lines: []string{part}}},
		}}, eventcontract.EventMeta{}))
	}
	return events
}
