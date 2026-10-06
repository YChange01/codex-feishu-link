package wrapper

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/YChange01/codex-feishu-link/internal/config"
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	relayruntime "github.com/YChange01/codex-feishu-link/internal/runtime"
)

func TestNewBackendRuntimeSharedAppServerCapabilityAndSubscription(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(fmt.Sprintf("shared=%t", shared), func(t *testing.T) {
			runtime := newBackendRuntime(Config{InstanceID: "inst-1", SharedAppServer: shared})
			if runtime.Backend() != agentproto.BackendCodex {
				t.Fatalf("expected Codex backend, got %q", runtime.Backend())
			}
			if got := runtime.Capabilities().SharedAppServer; got != shared {
				t.Fatalf("SharedAppServer capability = %t, want %t", got, shared)
			}
			result, err := runtime.TranslateCommand(agentproto.Command{
				Kind: agentproto.CommandThreadSubscribe, CommandID: "sub1",
				Target: agentproto.Target{ThreadID: "thread-1"},
			})
			if !shared {
				if err == nil {
					t.Fatal("standalone backend unexpectedly accepted a shared-thread subscription")
				}
				return
			}
			if err != nil {
				t.Fatalf("shared backend subscription: %v", err)
			}
			if len(result.Phases) != 1 || len(result.Phases[0].OutboundToChild) != 1 {
				t.Fatalf("expected one subscription request, got %#v", result)
			}
			var request struct {
				Method string `json:"method"`
				Params struct {
					ThreadID     string `json:"threadId"`
					ExcludeTurns bool   `json:"excludeTurns"`
				} `json:"params"`
			}
			if err := json.Unmarshal(result.Phases[0].OutboundToChild[0], &request); err != nil {
				t.Fatalf("decode subscription request: %v", err)
			}
			if request.Method != "thread/resume" || request.Params.ThreadID != "thread-1" || !request.Params.ExcludeTurns {
				t.Fatalf("runtime did not configure shared translator: %#v", request)
			}
		})
	}
}

func TestBuildCodexChildLaunchSharedAppServerPreservesArgsWithoutMCPInjection(t *testing.T) {
	for _, inheritedMode := range []string{"", "false", "1"} {
		t.Run("inherited="+inheritedMode, func(t *testing.T) {
			t.Setenv(config.SharedAppServerEnv, inheritedMode)
			clearFeishuMCPBearerEnv(t)
			statePath := writeToolServiceState(t, `{
				"url": "http://127.0.0.1:9702",
				"token": "test-shared-mode-token",
				"tokenType": "bearer"
			}`)
			app := New(Config{
				InstanceID: "inst-shared", Source: "headless", SharedAppServer: true,
				RuntimePaths: relayruntime.Paths{ToolServiceFile: statePath},
			})
			baseArgs := []string{"-c", "features.code_mode_host=true", "app-server", "--analytics-default-enabled"}
			args, childEnv := app.buildCodexChildLaunch(baseArgs)
			if !reflect.DeepEqual(args, baseArgs) {
				t.Fatalf("shared connection must retain input args without MCP startup overrides: got %#v, want %#v", args, baseArgs)
			}
			if got := lookupEnv(childEnv, config.SharedAppServerEnv); got != "1" {
				t.Fatalf("child shared-mode env = %q, want 1", got)
			}
			modeEntries := 0
			for _, entry := range childEnv {
				if strings.HasPrefix(entry, config.SharedAppServerEnv+"=") {
					modeEntries++
				}
			}
			if modeEntries != 1 {
				t.Fatalf("expected exactly one authoritative shared-mode env entry, got %d", modeEntries)
			}
			if got := lookupEnv(childEnv, feishuMCPBearerEnvName); got != "" {
				t.Fatalf("shared connection unexpectedly received startup MCP token: %q", got)
			}
		})
	}
}

func TestSharedAppServerRejectsChildRestartAndPreservesReadySubscription(t *testing.T) {
	runtime := newBackendRuntime(Config{InstanceID: "inst-shared", SharedAppServer: true})
	subscribe, err := runtime.TranslateCommand(agentproto.Command{
		Kind: agentproto.CommandThreadSubscribe, CommandID: "sub1",
		Target: agentproto.Target{ThreadID: "desktop-thread"},
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if len(subscribe.Phases) != 1 {
		t.Fatalf("expected one subscription phase, got %#v", subscribe)
	}
	resume := sharedRuntimeSingleFrame(t, subscribe.Phases[0].OutboundToChild, "thread/resume")
	resumed, err := runtime.ObserveServer([]byte(fmt.Sprintf(`{"id":%q,"result":{"thread":{"id":"desktop-thread","cwd":"/repo","status":{"type":"idle"}}}}`, resume["id"])))
	if err != nil {
		t.Fatalf("observe resume response: %v", err)
	}
	list := sharedRuntimeSingleFrame(t, resumed.OutboundToChild, "thread/turns/list")
	ready, err := runtime.ObserveServer([]byte(fmt.Sprintf(`{"id":%q,"result":{"data":[]}}`, list["id"])))
	if err != nil {
		t.Fatalf("observe turn snapshot: %v", err)
	}
	if len(ready.Events) != 1 || ready.Events[0].Kind != agentproto.EventThreadSubscribed || ready.Events[0].Status != "ready" || ready.Events[0].ThreadID != "desktop-thread" {
		t.Fatalf("subscription never became ready: %#v", ready.Events)
	}

	err = runtime.PrepareChildRestart("restart1", agentproto.DefaultPromptDispatchPlanForExecutionThread("desktop-thread"), &agentproto.CodexResumePolicy{
		Mode: agentproto.CodexResumeApplyTargetProfile, ModelProviderID: "other-provider",
		ModelMode: agentproto.CodexThreadValueExplicit, Model: "other-model",
	})
	var problem agentproto.ErrorInfo
	if !errors.As(err, &problem) || problem.Code != "shared_app_server_restart_not_supported" || problem.CommandID != "restart1" {
		t.Fatalf("expected shared restart rejection before changing subscription, got %v", err)
	}
	frame, requestID, needsRestore, err := runtime.BuildChildRestartRestoreFrame("restart1")
	if err == nil {
		t.Fatalf("restore entry point must reject shared restart too, got %v", err)
	}
	if len(frame) != 0 || requestID != "" || needsRestore {
		t.Fatalf("rejected shared restart must not emit restore work: frame=%s request=%q needsRestore=%t", frame, requestID, needsRestore)
	}

	prompt, err := runtime.TranslateCommand(agentproto.Command{
		Kind: agentproto.CommandPromptSend, CommandID: "continue1",
		Origin: agentproto.Origin{Surface: "feishu-chat"},
		Target: agentproto.Target{ThreadID: "desktop-thread"},
		Prompt: agentproto.Prompt{Inputs: []agentproto.Input{{Type: agentproto.InputText, Text: "continue"}}},
	})
	if err != nil {
		t.Fatalf("rejected restart broke ready subscription: %v", err)
	}
	if len(prompt.Phases) != 1 {
		t.Fatalf("expected direct continuation phase, got %#v", prompt)
	}
	turn := sharedRuntimeSingleFrame(t, prompt.Phases[0].OutboundToChild, "turn/start")
	params, _ := turn["params"].(map[string]any)
	if params["threadId"] != "desktop-thread" {
		t.Fatalf("rejected restart retargeted original subscription: %#v", params)
	}
}

func sharedRuntimeSingleFrame(t *testing.T, frames [][]byte, method string) map[string]any {
	t.Helper()
	if len(frames) != 1 {
		t.Fatalf("expected only %s frame, got %q", method, frames)
	}
	var frame map[string]any
	if err := json.Unmarshal(frames[0], &frame); err != nil {
		t.Fatalf("decode %s: %v", method, err)
	}
	if frame["method"] != method || frame["id"] == nil || frame["id"] == "" {
		t.Fatalf("expected %s with request id, got %#v", method, frame)
	}
	return frame
}
