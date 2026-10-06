package feishu

import (
	cardtransport "github.com/YChange01/codex-feishu-link/internal/adapter/feishu/cardtransport"
)

func feishuInteractiveMessageTransportFits(payload map[string]any) bool {
	return cardtransport.InteractiveMessagePayloadFits(payload)
}

func feishuInlineCallbackTransportFits(payload map[string]any) bool {
	return cardtransport.InlineCallbackPayloadFits(payload)
}
