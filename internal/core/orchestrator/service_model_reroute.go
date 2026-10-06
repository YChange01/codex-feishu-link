package orchestrator

import (
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
)

func (s *Service) applyTurnModelReroute(instanceID string, event agentproto.Event) []eventcontract.Event {
	inst := s.root.Instances[instanceID]
	if inst == nil {
		return nil
	}
	reroute := agentproto.NormalizeTurnModelReroute(event.ModelReroute)
	if reroute == nil {
		reroute = agentproto.NormalizeTurnModelReroute(&agentproto.TurnModelReroute{
			ThreadID: event.ThreadID,
			TurnID:   event.TurnID,
		})
	}
	if reroute == nil {
		return nil
	}
	if reroute.ThreadID == "" {
		reroute.ThreadID = strings.TrimSpace(event.ThreadID)
	}
	if reroute.TurnID == "" {
		reroute.TurnID = strings.TrimSpace(event.TurnID)
	}
	if reroute.ThreadID == "" || reroute.TurnID == "" {
		return nil
	}
	if inst.Capabilities.SharedAppServer && (inst.ActiveThreadID != reroute.ThreadID || inst.ActiveTurnID != reroute.TurnID) {
		return nil
	}
	thread := s.ensureThread(inst, reroute.ThreadID)
	thread.LastModelReroute = agentproto.CloneTurnModelReroute(reroute)
	if reroute.ToModel != "" {
		thread.ExplicitModel = reroute.ToModel
	}
	if binding := s.lookupRemoteTurn(instanceID, reroute.ThreadID, reroute.TurnID); binding != nil {
		binding.ModelReroute = agentproto.CloneTurnModelReroute(reroute)
	}
	event.Model, event.ThreadID, event.TurnID = reroute.ToModel, reroute.ThreadID, reroute.TurnID
	event.ModelReroute = reroute
	return s.sharedModelNotice(instanceID, event, "shared_model_rerouted", "本轮模型已切换")
}
