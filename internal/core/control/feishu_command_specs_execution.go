package control

var feishuCommandSpecsExecution = []feishuCommandSpec{
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandStop,
			GroupID:          FeishuCommandGroupCurrentWork,
			Title:            "停止推理",
			CanonicalSlash:   "/stop",
			CanonicalMenuKey: "stop",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "中断当前执行，并丢弃飞书侧尚未发送的排队输入。",
			ShowInHelp:       true,
			ShowInMenu:       true,
			RecommendedMenu: &FeishuRecommendedMenu{
				Key:         "stop",
				Name:        "停止推理",
				Description: "中断当前执行，并丢弃飞书侧尚未发送的排队输入。",
			},
		},
		textExact: []feishuCommandMatch{
			{alias: "/stop", action: Action{Kind: ActionStop}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "stop", action: Action{Kind: ActionStop}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandCompact,
			GroupID:          FeishuCommandGroupCurrentWork,
			Title:            "压缩上下文",
			CanonicalSlash:   "/compact",
			CanonicalMenuKey: "compact",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "对当前已选择的会话手动触发一次上下文压缩；当前有其他任务时会直接拒绝。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{
			{alias: "/compact", action: Action{Kind: ActionCompact}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "compact", action: Action{Kind: ActionCompact}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandSteerAll,
			GroupID:          FeishuCommandGroupCurrentWork,
			Title:            "全部加速",
			CanonicalSlash:   "/steerall",
			CanonicalMenuKey: "steerall",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "把当前队列里可并入本轮执行的输入一次性并入当前 running turn。",
			ShowInHelp:       true,
			ShowInMenu:       true,
			RecommendedMenu: &FeishuRecommendedMenu{
				Key:         "steerall",
				Name:        "全部加速",
				Description: "把当前队列里可并入本轮执行的输入一次性并入当前 running turn。",
			},
		},
		textExact: []feishuCommandMatch{
			{alias: "/steerall", action: Action{Kind: ActionSteerAll}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "steerall", action: Action{Kind: ActionSteerAll}},
			{alias: "steer_all", action: Action{Kind: ActionSteerAll}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandNew,
			GroupID:          FeishuCommandGroupCurrentWork,
			Title:            "新建会话",
			CanonicalSlash:   "/new",
			CanonicalMenuKey: "new",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "仅 headless 模式可用：准备一个新会话，下一条消息会作为首条输入。",
			ShowInHelp:       true,
			ShowInMenu:       true,
			RecommendedMenu: &FeishuRecommendedMenu{
				Key:         "new",
				Name:        "新建会话",
				Description: "仅 headless 模式可用：准备一个新会话，下一条消息会作为首条输入。",
			},
		},
		textExact: []feishuCommandMatch{
			{alias: "/new", action: Action{Kind: ActionNewThread}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "new", action: Action{Kind: ActionNewThread}},
			{alias: "newthread", action: Action{Kind: ActionNewThread}},
		},
	},
	{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandHistory,
			GroupID:          FeishuCommandGroupCommonTools,
			Title:            "查看会话历史",
			CanonicalSlash:   "/history",
			CanonicalMenuKey: "history",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "查看当前输入目标 thread 的历史 turn 列表，并在卡片里继续查看某一轮的详情。",
			ShowInHelp:       true,
			ShowInMenu:       true,
			RecommendedMenu: &FeishuRecommendedMenu{
				Key:         "history",
				Name:        "查看会话历史",
				Description: "查看当前输入目标 thread 的历史 turn 列表，并可进入某一轮的详情。",
			},
		},
		textExact: []feishuCommandMatch{
			{alias: "/history", action: Action{Kind: ActionShowHistory, Text: "/history"}},
		},
		menuExact: []feishuCommandMatch{
			{alias: "history", action: Action{Kind: ActionShowHistory, Text: "/history"}},
		},
	},
	feishuPrimaryCommandSpec,
	feishuCoworkersCommandSpec,
	reviewCommandSpec(),
	sendFileCommandSpec(),
}
