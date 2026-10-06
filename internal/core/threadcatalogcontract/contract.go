package threadcatalogcontract

import (
	"time"

	"github.com/YChange01/codex-feishu-link/internal/core/agentproto"
	"github.com/YChange01/codex-feishu-link/internal/core/state"
)

type PersistedThreadCatalog interface {
	RecentThreads(limit int) ([]state.ThreadRecord, error)
	RecentWorkspaces(limit int) (map[string]time.Time, error)
	ThreadByID(threadID string) (*state.ThreadRecord, error)
}

type BackendAwarePersistedThreadCatalog interface {
	RecentThreadsForBackend(agentproto.Backend, int) ([]state.ThreadRecord, error)
	RecentWorkspacesForBackend(agentproto.Backend, int) (map[string]time.Time, error)
	ThreadByIDForBackend(agentproto.Backend, string) (*state.ThreadRecord, error)
}
