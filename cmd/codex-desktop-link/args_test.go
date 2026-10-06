package main

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSharedInvocationRejectsUnappliedLaunchSettings(t *testing.T) {
	mcpURL := mcpPrefix + `url="http://127.0.0.1:9642/mcp?caller=instance-1"`
	mcpBearer := mcpPrefix + `bearer_token_env_var="CODEX_FEISHU_RELAY_MCP_BEARER"`
	tests := []struct {
		name             string
		args             []string
		shared, rejected bool
	}{
		{"default", []string{"app-server"}, true, false},
		{"automatic MCP", []string{"app-server", "-c", mcpURL, "-c", mcpBearer}, true, false},
		{"root automatic MCP", []string{"--config=" + mcpURL, "app-server", "--config", mcpBearer}, true, false},
		{"version", []string{"--version"}, false, false},
		{"help", []string{"app-server", "--help"}, false, false},
		{"schema", []string{"app-server", "generate-json-schema", "--out", "schema"}, false, false},
		{"daemon stop", []string{"app-server", "daemon", "stop"}, false, true},
		{"proxy", []string{"app-server", "proxy"}, false, true},
		{"custom security", []string{"app-server", "-c", `approval_policy="never"`}, false, true},
		{"root custom security", []string{"-c", `sandbox_mode="danger-full-access"`, "app-server"}, false, true},
		{"unknown root option", []string{"--enable", "feature", "app-server"}, false, true},
		{"custom listen", []string{"app-server", "--listen", "ws://127.0.0.1:9999"}, false, true},
		{"missing config", []string{"app-server", "-c"}, false, true},
		{"incomplete MCP", []string{"app-server", "-c", mcpURL}, false, true},
		{"duplicate MCP", []string{"app-server", "-c", mcpURL, "-c", mcpURL}, false, true},
		{"custom bearer", []string{"app-server", "-c", mcpURL, "-c", mcpPrefix + `bearer_token_env_var="OTHER_SECRET"`}, false, true},
		{"nonlocal MCP", []string{"app-server", "-c", mcpPrefix + `url="https://example.com/mcp"`, "-c", mcpBearer}, false, true},
		{"additional TOML", []string{"app-server", "-c", mcpURL + "\napproval_policy=\"never\"", "-c", mcpBearer}, false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shared, err := sharedServerInvocation(test.args)
			if (err != nil) != test.rejected || shared != test.shared {
				t.Fatalf("shared=%t err=%v", shared, err)
			}
		})
	}
}

func TestSharedEnvironmentRequiresOptInAndNativeProfile(t *testing.T) {
	home := t.TempDir()
	for _, test := range []struct {
		name, marker, profile, codexHome string
		wantErr                          bool
	}{
		{"missing opt in", "", "cp_native", "", true},
		{"native", "1", "cp_native", "", false},
		{"native default", "1", "", "", false},
		{"native home", "1", "cp_native", filepath.Join(home, ".codex"), false},
		{"API profile", "1", "cp_custom", "", true},
		{"other home", "1", "cp_native", filepath.Join(home, "other"), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			env := map[string]string{"CODEX_FEISHU_RELAY_SHARED_APP_SERVER": test.marker, "CODEX_FEISHU_RELAY_CODEX_PROFILE_ID": test.profile, "CODEX_HOME": test.codexHome}
			err := validateSharedEnvironment(home, func(key string) string { return env[key] })
			if (err != nil) != test.wantErr {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestNonServerCommandsPassThroughWithoutConnecting(t *testing.T) {
	args := []string{"--version"}
	var out, stderr strings.Builder
	code := run(context.Background(), args, io.NopCloser(strings.NewReader("")), nopWriteCloser{&out}, &stderr,
		func() (string, error) { t.Fatal("must not resolve daemon socket"); return "", nil },
		func(string) string { return "" },
		func(_ context.Context, got []string, _ io.Reader, stdout, _ io.Writer) error {
			if !reflect.DeepEqual(args, got) {
				t.Fatalf("args=%q", got)
			}
			_, err := io.WriteString(stdout, "codex-cli test\n")
			return err
		})
	if code != 0 || out.String() != "codex-cli test\n" || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
	}
}

func TestServerInvocationNeverFallsBackToStartingCodex(t *testing.T) {
	var out, stderr strings.Builder
	code := run(context.Background(), []string{"app-server"}, io.NopCloser(strings.NewReader("")), nopWriteCloser{&out}, &stderr,
		func() (string, error) { return "", errors.New("home unavailable") },
		func(string) string { return "1" },
		func(context.Context, []string, io.Reader, io.Writer, io.Writer) error {
			t.Fatal("must not start Codex")
			return nil
		})
	if code == 0 || !strings.Contains(stderr.String(), "home unavailable") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}
