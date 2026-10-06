package feishu

import (
	"testing"

	"github.com/YChange01/codex-feishu-link/internal/core/control"
	larkcallback "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
)

func TestModelEffortFormSubmitUsesBothSelectedValues(t *testing.T) {
	gateway := NewLiveGateway(LiveGatewayConfig{GatewayID: "app-1"})
	gateway.recordSurfaceMessage("om-model-effort", "feishu:app-1:user:user-1")
	userID := "user-1"
	build := func(values map[string]interface{}) *larkcallback.CardActionTriggerEvent {
		return &larkcallback.CardActionTriggerEvent{Event: &larkcallback.CardActionTriggerRequest{
			Operator: &larkcallback.Operator{UserID: &userID},
			Action: &larkcallback.CallBackAction{
				Value: map[string]interface{}{
					"kind": "page_submit", "action_kind": string(control.ActionModelCommand),
					"field_name": "command_args_model_preset", "secondary_field_name": "command_args_effort",
					"catalog_family_id": control.FeishuCommandModel, "catalog_variant_id": "model.codex.normal", "catalog_backend": "codex",
				},
				FormValue: values,
			},
			Context: &larkcallback.Context{OpenChatID: "oc_1", OpenMessageID: "om-model-effort"},
		}}
	}
	for _, tc := range []struct{ effort, want string }{{"high", "/model model-a high"}, {"clear", "/model model-a clear"}} {
		action, ok := gateway.parseCardActionTriggerEvent(build(map[string]interface{}{"command_args_model_preset": "model-a", "command_args_effort": tc.effort}))
		if !ok || action.Text != tc.want || action.Kind != control.ActionModelCommand {
			t.Fatalf("paired selection %q = %#v, %v", tc.effort, action, ok)
		}
	}
	if action, ok := gateway.parseCardActionTriggerEvent(build(map[string]interface{}{"command_args_model_preset": "model-a"})); !ok || action.Text != "/model model-a clear" {
		t.Fatalf("missing effort should use visible automatic default: %#v, %v", action, ok)
	}
	if action, ok := gateway.parseCardActionTriggerEvent(build(map[string]interface{}{"command_args_effort": "high"})); ok {
		t.Fatalf("accepted missing model: %#v", action)
	}
}
