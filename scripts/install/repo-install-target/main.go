package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/app/install"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "repo install target error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr *os.File) error {
	defaults, err := install.DetectPlatformDefaults()
	if err != nil {
		return err
	}

	flagSet := flag.NewFlagSet("repo-install-target", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	instanceID := flagSet.String("instance", "", "override install instance id")
	baseDir := flagSet.String("base-dir", "", "override install base dir")
	format := flagSet.String("format", "json", "output format: json or shell")
	allowUnbound := flagSet.Bool("allow-unbound", false, "allow falling back when the repo has no binding")
	if err := flagSet.Parse(args); err != nil {
		return err
	}

	info, err := install.ResolveRepoInstallTargetInfo(install.RepoInstallTargetOptions{
		InstanceID:      *instanceID,
		BaseDir:         *baseDir,
		FallbackBaseDir: defaults.BaseDir,
		GOOS:            defaults.GOOS,
		RequireBinding:  !*allowUnbound,
	})
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "", "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	case "shell":
		_, err := fmt.Fprint(stdout, shellAssignments(info))
		return err
	default:
		return fmt.Errorf("unsupported format %q", *format)
	}
}

func shellAssignments(info install.RepoInstallTargetInfo) string {
	values := []struct {
		key   string
		value string
	}{
		{"CODEX_FEISHU_RELAY_TARGET_REPO_ROOT", info.RepoRoot},
		{"CODEX_FEISHU_RELAY_TARGET_BINDING_PATH", info.BindingPath},
		{"CODEX_FEISHU_RELAY_TARGET_BINDING_SOURCE", info.BindingSource},
		{"CODEX_FEISHU_RELAY_TARGET_INSTANCE_ID", info.InstanceID},
		{"CODEX_FEISHU_RELAY_TARGET_BASE_DIR", info.BaseDir},
		{"CODEX_FEISHU_RELAY_TARGET_INSTALL_BIN_DIR", info.InstallBinDir},
		{"CODEX_FEISHU_RELAY_TARGET_CONFIG_PATH", info.ConfigPath},
		{"CODEX_FEISHU_RELAY_TARGET_CONFIG_EXISTS", shellBool(info.ConfigExists)},
		{"CODEX_FEISHU_RELAY_TARGET_STATE_PATH", info.StatePath},
		{"CODEX_FEISHU_RELAY_TARGET_STATE_EXISTS", shellBool(info.StateExists)},
		{"CODEX_FEISHU_RELAY_TARGET_SERVICE_NAME", info.ServiceName},
		{"CODEX_FEISHU_RELAY_TARGET_SERVICE_UNIT_PATH", info.ServiceUnitPath},
		{"CODEX_FEISHU_RELAY_TARGET_LOG_PATH", info.LogPath},
		{"CODEX_FEISHU_RELAY_TARGET_RAW_LOG_PATH", info.RawLogPath},
		{"CODEX_FEISHU_RELAY_TARGET_PID_PATH", info.PIDPath},
		{"CODEX_FEISHU_RELAY_TARGET_LOCAL_UPGRADE_ARTIFACT_PATH", info.LocalUpgradeArtifactPath},
		{"CODEX_FEISHU_RELAY_TARGET_CURRENT_VERSION", info.CurrentVersion},
		{"CODEX_FEISHU_RELAY_TARGET_CURRENT_BINARY_PATH", info.CurrentBinaryPath},
		{"CODEX_FEISHU_RELAY_TARGET_PENDING_UPGRADE_PHASE", info.PendingUpgradePhase},
		{"CODEX_FEISHU_RELAY_TARGET_RELAY_LISTEN_HOST", info.Relay.ListenHost},
		{"CODEX_FEISHU_RELAY_TARGET_RELAY_LISTEN_PORT", strconv.Itoa(info.Relay.ListenPort)},
		{"CODEX_FEISHU_RELAY_TARGET_RELAY_URL", info.Relay.URL},
		{"CODEX_FEISHU_RELAY_TARGET_RELAY_SERVER_URL", info.Relay.ServerURL},
		{"CODEX_FEISHU_RELAY_TARGET_ADMIN_LISTEN_HOST", info.Admin.ListenHost},
		{"CODEX_FEISHU_RELAY_TARGET_ADMIN_LISTEN_PORT", strconv.Itoa(info.Admin.ListenPort)},
		{"CODEX_FEISHU_RELAY_TARGET_ADMIN_URL", info.Admin.URL},
		{"CODEX_FEISHU_RELAY_TARGET_TOOL_LISTEN_HOST", info.Tool.ListenHost},
		{"CODEX_FEISHU_RELAY_TARGET_TOOL_LISTEN_PORT", strconv.Itoa(info.Tool.ListenPort)},
		{"CODEX_FEISHU_RELAY_TARGET_TOOL_URL", info.Tool.URL},
		{"CODEX_FEISHU_RELAY_TARGET_EXTERNAL_ACCESS_LISTEN_HOST", info.ExternalAccess.ListenHost},
		{"CODEX_FEISHU_RELAY_TARGET_EXTERNAL_ACCESS_LISTEN_PORT", strconv.Itoa(info.ExternalAccess.ListenPort)},
		{"CODEX_FEISHU_RELAY_TARGET_EXTERNAL_ACCESS_URL", info.ExternalAccess.URL},
		{"CODEX_FEISHU_RELAY_TARGET_PPROF_ENABLED", shellBool(info.Pprof.Enabled)},
		{"CODEX_FEISHU_RELAY_TARGET_PPROF_LISTEN_HOST", info.Pprof.ListenHost},
		{"CODEX_FEISHU_RELAY_TARGET_PPROF_LISTEN_PORT", strconv.Itoa(info.Pprof.ListenPort)},
		{"CODEX_FEISHU_RELAY_TARGET_PPROF_URL", info.Pprof.URL},
	}

	var b strings.Builder
	for _, item := range values {
		b.WriteString(item.key)
		b.WriteByte('=')
		b.WriteString(shellQuote(item.value))
		b.WriteByte('\n')
	}
	return b.String()
}

func shellBool(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
