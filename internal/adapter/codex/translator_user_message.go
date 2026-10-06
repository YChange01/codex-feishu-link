package codex

import (
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/xutil"
)

// User-authored commands/code must retain indentation and boundary whitespace;
// the generic tool-result extractor intentionally normalizes its summaries.
func extractUserMessageText(item map[string]any) string {
	if text, _ := item["text"].(string); text != "" {
		return text
	}
	var parts []string
	for _, raw := range contentArrayValues(item["content"]) {
		entry, _ := raw.(map[string]any)
		if normalizeStructuredContentType(xutil.LookupStringFromAny(entry["type"])) == "text" {
			parts = append(parts, choose(xutil.LookupStringFromAny(entry["text"]), xutil.LookupStringFromAny(entry["value"])))
		}
	}
	return strings.Join(parts, "\n")
}
