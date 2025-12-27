package hooks

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// LoadAuthContext returns a middleware that loads the auth record from the Authorization header
func LoadAuthContext(app core.App) func(next func(*core.RequestEvent) error) func(*core.RequestEvent) error {
	return func(next func(*core.RequestEvent) error) func(*core.RequestEvent) error {
		return func(e *core.RequestEvent) error {
			token := e.Request.Header.Get("Authorization")
			if token == "" {
				return next(e)
			}

			// Remove "Bearer " prefix if present
			if len(token) > 7 && strings.ToLower(token[:7]) == "bearer " {
				token = token[7:]
			}

			record, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
			if err != nil {
				fmt.Println("FindAuthRecordByToken error:", err)
			}
			if record != nil {
				e.Set("authRecord", record)
			}

			return next(e)
		}
	}
}

// RequireAuth returns a middleware that requires authentication
func RequireAuth() func(next func(*core.RequestEvent) error) func(*core.RequestEvent) error {
	return func(next func(*core.RequestEvent) error) func(*core.RequestEvent) error {
		return func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}
			return next(e)
		}
	}
}

// RequireAuthCollection returns a middleware that requires authentication for a specific collection
func RequireAuthCollection(collectionName string) func(next func(*core.RequestEvent) error) func(*core.RequestEvent) error {
	return func(next func(*core.RequestEvent) error) func(*core.RequestEvent) error {
		return func(e *core.RequestEvent) error {
			if e.Auth == nil || e.Auth.Collection().Name != collectionName {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return next(e)
		}
	}
}
