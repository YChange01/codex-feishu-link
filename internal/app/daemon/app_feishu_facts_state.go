package daemon

import (
	"github.com/YChange01/codex-feishu-link/internal/app/daemon/feishufacts"
)

func (a *App) configureFeishuFactsStateLocked(stateDir string) {
	path := feishufacts.StatePath(stateDir)
	a.feishuFactsState.persistedStoreRuntimeState = loadPersistedStore("Feishu bot facts", path, feishufacts.LoadStore)
}
