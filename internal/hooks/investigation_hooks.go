package hooks

import (
	"github.com/nannyagent/nannyapi/internal/investigations"
	"github.com/nannyagent/nannyapi/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterInvestigationHooks registers all investigation management endpoints
func RegisterInvestigationHooks(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// POST /api/investigations - Create investigation (portal-initiated)
		e.Router.POST("/api/investigations", func(c *core.RequestEvent) error {
			if err := utils.ExtractAuthFromHeader(c, app); err != nil {
				return err
			}

			return investigations.HandleInvestigations(app, c)
		})

		// GET /api/investigations - List/get investigations
		e.Router.GET("/api/investigations", func(c *core.RequestEvent) error {
			if err := utils.ExtractAuthFromHeader(c, app); err != nil {
				return err
			}

			return investigations.HandleInvestigations(app, c)
		})

		// PATCH /api/investigations - Update investigation
		e.Router.PATCH("/api/investigations", func(c *core.RequestEvent) error {
			if err := utils.ExtractAuthFromHeader(c, app); err != nil {
				return err
			}

			return investigations.HandleInvestigations(app, c)
		})

		return e.Next()
	})
}
