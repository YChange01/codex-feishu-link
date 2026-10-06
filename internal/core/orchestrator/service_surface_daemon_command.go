package orchestrator

import (
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
)

func (s *Service) boundDaemonCommandEvents(surface *state.SurfaceConsoleRecord, action control.Action) ([]eventcontract.Event, bool) {
	binding, ok := control.ResolveFeishuCommandBindingFromAction(action)
	if !ok || binding.DirectDaemonCommand == "" {
		return nil, false
	}
	command := &control.DaemonCommand{
		Kind:             binding.DirectDaemonCommand,
		GatewayID:        surface.GatewayID,
		SurfaceSessionID: surface.SurfaceSessionID,
		SourceMessageID:  action.MessageID,
		Text:             action.Text,
	}
	if binding.PropagateCardActionToDaemon || action.LocalPageAction {
		command.FromCardAction = action.IsCardAction()
	}
	return []eventcontract.Event{{
		Kind:             eventcontract.KindDaemonCommand,
		GatewayID:        surface.GatewayID,
		SurfaceSessionID: surface.SurfaceSessionID,
		SourceMessageID:  action.MessageID,
		DaemonCommand:    command,
	}}, true
}
