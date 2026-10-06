package control

var feishuCommandSpecsWorkspace = []feishuCommandSpec{
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandWorkspace,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "工作区与会话",
			CanonicalSlash:   "/workspace",
			CanonicalMenuKey: "workspace",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "打开 headless 模式的工作区与会话主页，里面提供切换、从目录新建、从 GIT URL 新建、从 Worktree 新建、解除接管。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/workspace", action: Action{Kind: ActionWorkspaceRoot}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "workspace", action: Action{Kind: ActionWorkspaceRoot}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandWorkspaceList,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "切换工作区与会话",
			CanonicalSlash:   "/workspace list",
			CanonicalMenuKey: "workspace_list",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "打开工作区与会话切换卡；headless 模式下 `/list`、`/use`、`/useall` 都会汇合到这里。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/workspace list", action: Action{Kind: ActionWorkspaceList}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "workspace_list", action: Action{Kind: ActionWorkspaceList}},
			{alias: "workspacelist", action: Action{Kind: ActionWorkspaceList}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandWorkspaceNew,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "新建工作区",
			CanonicalSlash:   "/workspace new",
			CanonicalMenuKey: "workspace_new",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "打开新建工作区入口页，可继续选择从目录新建、从 GIT URL 新建或从 Worktree 新建。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/workspace new", action: Action{Kind: ActionWorkspaceNew}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "workspace_new", action: Action{Kind: ActionWorkspaceNew}},
			{alias: "workspacenew", action: Action{Kind: ActionWorkspaceNew}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandWorkspaceNewDir,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "从目录新建",
			CanonicalSlash:   "/workspace new dir",
			CanonicalMenuKey: "workspace_new_dir",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "直接打开从本地目录新建工作区卡片。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/workspace new dir", action: Action{Kind: ActionWorkspaceNewDir}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "workspace_new_dir", action: Action{Kind: ActionWorkspaceNewDir}},
			{alias: "workspacenewdir", action: Action{Kind: ActionWorkspaceNewDir}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandWorkspaceNewGit,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "从 GIT URL 新建",
			CanonicalSlash:   "/workspace new git",
			CanonicalMenuKey: "workspace_new_git",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "直接打开从 GIT URL 新建工作区卡片。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/workspace new git", action: Action{Kind: ActionWorkspaceNewGit}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "workspace_new_git", action: Action{Kind: ActionWorkspaceNewGit}},
			{alias: "workspacenewgit", action: Action{Kind: ActionWorkspaceNewGit}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandWorkspaceNewWorktree,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "从 Worktree 新建",
			CanonicalSlash:   "/workspace new worktree",
			CanonicalMenuKey: "workspace_new_worktree",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "基于一个已接入的 Git 工作区创建新的 worktree，并自动进入新会话。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/workspace new worktree", action: Action{Kind: ActionWorkspaceNewWorktree}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "workspace_new_worktree", action: Action{Kind: ActionWorkspaceNewWorktree}},
			{alias: "workspacenewworktree", action: Action{Kind: ActionWorkspaceNewWorktree}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandWorkspaceDetach,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "解除接管",
			CanonicalSlash:   "/workspace detach",
			CanonicalMenuKey: "workspace_detach",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "解除当前接管；headless 模式下 `/detach` 会汇合到这里。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/workspace detach", action: Action{Kind: ActionWorkspaceDetach}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "workspace_detach", action: Action{Kind: ActionWorkspaceDetach}},
			{alias: "workspacedetach", action: Action{Kind: ActionWorkspaceDetach}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandList,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "工作区与会话",
			CanonicalSlash:   "/list",
			CanonicalMenuKey: "list",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "打开工作区/会话目录；headless 模式走统一工作区/会话选择，vscode 模式列出可接管实例。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/list", action: Action{Kind: ActionListInstances}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "list", action: Action{Kind: ActionListInstances}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandUse,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "接管会话",
			CanonicalSlash:   "/use",
			CanonicalMenuKey: "use",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "展示最近会话；headless attached 时只看当前工作区并附带“显示全部”按钮，headless detached 时等同 `/useall` 的最近工作区总览，vscode detached 时需先 `/list`。",
			Examples:         []string{"/useall"},
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/use", action: Action{Kind: ActionShowThreads}},
			{alias: "/threads", action: Action{Kind: ActionShowThreads}},
			{alias: "/sessions", action: Action{Kind: ActionShowThreads}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "use", action: Action{Kind: ActionShowThreads}},
			{alias: "threads", action: Action{Kind: ActionShowThreads}},
			{alias: "sessions", action: Action{Kind: ActionShowThreads}},
			{alias: "showthreads", action: Action{Kind: ActionShowThreads}},
			{alias: "showsessions", action: Action{Kind: ActionShowThreads}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandUseAll,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "全部会话",
			CanonicalSlash:   "/useall",
			CanonicalMenuKey: "useall",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "展示跨工作区会话总览；headless 模式下默认先显示最近 5 个工作区并可卡片内展开全部，vscode attached 时仍只看当前实例，detached 时需先 `/list`。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/useall", action: Action{Kind: ActionShowAllThreads}},
			{alias: "/sessionsall", action: Action{Kind: ActionShowAllThreads}},
			{alias: "/sessions/all", action: Action{Kind: ActionShowAllThreads}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "useall", action: Action{Kind: ActionShowAllThreads}},
			{alias: "threadsall", action: Action{Kind: ActionShowAllThreads}},
			{alias: "sessionsall", action: Action{Kind: ActionShowAllThreads}},
			{alias: "showallthreads", action: Action{Kind: ActionShowAllThreads}},
			{alias: "showallsessions", action: Action{Kind: ActionShowAllThreads}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandDetach,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "解除接管",
			CanonicalSlash:   "/detach",
			CanonicalMenuKey: "detach",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "解除当前接管，停止把后续输入发送到当前实例。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/detach", action: Action{Kind: ActionDetach}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "detach", action: Action{Kind: ActionDetach}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandFollow,
			GroupID:          FeishuCommandGroupSwitchTarget,
			Title:            "跟随当前",
			CanonicalSlash:   "/follow",
			CanonicalMenuKey: "follow",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "仅 `vscode` 模式可用：跟随当前 VS Code 聚焦会话；headless 模式请改走 `/use`、`/new` 或 `/mode vscode`。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/follow", action: Action{Kind: ActionFollowLocal}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "follow", action: Action{Kind: ActionFollowLocal}},
		},
	},
	adminCommandSpec(),
	adminSubcommandSpec(),
}
