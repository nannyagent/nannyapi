package hooks

import (
	"github.com/nannyagent/nannyapi/internal/patches"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterPatchHooks registers all patch management endpoints
func RegisterPatchHooks(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Helper to wrap handler with auth middleware
		withAuth := func(handler func(*core.RequestEvent) error) func(*core.RequestEvent) error {
			return LoadAuthContext(app)(RequireAuth()(handler))
		}

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

		// Agent endpoints
		e.Router.POST("/api/patches/{id}/result", withAuth(func(c *core.RequestEvent) error {
			return patches.HandlePatchResult(app, c)
		}))

		e.Router.GET("/api/scripts/{id}/validate", withAuth(func(c *core.RequestEvent) error {
			return patches.HandleValidateScript(app, c)
		}))

		return e.Next()
	})
}
