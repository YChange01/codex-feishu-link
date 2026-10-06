package control

import "github.com/YChange01/codex-feishu-link/internal/core/upgradecontract"

func FeishuUpgradeCommandRunsImmediately(text string) bool {
	return upgradecontract.RunsImmediately(text)
}
