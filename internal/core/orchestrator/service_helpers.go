package orchestrator

import (
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
)

func isHeadlessInstance(inst *state.InstanceRecord) bool {
	return state.IsManagedHeadlessInstance(inst)
}

func isVSCodeInstance(inst *state.InstanceRecord) bool {
	return inst != nil && state.IsVSCodeOrDefaultSource(inst.Source)
}

func headlessThreadWorkspaceMustMatch(inst *state.InstanceRecord) bool {
	return isHeadlessInstance(inst) && state.EffectiveInstanceBackend(inst) == agentproto.BackendClaude
}
