package orchestrator

import (
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
)

// Only the current subscription owns these observations. Reconnect/switch drops
// the small dedup state along with the subscription, never persisting it as a setting.
type sharedModelNoticeState struct {
	Code, TurnID, Model, Effort, Reason string
}

func (s *Service) sharedModelNotice(instanceID string, event agentproto.Event, code, title string) []eventcontract.Event {
	inst := s.root.Instances[instanceID]
	if inst == nil || !inst.Online || !inst.Capabilities.SharedAppServer || event.TrafficClass == agentproto.TrafficClassInternalHelper {
		return nil
	}
	sub := s.sharedSubscriptions[instanceID]
	surface := s.turnSurface(instanceID, event.ThreadID, event.TurnID)
	if sub == nil || !sub.Ready || sub.ThreadID != event.ThreadID || surface == nil || surface.AttachedInstanceID != instanceID || surface.SelectedThreadID != event.ThreadID {
		return nil
	}
	model, effort := strings.TrimSpace(event.Model), strings.TrimSpace(event.ReasoningEffort)
	// Only use backend observations; never infer an actual model from the bot
	// override, workspace default or a requested resume policy.
	if code == "shared_turn_model" && event.TurnID != "" && sub.ModelNoticeTurnID == event.TurnID {
		return nil
	}
	if code == "shared_model_settings" {
		if model == "" {
			model = sub.ModelNotice.Model
		}
		effortCleared, _ := event.Metadata["reasoningEffortCleared"].(bool)
		if effort == "" && !effortCleared {
			effort = sub.ModelNotice.Effort
		}
	}
	current := sharedModelNoticeState{Code: code, TurnID: event.TurnID, Model: model, Effort: effort}
	if event.ModelReroute != nil {
		current.Reason = event.ModelReroute.Reason
	}
	if current == sub.ModelNotice {
		return nil
	}
	if code == "shared_model_settings" && model == sub.ModelNotice.Model && effort == sub.ModelNotice.Effort {
		return nil
	}
	sub.ModelNotice = current
	if code == "shared_turn_model" {
		sub.ModelNoticeTurnID = event.TurnID
	}
	sections := []control.FeishuCardTextSection{
		{Label: "模型（后台确认）", Lines: []string{observedModelValue(model)}},
		{Label: "推理强度（后台确认）", Lines: []string{observedModelValue(effort)}},
	}
	if code == "shared_model_rerouted" && event.ModelReroute != nil {
		sections = append(sections, control.FeishuCardTextSection{Label: "模型变化", Lines: []string{observedModelValue(event.ModelReroute.FromModel) + " → " + observedModelValue(model)}})
		if current.Reason != "" {
			sections = append(sections, control.FeishuCardTextSection{Label: "后台报告原因", Lines: []string{current.Reason}})
		}
	}
	override := state.EffectiveSurfaceCapabilitySettings(s.root, surface).PromptOverride
	if override.Model != "" || override.ReasoningEffort != "" {
		m, e := override.Model, override.ReasoningEffort
		if m == "" {
			m = "跟随电脑"
		}
		if e == "" {
			e = "跟随电脑"
		}
		sections = append(sections, control.FeishuCardTextSection{Label: "后续飞书输入的固定设置", Lines: []string{m + " / " + e, "发送 /model clear 可恢复跟随电脑。"}})
	}
	if binding := s.lookupRemoteTurn(instanceID, event.ThreadID, event.TurnID); binding != nil {
		if item := surface.QueueItems[binding.QueueItemID]; item != nil {
			requested := item.FrozenOverride
			if requested.Model != "" && model != "" && requested.Model != model || requested.ReasoningEffort != "" && effort != "" && requested.ReasoningEffort != effort {
				sections = append(sections, control.FeishuCardTextSection{Label: "请求与后台确认不一致", Lines: []string{"本轮请求：" + observedModelValue(requested.Model) + " / " + observedModelValue(requested.ReasoningEffort)}})
			}
		}
	}
	return []eventcontract.Event{surfaceEventFromPayload(surface, eventcontract.NoticePayload{Notice: control.Notice{Code: code, Title: title, Sections: sections}}, eventcontract.EventMeta{})}
}

func observedModelValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "未确认"
	}
	return value
}
