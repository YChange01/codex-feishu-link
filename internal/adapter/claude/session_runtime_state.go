package claude

import (
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/claudesessionstore"
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
)

func (t *Translator) RuntimeStateSnapshot() claudesessionstore.RuntimeStateSnapshot {
	if t == nil {
		return claudesessionstore.RuntimeStateSnapshot{}
	}
	snapshot := claudesessionstore.RuntimeStateSnapshot{
		SessionID:            strings.TrimSpace(t.sessionID),
		CWD:                  strings.TrimSpace(t.cwd),
		Model:                strings.TrimSpace(t.model),
		NativePermissionMode: strings.TrimSpace(t.permissionMode),
	}
	snapshot.ObservedPermission = claudesessionstore.CompileObservedPermissionStateFromClaudeNative(snapshot.NativePermissionMode)
	selection := claudePermissionSelectionFromNative(snapshot.NativePermissionMode)
	snapshot.AccessMode = selection.AccessMode
	snapshot.PlanMode = selection.PlanMode
	if t.activeTurn != nil {
		snapshot.ActiveTurnID = strings.TrimSpace(t.activeTurn.TurnID)
	}
	for _, request := range t.pendingRequests {
		if request == nil {
			continue
		}
		switch request.RequestType {
		case agentproto.RequestTypeApproval:
			snapshot.WaitingOnApproval = true
		case agentproto.RequestTypeRequestUserInput:
			snapshot.WaitingOnUserInput = true
		}
	}
	return snapshot
}
