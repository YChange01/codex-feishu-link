package relayruntime

import "testing"

func TestNormalizeDetachedCommandPathStripsWindowsExtendedPrefix(t *testing.T) {
	tests := map[string]string{
		`\\?\C:\repo\bin\codex-feishu-relay.exe`: `C:\repo\bin\codex-feishu-relay.exe`,
		`//?/C:/repo/bin/codex-feishu-relay.exe`: `C:\repo\bin\codex-feishu-relay.exe`,
		`\\?\UNC\server\share\run.log`:           `\\server\share\run.log`,
	}
	for input, want := range tests {
		if got := normalizeDetachedCommandPath(input); got != want {
			t.Fatalf("normalizeDetachedCommandPath(%q) = %q, want %q", input, got, want)
		}
	}
}
