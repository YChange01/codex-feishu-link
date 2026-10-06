package feishu

import (
	"testing"

	"github.com/YChange01/codex-feishu-link/internal/core/control"
)

func TestMenuActionReasoningPresets(t *testing.T) {
	tests := map[string]string{
		"reasoning_low":    "/reasoning low",
		"reason_low":       "/reasoning low",
		"reasonlow":        "/reasoning low",
		"reasoning_medium": "/reasoning medium",
		"reason_medium":    "/reasoning medium",
		"reasonmedium":     "/reasoning medium",
		"reasoning_high":   "/reasoning high",
		"reason_high":      "/reasoning high",
		"reasonhigh":       "/reasoning high",
		"reasoning_xhigh":  "/reasoning xhigh",
		"reason_xhigh":     "/reasoning xhigh",
		"reasonxhigh":      "/reasoning xhigh",
		"reasoning_clear":  "/reasoning clear",
	}
	for key, wantText := range tests {
		got, ok := menuAction(key)
		if !ok {
			t.Fatalf("expected menu action for %q", key)
		}
		if got.Kind != control.ActionReasoningCommand || got.Text != wantText {
			t.Fatalf("event key %q => %#v, want reasoning command %q", key, got, wantText)
		}
	}
}

func TestMenuActionDynamicModelPreset(t *testing.T) {
	tests := map[string]string{
		"model_gpt-5.5":       "/model gpt-5.5",
		"model_gpt-5.5-mini":  "/model gpt-5.5-mini",
		"model-gpt-5.5":       "/model gpt-5.5",
		" model_gpt-5.5 \n\t": "/model gpt-5.5",
	}
	for key, wantText := range tests {
		got, ok := menuAction(key)
		if !ok {
			t.Fatalf("expected dynamic model action for %q", key)
		}
		if got.Kind != control.ActionModelCommand || got.Text != wantText {
			t.Fatalf("event key %q => %#v, want model command %q", key, got, wantText)
		}
	}
}

func TestMenuActionAccessPresets(t *testing.T) {
	tests := map[string]string{
		"accessfull":     "/permission full",
		"access_full":    "/permission full",
		"accessFull":     "/permission full",
		"accessconfirm":  "/permission confirm",
		"access_confirm": "/permission confirm",
		"accessConfirm":  "/permission confirm",
	}
	for key, wantText := range tests {
		got, ok := menuAction(key)
		if !ok {
			t.Fatalf("expected menu action for %q", key)
		}
		if got.Kind != control.ActionAccessCommand || got.Text != wantText {
			t.Fatalf("event key %q => %#v, want access command %q", key, got, wantText)
		}
	}
}
