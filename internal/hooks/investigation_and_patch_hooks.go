package hooks

import (
	"net/http"

	"github.com/nannyagent/nannyapi/internal/investigations"
	"github.com/nannyagent/nannyapi/internal/patches"
	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterInvestigationAndPatchHooks registers all investigation and patch management endpoints
func RegisterInvestigationAndPatchHooks(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Single endpoint for all investigation operations
		// POST /api/investigations - Create investigation
		// GET /api/investigations - List/get investigations
		// PATCH /api/investigations - Update investigation
		e.Router.POST("/api/investigations", func(c *core.RequestEvent) error {
			// Extract auth from header
			authHeader := c.Request.Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token := authHeader[7:]
				record, _ := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
				if record != nil {
					c.Set("authRecord", record)
				}
			}

			authRecord := c.Get("authRecord")
			if authRecord == nil {
				return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
			}

			return investigations.HandleInvestigations(app, c)
		})

		e.Router.GET("/api/investigations", func(c *core.RequestEvent) error {
			authHeader := c.Request.Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token := authHeader[7:]
				record, _ := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
				if record != nil {
					c.Set("authRecord", record)
				}
			}

			authRecord := c.Get("authRecord")
			if authRecord == nil {
				return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
			}

			return investigations.HandleInvestigations(app, c)
		})

		e.Router.PATCH("/api/investigations", func(c *core.RequestEvent) error {
			authHeader := c.Request.Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token := authHeader[7:]
				record, _ := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
				if record != nil {
					c.Set("authRecord", record)
				}
			}

			authRecord := c.Get("authRecord")
			if authRecord == nil {
				return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
			}

			return investigations.HandleInvestigations(app, c)
		})

		// Single endpoint for all patch operations
		// POST /api/patches - Create patch operation
		// GET /api/patches - List/get patch operations
		// PATCH /api/patches - Update patch operation
		e.Router.POST("/api/patches", func(c *core.RequestEvent) error {
			authHeader := c.Request.Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token := authHeader[7:]
				record, _ := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
				if record != nil {
					c.Set("authRecord", record)
				}
			}

			authRecord := c.Get("authRecord")
			if authRecord == nil {
				return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
			}

			return patches.HandlePatchOperations(app, c)
		})

		e.Router.GET("/api/patches", func(c *core.RequestEvent) error {
			authHeader := c.Request.Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token := authHeader[7:]
				record, _ := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
				if record != nil {
					c.Set("authRecord", record)
				}
			}

			authRecord := c.Get("authRecord")
			if authRecord == nil {
				return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
			}

			return patches.HandlePatchOperations(app, c)
		})

		e.Router.PATCH("/api/patches", func(c *core.RequestEvent) error {
			authHeader := c.Request.Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token := authHeader[7:]
				record, _ := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
				if record != nil {
					c.Set("authRecord", record)
				}
			}

			authRecord := c.Get("authRecord")
			if authRecord == nil {
				return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
			}

			return patches.HandlePatchOperations(app, c)
		})

		return nil
	})
}
