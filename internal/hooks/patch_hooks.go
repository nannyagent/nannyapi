package hooks

import (
	"github.com/nannyagent/nannyapi/internal/patches"
	"github.com/nannyagent/nannyapi/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterPatchHooks registers all patch management endpoints
func RegisterPatchHooks(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// POST /api/patches - Create patch operation
		e.Router.POST("/api/patches", func(c *core.RequestEvent) error {
			if err := utils.ExtractAuthFromHeader(c, app); err != nil {
				return err
			}

			return patches.HandlePatchOperations(app, c)
		})

		// GET /api/patches - List/get patch operations
		e.Router.GET("/api/patches", func(c *core.RequestEvent) error {
			if err := utils.ExtractAuthFromHeader(c, app); err != nil {
				return err
			}

			return patches.HandlePatchOperations(app, c)
		})

		// PATCH /api/patches - Update patch operation
		e.Router.PATCH("/api/patches", func(c *core.RequestEvent) error {
			if err := utils.ExtractAuthFromHeader(c, app); err != nil {
				return err
			}

			return patches.HandlePatchOperations(app, c)
		})

		return nil
	})
}
