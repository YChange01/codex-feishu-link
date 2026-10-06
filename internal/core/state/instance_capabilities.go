package state

import "github.com/YChange01/codex-feishu-link/internal/core/agentproto"

func EffectiveInstanceBackend(inst *InstanceRecord) agentproto.Backend {
	if inst == nil {
		return agentproto.BackendCodex
	}
	return agentproto.NormalizeBackend(inst.Backend)
}

func EffectiveInstanceCapabilities(inst *InstanceRecord) agentproto.Capabilities {
	if inst == nil {
		return agentproto.Capabilities{}
	}
	if inst.CapabilitiesDeclared {
		return inst.Capabilities
	}
	return agentproto.EffectiveCapabilitiesForBackend(inst.Backend, inst.Capabilities)
}

func InstanceSupportsThreadsRefresh(inst *InstanceRecord) bool {
	return inst != nil && EffectiveInstanceCapabilities(inst).ThreadsRefresh
}

func InstanceSupportsModelCatalog(inst *InstanceRecord) bool {
	return inst != nil &&
		EffectiveInstanceBackend(inst) == agentproto.BackendCodex &&
		EffectiveInstanceCapabilities(inst).ModelCatalog
}
