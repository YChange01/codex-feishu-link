package daemon

import (
	"context"
	"fmt"
	"strings"

	"github.com/YChange01/codex-feishu-link/internal/adapter/feishu"
	"github.com/YChange01/codex-feishu-link/internal/core/control"
)

func handleGatewayActionForTest(ctx context.Context, app *App, action control.Action) *feishu.ActionResult {
	return app.HandleGatewayAction(ctx, stampGatewayCardCommandActionForTest(app, action))
}

func applyGatewayActionForTest(ctx context.Context, app *App, action control.Action) {
	app.HandleAction(ctx, stampGatewayCardCommandActionForTest(app, action))
}

func stampGatewayCardCommandActionForTest(app *App, action control.Action) control.Action {
	if app == nil || action.Inbound == nil {
		return action
	}
	if strings.TrimSpace(action.Inbound.CardDaemonLifecycleID) == "" {
		return action
	}
	if !control.ActionRoutesThroughFeishuCommandCatalog(action) || control.HasStrictFeishuCommandCatalogProvenance(action) {
		return action
	}
	surfaceID := strings.TrimSpace(action.SurfaceSessionID)
	surface := app.service.Surface(surfaceID)
	if surface == nil {
		panic(fmt.Sprintf("missing surface for stamped gateway test action: %q", surfaceID))
	}
	resolved, ok := control.ResolveFeishuActionCatalog(control.CatalogContext{
		Backend:     app.service.SurfaceBackend(surfaceID),
		ProductMode: string(surface.ProductMode),
	}, action)
	if !ok {
		panic(fmt.Sprintf("failed to resolve command provenance for gateway test action: kind=%q text=%q", action.Kind, action.Text))
	}
	return resolved.Action
}
