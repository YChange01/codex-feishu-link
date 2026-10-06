# Local desktop Codex launcher

This launcher connects this Feishu installation to the existing desktop Codex
app-server. If its socket is unavailable, it starts the shared daemon via the
official `codex app-server daemon start` command, guarded by a process-shared
lock and a 30-second deadline. A live daemon is never restarted or stopped.
`app-server` uses
WebSocket messages over `~/.codex/app-server-control/app-server-control.sock`,
translating them to the JSONL stream expected by the Feishu wrapper.

Shared mode requires `CODEX_FEISHU_RELAY_SHARED_APP_SERVER=1` and the `cp_native`
profile (an empty profile means the native default). Custom `CODEX_HOME` and
server launch overrides are rejected. The wrapper's paired, optional Feishu MCP
URL and bearer-environment overrides are recognized but omitted with a stderr
notice: the existing desktop daemon retains its own configuration. No MCP server
is installed into that daemon by this launcher. Per-request JSON-RPC payloads are
passed through unchanged; thread selection, approvals, active-turn routing, and
request-level policy belong to the bridge above this transport.

Other Codex CLI commands and app-server schema/help tooling are forwarded to
`/usr/local/bin/codex`. App-server daemon management and raw proxy subcommands are
rejected. This binary is specific to the local Unix installation.

Each input line must be one JSON object. WebSocket text messages become compact
JSON objects followed by a newline; control frames are handled by the WebSocket
library. Both directions enforce a 64 MiB message limit. Writes have a 15-second
timeout. EOF, cancellation, or either transport error closes only this client's
connection and unblocks its stdio pumps. EOF terminates the connection immediately;
the wrapper must keep stdin open while awaiting responses.

```sh
go test -race ./cmd/codex-desktop-link -count=1
go build -o /path/to/codex-desktop-link ./cmd/codex-desktop-link
```

Tests use temporary Unix WebSocket servers and do not connect to the desktop
daemon or resume, start, or stop any real Codex thread.

## VS Code project recovery

`open-vscode --cwd <absolute-project> --thread <id>` asks the local
`extensions/codex-feishu-relay-desktop` companion extension to open the exact existing
conversation. Private requests live in `~/.codex/feishu-desktop/requests`.
An existing matching project window gets the first opportunity to claim a
request; otherwise the launcher opens a new VS Code project window. Successful
delivery requires an acknowledgement with the same request, project and thread.
The wait is bounded to 90 seconds and failures can be retried.

Install `codex-vscode-link` beside this binary and configure the companion's
trusted `codexRemoteDesktop.cliExecutable` setting to that launcher. The
companion connects the Codex extension to the shared daemon and reloads only
the matching VS Code window when its previous backend connection is stale.
No prompt is created as part of opening a conversation. The daemon integration
is opt-in through `CODEX_FEISHU_RELAY_DESKTOP_OPEN=1` alongside shared app-server mode.
VS Code opening currently targets the standard macOS installation path.
