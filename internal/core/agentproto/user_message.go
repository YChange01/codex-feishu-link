package agentproto

// RemoteUserMessageClientPrefix marks input submitted by a shared-desktop
// bridge. Codex echoes clientUserMessageId as userMessage.clientId. It is a
// display-origin marker, not an authorization credential.
const RemoteUserMessageClientPrefix = "codex-feishu-relay:"
