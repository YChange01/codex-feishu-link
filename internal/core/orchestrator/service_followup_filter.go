package orchestrator

import (
	"github.com/YChange01/codex-feishu-link/internal/core/eventcontract"
)

var dropNoticeFollowupPolicy = eventcontract.FollowupPolicy{
	DropClasses: []eventcontract.HandoffClass{
		eventcontract.HandoffClassNotice,
		eventcontract.HandoffClassThreadSelection,
	},
}

func filterFollowupEventsByPolicy(events []eventcontract.Event, policy eventcontract.FollowupPolicy) []eventcontract.Event {
	return eventcontract.FilterEventsByFollowupPolicy(events, policy)
}
