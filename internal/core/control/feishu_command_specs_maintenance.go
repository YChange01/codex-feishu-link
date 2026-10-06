package control

var feishuCommandSpecsMaintenance = []feishuCommandSpec{
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandHelp,
			GroupID:          FeishuCommandGroupMaintenance,
			Title:            "命令帮助",
			CanonicalSlash:   "/help",
			CanonicalMenuKey: "help",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "查看 canonical slash command 列表和示例。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/help", action: Action{Kind: ActionShowCommandHelp}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "help", action: Action{Kind: ActionShowCommandHelp}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandMenu,
			GroupID:          FeishuCommandGroupMaintenance,
			Title:            "命令菜单",
			CanonicalSlash:   "/menu",
			CanonicalMenuKey: "menu",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "打开阶段感知的命令菜单首页。",
			ShowInHelp:       true,
			ShowInMenu:       false,
			RecommendedMenu: &FeishuRecommendedMenu{
				Key:         "menu",
				Name:        "命令菜单",
				Description: "打开阶段感知的命令菜单首页。",
			},
		},
		textPrefixes: []feishuCommandPrefixMatch{
			{alias: "/menu", kind: ActionShowCommandMenu},
		},
		menuExact: []feishuCommandMatch{
			{alias: "menu", action: Action{Kind: ActionShowCommandMenu}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandCron,
			GroupID:          FeishuCommandGroupCommonTools,
			Title:            "定时任务",
			CanonicalSlash:   "/cron",
			CanonicalMenuKey: "cron",
			ArgumentKind:     FeishuCommandArgumentText,
			ArgumentFormHint: "reload",
			ArgumentFormNote: "例如 reload。",
			ArgumentSubmit:   "执行",
			Description:      "打开当前服务实例专属的定时任务多维表格，或用 `/cron reload` 重新加载任务配置。",
			Examples:         []string{"/cron", "/cron reload"},
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textPrefixes: []feishuCommandPrefixMatch{
			{alias: "/cron", kind: ActionCronCommand},
		},
		menuExact: []feishuCommandMatch{
			{alias: "cron", action: Action{Kind: ActionCronCommand, Text: "/cron"}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandMCPOAuth,
			GroupID:          FeishuCommandGroupCommonTools,
			Title:            "MCP 服务认证",
			CanonicalSlash:   "/mcpoauth",
			CanonicalMenuKey: "mcpoauth",
			ArgumentKind:     FeishuCommandArgumentText,
			ArgumentFormHint: "github",
			ArgumentFormNote: "输入需要认证的 MCP 服务名。",
			ArgumentSubmit:   "生成授权链接",
			Description:      "对当前接管实例发起某个 MCP 服务的 OAuth 认证。",
			Examples:         []string{"/mcpoauth github"},
			ShowInHelp:       true,
			ShowInMenu:       false,
		},
		textPrefixes: []feishuCommandPrefixMatch{
			{alias: "/mcpoauth", kind: ActionMCPOAuthCommand},
			{alias: "/mcp-oauth", kind: ActionMCPOAuthCommand},
		},
		menuExact: []feishuCommandMatch{
			{alias: "mcpoauth", action: Action{Kind: ActionMCPOAuthCommand, Text: "/mcpoauth"}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandUpgrade,
			GroupID:          FeishuCommandGroupMaintenance,
			Title:            "升级系统",
			CanonicalSlash:   "/upgrade",
			CanonicalMenuKey: "upgrade",
			ArgumentKind:     FeishuCommandArgumentChoice,
			ArgumentFormHint: "latest",
			ArgumentSubmit:   "执行",
			Description:      "查看升级状态或执行升级子命令。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textPrefixes: []feishuCommandPrefixMatch{
			{alias: "/upgrade", kind: ActionUpgradeCommand},
		},
		menuExact: []feishuCommandMatch{
			{alias: "upgrade", action: Action{Kind: ActionUpgradeCommand, Text: "/upgrade"}},
		},
		menuDynamic: []feishuCommandDynamicMenuMatch{
			{prefix: "upgrade_", kind: ActionUpgradeCommand, parseArgument: normalizeUpgradeMenuArgument},
			{prefix: "upgrade-", kind: ActionUpgradeCommand, parseArgument: normalizeUpgradeMenuArgument},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandPatch,
			GroupID:          FeishuCommandGroupCommonTools,
			Title:            "修补当前会话",
			CanonicalSlash:   "/bendtomywill",
			CanonicalMenuKey: "patch",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "对当前会话最新一轮助手回复打开受控修补卡；只支持 headless 模式，且当前实例必须空闲。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textPrefixes: []feishuCommandPrefixMatch{
			{alias: "/bendtomywill", kind: ActionTurnPatchCommand},
		},
		menuExact: []feishuCommandMatch{
			{alias: "patch", action: Action{Kind: ActionTurnPatchCommand, Text: "/bendtomywill"}},
		},
		extraActionRoutes: []feishuCommandActionRoute{
			{kind: ActionTurnPatchRollback, title: "回滚最近一次修补", canonicalSlash: "/bendtomywill rollback"},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandDebug,
			GroupID:          FeishuCommandGroupMaintenance,
			Title:            "调试",
			CanonicalSlash:   "/debug",
			CanonicalMenuKey: "debug",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "查看调试入口；管理页相关功能已收口到 `/admin`。",
			Examples:         []string{"/debug"},
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textPrefixes: []feishuCommandPrefixMatch{
			{alias: "/debug", kind: ActionDebugCommand},
		},
		menuExact: []feishuCommandMatch{
			{alias: "debug", action: Action{Kind: ActionDebugCommand, Text: "/debug"}},
		},
	},
	vscodeMigrateCommandSpec(),
}
