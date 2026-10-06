package orchestrator

import (
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
)

func (s *Service) applyCapabilityStateUpdate(instanceID string, event agentproto.Event) []eventcontract.Event {
	inst := s.root.Instances[instanceID]
	if inst == nil {
		return nil
	}
	update := agentproto.NormalizeCapabilityStateUpdate(event.CapabilityState)
	if update == nil {
		return nil
	}
	if update.ThreadID == "" {
		update.ThreadID = strings.TrimSpace(event.ThreadID)
	}
	inst.LastCapabilityState = agentproto.CloneCapabilityStateUpdate(update)
	if update.ThreadID != "" {
		thread := s.ensureThread(inst, update.ThreadID)
		thread.LastCapabilityState = agentproto.CloneCapabilityStateUpdate(update)
	}
	events := s.projectCapabilityStateUpdate(instanceID, *update)
	if update.RateLimitsReadComplete {
		surfaceID, _ := event.Metadata["surfaceSessionId"].(string)
		surface := s.root.Surfaces[strings.TrimSpace(surfaceID)]
		if surface != nil && surface.AttachedInstanceID == instanceID {
			events = append(events, notice(surface, "codex_quota_result", formatQuotaNotice(update))...)
		}
	}
	return events
}
