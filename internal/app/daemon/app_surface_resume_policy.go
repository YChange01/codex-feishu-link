package daemon

import (
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/app/daemon/surfaceresume"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
	"github.com/YChange01/codex-feishu-link/internal/feishuidentity"
)

func surfaceResumeEntryAllowsBackgroundRecovery(entry surfaceresume.Entry) bool {
	return !surfaceResumeEntryIsFeishuGroup(entry)
}

func surfaceAllowsDaemonLifecycleNotice(surface *state.SurfaceConsoleRecord) bool {
	return !surfaceIsFeishuGroup(strings.TrimSpace(surfaceIDFromSurface(surface)))
}

func surfaceResumeEntryAllowsOnDemandRecovery(entry surfaceresume.Entry, action control.Action) bool {
	return surfaceResumeEntryIsFeishuGroup(entry) &&
		action.Kind == control.ActionTextMessage &&
		strings.TrimSpace(action.SurfaceSessionID) == strings.TrimSpace(entry.SurfaceSessionID)
}

func surfaceResumeEntryIsFeishuGroup(entry surfaceresume.Entry) bool {
	return surfaceIsFeishuGroup(strings.TrimSpace(entry.SurfaceSessionID))
}

func surfaceIDFromSurface(surface *state.SurfaceConsoleRecord) string {
	if surface == nil {
		return ""
	}
	return surface.SurfaceSessionID
}

func surfaceIsFeishuGroup(surfaceID string) bool {
	ref, ok := feishuidentity.ParseSurfaceRef(surfaceID)
	return ok && ref.IsChat()
}
