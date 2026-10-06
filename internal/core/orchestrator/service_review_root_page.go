package orchestrator

import (
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
)

func (s *Service) reviewRootPageTriggeredFromMenu(surface *state.SurfaceConsoleRecord, sourceMessageID string) bool {
	if surface == nil {
		return false
	}
	sourceMessageID = strings.TrimSpace(sourceMessageID)
	if sourceMessageID == "" {
		return false
	}
	return s.activeCommandLauncherMessageID(surface) == sourceMessageID
}

func (s *Service) reviewRootPageEvent(surface *state.SurfaceConsoleRecord, fromMenu bool) eventcontract.Event {
	view := control.BuildFeishuReviewRootPageView(fromMenu)
	view.CatalogBackend = agentproto.NormalizeBackend(s.surfaceBackend(surface))
	if flow := s.activeCommandLauncherFlow(surface); flow != nil && flow.Role == frontstageFlowRoleLauncher {
		view.TrackingKey = strings.TrimSpace(flow.FlowID)
	}
	return s.pageEvent(surface, view)
}
