package hooks

import (
	"github.com/nannyagent/nannyapi/internal/investigations"
	"github.com/nannyagent/nannyapi/internal/patches"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterInvestigationAndPatchHooks registers all investigation and patch management endpoints
func RegisterInvestigationAndPatchHooks(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Helper to wrap handler with auth middleware
		withAuth := func(handler func(*core.RequestEvent) error) func(*core.RequestEvent) error {
			return LoadAuthContext(app)(RequireAuth()(handler))
		}

		// Single endpoint for all investigation operations
		// POST /api/investigations - Create investigation
		// GET /api/investigations - List/get investigations
		// PATCH /api/investigations - Update investigation
		e.Router.POST("/api/investigations", withAuth(func(c *core.RequestEvent) error {
			return investigations.HandleInvestigations(app, c)
		}))

		e.Router.GET("/api/investigations", withAuth(func(c *core.RequestEvent) error {
			return investigations.HandleInvestigations(app, c)
		}))

		e.Router.PATCH("/api/investigations", withAuth(func(c *core.RequestEvent) error {
			return investigations.HandleInvestigations(app, c)
		}))

		// Single endpoint for all patch operations
		// POST /api/patches - Create patch operation
		// GET /api/patches - List/get patch operations
		// PATCH /api/patches - Update patch operation
		e.Router.POST("/api/patches", withAuth(func(c *core.RequestEvent) error {
			return patches.HandlePatchOperations(app, c)
		}))

		e.Router.GET("/api/patches", withAuth(func(c *core.RequestEvent) error {
			return patches.HandlePatchOperations(app, c)
		}))

		e.Router.PATCH("/api/patches", withAuth(func(c *core.RequestEvent) error {
			return patches.HandlePatchOperations(app, c)
		}))

		return e.Next()
	})
}
