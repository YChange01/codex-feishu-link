package orchestrator

import (
	"strings"
	"time"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
)

// Connection-local readiness is intentionally absent from persisted route state.
type sharedThreadSubscription struct {
	ThreadID          string
	CommandID         string
	Ready             bool
	Pending           bool
	StartedAt         time.Time
	UserMessageIDs    []string
	ModelNotice       sharedModelNoticeState
	ModelNoticeTurnID string
}

func (s *Service) ensureSharedThreadSubscription(surface *state.SurfaceConsoleRecord) []eventcontract.Event {
	if surface == nil || surface.SelectedThreadID == "" {
		return nil
	}
	inst := s.root.Instances[surface.AttachedInstanceID]
	if inst == nil || !inst.Online || !inst.Capabilities.SharedAppServer {
		return nil
	}
	if current := s.sharedSubscriptions[inst.InstanceID]; current != nil && current.ThreadID == surface.SelectedThreadID && (current.Ready || current.Pending) {
		return nil
	}
	if s.sharedSubscriptions == nil {
		s.sharedSubscriptions = map[string]*sharedThreadSubscription{}
	}
	commandID := s.nextAgentCommandID()
	s.sharedSubscriptions[inst.InstanceID] = &sharedThreadSubscription{
		ThreadID: surface.SelectedThreadID, CommandID: commandID, Pending: true, StartedAt: s.now(),
	}
	if inst.ActiveThreadID != surface.SelectedThreadID {
		inst.ActiveThreadID, inst.ActiveTurnID = surface.SelectedThreadID, ""
	}
	return []eventcontract.Event{{
		Kind: eventcontract.KindAgentCommand, SurfaceSessionID: surface.SurfaceSessionID,
		Command: &agentproto.Command{
			CommandID: commandID, Kind: agentproto.CommandThreadSubscribe,
			Origin: agentproto.Origin{Surface: surface.SurfaceSessionID, UserID: surface.ActorUserID, ChatID: surface.ChatID},
			Target: agentproto.Target{ThreadID: surface.SelectedThreadID},
		},
	}}
}

func (s *Service) sharedThreadSubscriptionReady(instanceID, threadID string) bool {
	current := s.sharedSubscriptions[instanceID]
	return current != nil && current.Ready && current.ThreadID == threadID
}

func (s *Service) sharedThreadEventAllowed(inst *state.InstanceRecord, event agentproto.Event) bool {
	if inst == nil || !inst.Capabilities.SharedAppServer || event.ThreadID == "" {
		return true
	}
	switch event.Kind {
	case agentproto.EventThreadSubscribed, agentproto.EventThreadsSnapshot, agentproto.EventThreadHistoryRead, agentproto.EventThreadDiscovered:
		return true
	}
	current := s.sharedSubscriptions[inst.InstanceID]
	if current == nil || current.ThreadID != event.ThreadID {
		return false
	}
	if s.lookupRemoteTurnForEvent(inst.InstanceID, event) != nil {
		return true
	}
	owner := s.threadClaimSurface(event.ThreadID)
	return owner != nil && owner.AttachedInstanceID == inst.InstanceID && owner.SelectedThreadID == event.ThreadID
}

func (s *Service) applySharedThreadSubscription(inst *state.InstanceRecord, event agentproto.Event) []eventcontract.Event {
	if event.Status == "created" {
		if s.pendingRemoteBindingByCommandForInstance(inst.InstanceID, event.CommandID) == nil {
			return nil
		}
		if s.sharedSubscriptions == nil {
			s.sharedSubscriptions = map[string]*sharedThreadSubscription{}
		}
		s.sharedSubscriptions[inst.InstanceID] = &sharedThreadSubscription{ThreadID: event.ThreadID, Ready: true}
		return nil
	}
	current := s.sharedSubscriptions[inst.InstanceID]
	if current == nil || current.ThreadID != event.ThreadID || current.CommandID != event.CommandID || !current.Pending {
		return nil
	}
	if event.Status != "ready" || event.ErrorMessage != "" {
		return s.failSharedThreadSubscription(inst.InstanceID, event.ErrorMessage)
	}
	current.Pending, current.Ready = false, true
	thread := s.ensureThread(inst, event.ThreadID)
	if event.Model != "" {
		thread.ExplicitModel = event.Model
	}
	if event.ReasoningEffort != "" {
		thread.ExplicitReasoningEffort = event.ReasoningEffort
	}
	var events []eventcontract.Event
	stale, _ := event.Metadata["turnSnapshotStale"].(bool)
	if !stale {
		if event.TurnID != "" && inst.ActiveTurnID != event.TurnID {
			events = append(events, s.ApplyAgentEvent(inst.InstanceID, agentproto.Event{
				Kind: agentproto.EventTurnStarted, ThreadID: event.ThreadID, TurnID: event.TurnID,
				Status: "running", Initiator: agentproto.Initiator{Kind: agentproto.InitiatorLocalUI},
				Model: event.Model, ReasoningEffort: event.ReasoningEffort,
			})...)
		} else if event.TurnID == "" {
			inst.ActiveTurnID = ""
		}
	}
	if event.TurnID == "" && !stale {
		events = append(events, s.sharedModelNotice(inst.InstanceID, event, "shared_model_connected", "已连接桌面会话")...)
	}
	for _, surface := range s.findAttachedSurfaces(inst.InstanceID) {
		if surface.SelectedThreadID == event.ThreadID {
			events = append(events, s.sharedDesktopOpenEvents(surface, inst)...)
			events = append(events, s.dispatchNext(surface)...)
		}
	}
	return events
}

func (s *Service) failSharedThreadSubscription(instanceID, reason string) []eventcontract.Event {
	current := s.sharedSubscriptions[instanceID]
	if current == nil {
		return nil
	}
	current.Pending, current.Ready = false, false
	text := "桌面会话订阅失败，输入仍保留在队列中。发送 /status 或重新选择会话可重试。"
	if reason = strings.TrimSpace(reason); reason != "" {
		text += " " + reason
	}
	var events []eventcontract.Event
	for _, surface := range s.findAttachedSurfaces(instanceID) {
		if surface.SelectedThreadID == current.ThreadID {
			events = append(events, notice(surface, "shared_thread_subscription_failed", text)...)
		}
	}
	return events
}

func (s *Service) expireSharedThreadSubscriptions(now time.Time) []eventcontract.Event {
	var events []eventcontract.Event
	for instanceID, current := range s.sharedSubscriptions {
		if current.Pending && !now.Before(current.StartedAt.Add(30*time.Second)) {
			events = append(events, s.failSharedThreadSubscription(instanceID, "等待订阅响应超时。")...)
		}
	}
	return events
}

func (s *Service) surfaceOwnsSharedThread(surface *state.SurfaceConsoleRecord, instanceID, threadID string) bool {
	if surface == nil || surface.AttachedInstanceID != instanceID {
		return false
	}
	if surface.SelectedThreadID == threadID {
		return true
	}
	binding := s.activeRemoteBindingForSurface(surface)
	return binding != nil && remoteBindingExecutionThreadID(binding) == threadID
}

// Only a confirmed subscription may request desktop UI navigation.
func (s *Service) sharedDesktopOpenEvents(surface *state.SurfaceConsoleRecord, inst *state.InstanceRecord) []eventcontract.Event {
	if surface == nil || inst == nil || !inst.Online || !inst.Capabilities.SharedAppServer || surface.AttachedInstanceID != inst.InstanceID || !s.sharedThreadSubscriptionReady(inst.InstanceID, surface.SelectedThreadID) {
		return nil
	}
	thread := inst.Threads[surface.SelectedThreadID]
	if thread == nil || strings.TrimSpace(thread.CWD) == "" {
		return nil
	}
	return []eventcontract.Event{{Kind: eventcontract.KindDaemonCommand, SurfaceSessionID: surface.SurfaceSessionID, DaemonCommand: &control.DaemonCommand{Kind: control.DaemonCommandOpenSharedDesktop, SurfaceSessionID: surface.SurfaceSessionID, InstanceID: inst.InstanceID, ThreadID: surface.SelectedThreadID, ThreadCWD: thread.CWD}}}
}
