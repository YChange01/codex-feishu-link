package control

func quotaCommandSpec() feishuCommandSpec {
	return feishuCommandSpec{
		definition: FeishuCommandDefinition{
			ID:               FeishuCommandQuota,
			GroupID:          FeishuCommandGroupCurrentWork,
			Title:            "剩余额度",
			CanonicalSlash:   "/quota",
			CanonicalMenuKey: "quota",
			ArgumentKind:     FeishuCommandArgumentNone,
			Description:      "查看当前 Codex 账户各限额窗口的剩余比例和重置时间。",
			ShowInHelp:       true,
			ShowInMenu:       true,
		},
		textExact: []feishuCommandMatch{{alias: "/quota", action: Action{Kind: ActionQuotaCommand}}},
		menuExact: []feishuCommandMatch{{alias: "quota", action: Action{Kind: ActionQuotaCommand}}},
	}
}
