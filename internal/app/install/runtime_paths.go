package install

import (
	"path/filepath"
	"strings"

	relayruntime "github.com/YChange01/codex-feishu-link/internal/runtime"
)

func RuntimePathsForState(state InstallState) relayruntime.Paths {
	layout := installLayoutForInstance(state.BaseDir, state.InstanceID)
	logsDir := filepath.Join(layout.StateDir, "logs")
	stateDir := filepath.Join(layout.StateHome, productName)
	return relayruntime.Paths{
		ConfigDir:        layout.ConfigDir,
		ConfigFile:       filepath.Join(layout.ConfigDir, "config.json"),
		DataDir:          layout.StateDir,
		LogsDir:          logsDir,
		DaemonLogFile:    filepath.Join(logsDir, "codex-feishu-relay-relayd.log"),
		DaemonRawLogFile: filepath.Join(logsDir, "codex-feishu-relay-relayd-raw.ndjson"),
		StateDir:         stateDir,
		ManagerLockFile:  filepath.Join(stateDir, "relay-manager.lock"),
		DaemonLockFile:   filepath.Join(stateDir, "relayd.lock"),
		PIDFile:          filepath.Join(stateDir, "codex-feishu-relay-relayd.pid"),
		IdentityFile:     filepath.Join(stateDir, "codex-feishu-relay-relayd.identity.json"),
		ToolServiceFile:  filepath.Join(stateDir, "codex-feishu-relay-tool-service.json"),
	}
}

func RuntimeEnvForState(state InstallState) []string {
	layout := installLayoutForInstance(state.BaseDir, state.InstanceID)
	configPath := strings.TrimSpace(state.ConfigPath)
	if configPath == "" {
		configPath = filepath.Join(layout.ConfigDir, "config.json")
	}
	return []string{
		"CODEX_FEISHU_RELAY_CONFIG=" + configPath,
		"XDG_CONFIG_HOME=" + layout.ConfigHome,
		"XDG_DATA_HOME=" + layout.DataHome,
		"XDG_STATE_HOME=" + layout.StateHome,
	}
}
