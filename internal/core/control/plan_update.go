package control

import "github.com/YChange01/codex-feishu-link/internal/core/agentproto"

type PlanUpdateStep struct {
	Step   string
	Status agentproto.TurnPlanStepStatus
}

type PlanUpdate struct {
	ThreadID              string
	TurnID                string
	TemporarySessionLabel string
	Explanation           string
	Steps                 []PlanUpdateStep
}
