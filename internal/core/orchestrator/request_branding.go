package orchestrator

import (
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
)

func requestPromptBackend(record *state.RequestPromptRecord) agentproto.Backend {
	if record == nil {
		return agentproto.BackendCodex
	}
	return agentproto.NormalizeBackend(record.Backend)
}

func requestLocalBackendDisplayName(backend agentproto.Backend) string {
	return control.RequestLocalBackendDisplayName(backend)
}

func requestFeedbackActionLabel(backend agentproto.Backend) string {
	return control.RequestFeedbackActionLabel(backend)
}

func requestWaitingContinueText(backend agentproto.Backend) string {
	return control.RequestWaitingContinueText(backend)
}
