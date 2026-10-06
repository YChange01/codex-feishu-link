package control

var feishuCommandSpecsRuntime = []feishuCommandSpec{
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandStatus,
			GroupID:          FeishuCommandGroupCurrentWork,
			Title:            "当前状态",
			CanonicalSlash:   "/status",
			CanonicalMenuKey: "status",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "查看当前模式、接管对象类型、输入目标和飞书侧临时覆盖。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/status", action: Action{Kind: ActionStatus}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "status", action: Action{Kind: ActionStatus}},
		},
	},
	quotaCommandSpec(),
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandMode,
			GroupID:          FeishuCommandGroupSendSettings,
			Title:            "切换模式",
			CanonicalSlash:   "/mode",
			CanonicalMenuKey: "mode",
			ArgumentKind:     FeishuCommandArgumentChoice,
			ArgumentFormHint: "codex",
			ArgumentFormNote: "输入 codex / claude / opencode / vscode；`normal` 仍兼容为 `codex`。",
			ArgumentSubmit:   "切换",
			Description:      "查看当前模式；bare `/mode` 会返回 codex / claude / opencode / vscode 切换卡片。",
			Examples:         []string{"/mode codex", "/mode claude", "/mode opencode", "/mode vscode", "/mode normal"},
			Options: []FeishuCommandOption{
				commandOption("/mode", "mode", "codex", "codex", "切换到 headless 的 Codex 模式。"),
				commandOption("/mode", "mode", "claude", "claude", "切换到 headless 的 Claude 模式。"),
				commandOption("/mode", "mode", "opencode", "opencode", "切换到 headless 的 OpenCode 模式。"),
				commandOption("/mode", "mode", "vscode", "vscode", "切换到 vscode 模式。"),
			},
			ShowInHelp: true,
			ShowInMenu: true,
		},
		textPrefixes: []feishuCommandPrefixMatch{
			{alias: "/mode", kind: ActionModeCommand},
		},
		menuExact: []feishuCommandMatch{
			{alias: "mode", action: Action{Kind: ActionModeCommand, Text: "/mode"}},
		},
		menuDynamic: []feishuCommandDynamicMenuMatch{
			{prefix: "mode_", kind: ActionModeCommand, parseArgument: normalizeModeMenuArgument},
			{prefix: "mode-", kind: ActionModeCommand, parseArgument: normalizeModeMenuArgument},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandAutoWhip,
			GroupID:          FeishuCommandGroupCommonTools,
			Title:            "AutoWhip",
			CanonicalSlash:   "/autowhip",
			CanonicalMenuKey: "autowhip",
			ArgumentKind:     FeishuCommandArgumentChoice,
			ArgumentFormHint: "on",
			ArgumentFormNote: "输入 on 或 off。",
			ArgumentSubmit:   "应用",
			Description:      "查看当前 autowhip 状态；bare `/autowhip` 会返回 on / off 切换卡片。",
			Examples:         []string{"/autowhip on", "/autowhip off"},
			Options: []FeishuCommandOption{
				commandOption("/autowhip", "autowhip", "on", "on", "开启当前飞书会话的 autowhip。"),
				commandOption("/autowhip", "autowhip", "off", "off", "关闭当前飞书会话的 autowhip。"),
			},
			ShowInHelp: true,
			ShowInMenu: true,
		},
		textPrefixes: []feishuCommandPrefixMatch{
			{alias: "/autowhip", kind: ActionAutoWhipCommand},
		},
		menuExact: []feishuCommandMatch{
			{alias: "autowhip", action: Action{Kind: ActionAutoWhipCommand, Text: "/autowhip"}},
		},
		menuDynamic: []feishuCommandDynamicMenuMatch{
			{prefix: "autowhip_", kind: ActionAutoWhipCommand, parseArgument: normalizeAutoWhipMenuArgument},
			{prefix: "autowhip-", kind: ActionAutoWhipCommand, parseArgument: normalizeAutoWhipMenuArgument},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandAutoContinue,
			GroupID:          FeishuCommandGroupSendSettings,
			Title:            "自动继续",
			CanonicalSlash:   "/autocontinue",
			CanonicalMenuKey: "autocontinue",
			ArgumentKind:     FeishuCommandArgumentChoice,
			ArgumentFormHint: "on",
			ArgumentFormNote: "输入 on 或 off。",
			ArgumentSubmit:   "应用",
			Description:      "查看当前自动继续状态；只处理上游可重试失败，不影响 AutoWhip。",
			Examples:         []string{"/autocontinue on", "/autocontinue off"},
			Options: []FeishuCommandOption{
				commandOption("/autocontinue", "autocontinue", "on", "on", "开启当前飞书会话的自动继续。"),
				commandOption("/autocontinue", "autocontinue", "off", "off", "关闭当前飞书会话的自动继续。"),
			},
			ShowInHelp: true,
			ShowInMenu: true,
		},
		textPrefixes: []feishuCommandPrefixMatch{
			{alias: "/autocontinue", kind: ActionAutoContinueCommand},
		},
		menuExact: []feishuCommandMatch{
			{alias: "autocontinue", action: Action{Kind: ActionAutoContinueCommand, Text: "/autocontinue"}},
		},
		menuDynamic: []feishuCommandDynamicMenuMatch{
			{prefix: "autocontinue_", kind: ActionAutoContinueCommand, parseArgument: normalizeAutoContinueMenuArgument},
			{prefix: "autocontinue-", kind: ActionAutoContinueCommand, parseArgument: normalizeAutoContinueMenuArgument},
		},
	},
}
