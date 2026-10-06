package daemon

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/YChange01/codex-feishu-link/internal/app/codexprofile"
	"github.com/YChange01/codex-feishu-link/internal/config"
	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
	relayruntime "github.com/YChange01/codex-feishu-link/internal/runtime"
)

func TestSharedCodexAppServerEnvHonorsConfigAndEnvironment(t *testing.T) {
	for _, tt := range []struct {
		name       string
		configured bool
		envValue   string
		want       string
	}{
		{"persisted on", true, "", "1"},
		{"persisted off", false, "", "0"},
		{"environment on", false, "true", "1"},
		{"environment off", true, "0", "0"},
		{"invalid environment falls back", true, "invalid", "1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := []string{"PATH=/test", config.SharedAppServerEnv + "=" + tt.envValue}
			original := append([]string{}, env...)
			got := applySharedCodexAppServerEnv(env, tt.configured)
			if envMap(got)[config.SharedAppServerEnv] != tt.want || envMap(got)["PATH"] != "/test" {
				t.Fatalf("incorrect shared launcher environment: %#v", got)
			}
			if !reflect.DeepEqual(env, original) {
				t.Fatal("shared environment changed caller's slice")
			}
		})
	}
}

func TestSharedCodexNativeProbeUsesDesktopConnectionAndBlocksIsolatedProfiles(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.json")
	cfg := config.DefaultAppConfig()
	cfg.Wrapper.SharedAppServer = true
	record, err := config.PrepareCodexAPIProfileCreate(nil, config.CodexAPIProfileInput{
		Name: "API", BaseURL: "https://api.example/v1", APIKey: "test-key", Model: "test-model", ReasoningEffort: "high",
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg.Codex.Profiles = []config.CodexAPIProfileRecord{record}
	if err := config.WriteAppConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	app := New(":0", ":0", nil, agentproto.ServerIdentity{})
	baseEnv := applySharedCodexAppServerEnv([]string{"PATH=/test"}, cfg.Wrapper.SharedAppServer)
	app.SetHeadlessRuntime(HeadlessRuntimeConfig{
		CodexRealBinary: "desktop-link", ConfigPath: configPath, BaseEnv: baseEnv,
		Paths: relayruntime.Paths{StateDir: filepath.Join(root, "state")},
	})
	app.ConfigureAdmin(AdminRuntimeOptions{ConfigPath: configPath})
	app.runCodexCapabilityPreflight = func(context.Context, codexprofile.CapabilityPreflightOptions) (codexprofile.CapabilityPreflightObservation, error) {
		t.Fatal("shared mode must not run an isolated preflight or create its ephemeral thread")
		return codexprofile.CapabilityPreflightObservation{}, nil
	}
	nativeCalls := 0
	app.runCodexNativeConfigProbe = func(_ context.Context, options codexprofile.NativeConfigProbeOptions) (codexprofile.NativeConfigObservation, error) {
		nativeCalls++
		if options.BinaryPath != "desktop-link" || envMap(options.Env)[config.SharedAppServerEnv] != "1" {
			t.Fatalf("native probe did not receive desktop launcher transport: %#v", options)
		}
		return codexprofile.NativeConfigObservation{ModelProviderID: "desktop-provider", ModelEndpoint: "https://desktop.example/v1"}, nil
	}
	app.runCodexOAuthProbe = func(context.Context, codexprofile.OAuthProbeOptions) (codexprofile.OAuthProbeObservation, error) {
		t.Fatal("shared mode must not launch the independent OAuth probe")
		return codexprofile.OAuthProbeObservation{}, nil
	}
	app.ensureCodexRuntimeCapability(context.Background())
	app.ensureCodexNativeConnectionEvidence(context.Background())
	if nativeCalls != 2 || app.effectiveCodexRuntimeCapabilitySetLocked() != codexprofile.CodexSharedNativeCapabilitySetV1 {
		t.Fatalf("shared native capability not proven: calls=%d state=%#v", nativeCalls, app.codexRuntimeCapability)
	}
	app.mu.Lock()
	task, ok := app.beginCodexOAuthProbeLocked(false)
	app.mu.Unlock()
	if !ok {
		t.Fatal("OAuth unavailability was not materialized")
	}
	if err := app.runCodexOAuthProbeTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	env, args, err := app.applyCodexHeadlessProfileConfig(baseEnv, []string{"app-server"}, agentproto.BackendCodex, config.CodexNativeProfileID)
	if err != nil {
		t.Fatalf("native shared launch blocked: %v", err)
	}
	if !reflect.DeepEqual(args, []string{"app-server"}) || !sharedCodexAppServerEnabled(env) {
		t.Fatalf("native shared launch gained overrides or lost marker: env=%#v args=%#v", env, args)
	}
	if app.codexNativeConnection.evidence.ModelProviderID != "desktop-provider" {
		t.Fatal("native evidence did not reflect the shared desktop configuration")
	}
	for _, profileID := range []string{record.ID, config.CodexOAuthProfileID} {
		_, _, err := app.applyCodexHeadlessProfileConfig(baseEnv, []string{"app-server"}, agentproto.BackendCodex, profileID)
		if codexprofile.RuntimeErrorCode(err) != codexprofile.ErrorCodexCapabilityUnsupported {
			t.Fatalf("shared profile %s must reject isolated launch: %v", profileID, err)
		}
	}
	for _, profile := range app.materializeCodexProfileSummariesLocked(cfg) {
		if profile.ID == state.NativeCodexProfileID {
			if !profile.Available {
				t.Fatal("native profile should remain selectable")
			}
		} else if profile.Available || profile.StatusCode != codexprofile.ErrorCodexCapabilityUnsupported {
			t.Fatalf("isolated profile remains selectable in shared mode: %#v", profile)
		}
	}
}

func TestSharedCodexNativeProbeFailureDoesNotClaimCapability(t *testing.T) {
	app := New(":0", ":0", nil, agentproto.ServerIdentity{})
	app.SetHeadlessRuntime(HeadlessRuntimeConfig{CodexRealBinary: "desktop-link", BaseEnv: []string{config.SharedAppServerEnv + "=1"}, Paths: relayruntime.Paths{StateDir: t.TempDir()}})
	app.runCodexNativeConfigProbe = func(context.Context, codexprofile.NativeConfigProbeOptions) (codexprofile.NativeConfigObservation, error) {
		return codexprofile.NativeConfigObservation{}, errors.New("desktop socket unavailable")
	}
	app.ensureCodexRuntimeCapability(context.Background())
	if app.effectiveCodexRuntimeCapabilitySetLocked() != "" || app.effectiveCodexRuntimeCapabilityErrorCodeLocked() != codexprofile.ErrorCodexProbeUnavailable {
		t.Fatalf("failed shared connection advertised capability: %#v", app.codexRuntimeCapability)
	}
}
