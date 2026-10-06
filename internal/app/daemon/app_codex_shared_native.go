package daemon

import (
	"context"
	"strconv"
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/app/codexprofile"
	"github.com/YChange01/codex-feishu-link/internal/config"
)

// Use the same environment-over-file precedence as wrapper.LoadWrapperConfig.
// Normalize it for the launcher, which intentionally accepts the explicit "1".
func applySharedCodexAppServerEnv(env []string, configured bool) []string {
	if value := strings.TrimSpace(envMap(env)[config.SharedAppServerEnv]); value != "" {
		if enabled, err := strconv.ParseBool(value); err == nil {
			configured = enabled
		}
	}
	value := "0"
	if configured {
		value = "1"
	}
	return config.UpsertEnvValue(env, config.SharedAppServerEnv, value)
}

func sharedCodexAppServerEnabled(env []string) bool {
	enabled, _ := strconv.ParseBool(strings.TrimSpace(envMap(env)[config.SharedAppServerEnv]))
	return enabled
}

func sharedCodexNativePreflight(probe func(context.Context, codexprofile.NativeConfigProbeOptions) (codexprofile.NativeConfigObservation, error)) func(context.Context, codexprofile.CapabilityPreflightOptions) (codexprofile.CapabilityPreflightObservation, error) {
	return func(ctx context.Context, options codexprofile.CapabilityPreflightOptions) (codexprofile.CapabilityPreflightObservation, error) {
		if probe == nil {
			return codexprofile.CapabilityPreflightObservation{}, &codexprofile.OAuthProbeError{Code: codexprofile.ErrorCodexProbeUnavailable, Stage: "shared_native_config_read"}
		}
		// initialize + config/read only; the independent-runtime preflight's
		// startup overrides and ephemeral thread/start do not apply to a daemon
		// already owned by the desktop client.
		_, err := probe(ctx, codexprofile.NativeConfigProbeOptions{BinaryPath: options.BinaryPath, Env: options.Env, Version: options.Version})
		if err != nil {
			return codexprofile.CapabilityPreflightObservation{}, err
		}
		return codexprofile.CapabilityPreflightObservation{CapabilitySet: codexprofile.CodexSharedNativeCapabilitySetV1}, nil
	}
}

func sharedCodexOAuthProbeUnsupported(context.Context, codexprofile.OAuthProbeOptions) (codexprofile.OAuthProbeObservation, error) {
	return codexprofile.OAuthProbeObservation{}, &codexprofile.OAuthProbeError{
		Code: codexprofile.ErrorCodexCapabilityUnsupported, Stage: "shared_daemon_profile_isolation_unsupported",
	}
}
