package feishu

import (
	gatewaypkg "github.com/YChange01/codex-feishu-link/internal/adapter/feishu/gateway"
)

func ResolveReceiveTarget(chatID, actorUserID string) (string, string) {
	return gatewaypkg.ResolveReceiveTarget(chatID, actorUserID)
}

func reactionKey(messageID, emojiType string) string {
	return messageID + "|" + emojiType
}

func mimeExtension(mimeType string) string {
	switch mimeType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
