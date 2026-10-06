package control

import "testing"

func TestPermissionCommandAliasesKeepExistingAccessContract(t *testing.T) {
	for _, alias := range []string{"/permission", "/permissions", "/access", "/approval"} {
		for _, argument := range []string{"", " full", " confirm", " clear"} {
			input := alias + argument
			action, ok := ParseFeishuTextActionWithoutCatalog(input)
			if !ok || action.Kind != ActionAccessCommand || action.CommandID != FeishuCommandAccess {
				t.Fatalf("%q did not resolve to existing access contract: %#v, %v", input, action, ok)
			}
			if action.Text != input {
				t.Fatalf("parser lost permission argument: got %q, want %q", action.Text, input)
			}
		}
	}
	for _, argument := range []string{"full", "confirm", "clear"} {
		if got := BuildFeishuActionText(ActionAccessCommand, argument); got != "/permission "+argument {
			t.Fatalf("canonical permission action = %q", got)
		}
	}
	for _, key := range []string{"access", "approval", "access_confirm", "approval_confirm"} {
		action, ok := ParseFeishuMenuActionWithoutCatalog(key)
		if !ok || action.Kind != ActionAccessCommand || action.CommandID != FeishuCommandAccess {
			t.Fatalf("legacy menu key %q lost compatibility: %#v, %v", key, action, ok)
		}
	}
}
