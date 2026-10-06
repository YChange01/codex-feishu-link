package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/app/appserverargs"
)

const mcpPrefix = "mcp_servers.codex_feishu_relay."

// Only the wrapper's optional MCP publication overrides may be discarded. All
// other server launch settings fail closed instead of claiming to apply them to
// an already running daemon.
func sharedServerInvocation(args []string) (bool, error) {
	match, found := appserverargs.Find(args)
	if !found || match.Mode != appserverargs.ModeCodex {
		for _, arg := range args {
			if arg == "app-server" {
				return false, fmt.Errorf("unsupported options before app-server; shared desktop mode cannot apply server launch overrides")
			}
		}
		return false, nil
	}
	serverArgs := args[match.Index+1:]
	if len(serverArgs) > 0 {
		switch serverArgs[0] {
		case "generate-json-schema", "generate-ts", "help", "--help", "-h":
			return false, nil
		case "daemon", "proxy":
			return false, fmt.Errorf("app-server %s is not supported by this launcher; manage the desktop daemon explicitly with the real Codex CLI", serverArgs[0])
		}
	}
	overrides := append(append([]string{}, args[:match.Index]...), serverArgs...)
	seen := map[string]bool{}
	for i := 0; i < len(overrides); i++ {
		arg := overrides[i]
		var value string
		switch {
		case arg == "-c" || arg == "--config":
			i++
			if i >= len(overrides) {
				return false, fmt.Errorf("missing value for %s", arg)
			}
			value = overrides[i]
		case strings.HasPrefix(arg, "--config="):
			value = strings.TrimPrefix(arg, "--config=")
		default:
			return false, fmt.Errorf("unsupported app-server option %q; shared desktop mode uses the desktop daemon configuration", optionName(arg))
		}
		key, raw, ok := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		if !ok {
			return false, fmt.Errorf("malformed app-server config override; expected key=value")
		}
		if key != mcpPrefix+"url" && key != mcpPrefix+"bearer_token_env_var" {
			return false, fmt.Errorf("unsupported app-server config override %q; shared desktop mode cannot apply custom launch settings", key)
		}
		if seen[key] {
			return false, fmt.Errorf("duplicate Feishu MCP override %q", key)
		}
		seen[key] = true
		decoded, err := strconv.Unquote(strings.TrimSpace(raw))
		if err != nil {
			return false, fmt.Errorf("invalid Feishu MCP override %q", key)
		}
		if key == mcpPrefix+"bearer_token_env_var" {
			if decoded != "CODEX_FEISHU_RELAY_MCP_BEARER" {
				return false, fmt.Errorf("unsupported Feishu MCP bearer environment variable")
			}
		} else {
			u, err := url.Parse(decoded)
			if err != nil || u.Scheme != "http" || u.User != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost" && u.Hostname() != "::1") {
				return false, fmt.Errorf("only the wrapper's loopback Feishu MCP URL can be omitted in shared desktop mode")
			}
		}
	}
	if len(seen) == 1 {
		return false, fmt.Errorf("incomplete automatic Feishu MCP overrides; expected both url and bearer_token_env_var")
	}
	return true, nil
}

func optionName(arg string) string {
	if !strings.HasPrefix(arg, "-") {
		return "<positional argument>"
	}
	name, _, _ := strings.Cut(arg, "=")
	return name
}

func validateSharedEnvironment(home string, getenv func(string) string) error {
	if getenv("CODEX_FEISHU_RELAY_SHARED_APP_SERVER") != "1" {
		return fmt.Errorf("desktop connection requires CODEX_FEISHU_RELAY_SHARED_APP_SERVER=1")
	}
	if profile := strings.TrimSpace(getenv("CODEX_FEISHU_RELAY_CODEX_PROFILE_ID")); profile != "" && profile != "cp_native" {
		return fmt.Errorf("shared desktop mode requires the cp_native profile")
	}
	if codexHome := strings.TrimSpace(getenv("CODEX_HOME")); codexHome != "" && filepath.Clean(codexHome) != filepath.Join(home, ".codex") {
		return fmt.Errorf("custom CODEX_HOME is not supported by this local desktop launcher")
	}
	return nil
}
