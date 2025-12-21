package utils

import (
	"net/http"

	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
)

// ExtractAuthFromHeader extracts and validates the Bearer token from Authorization header
// Sets authRecord in request context if valid
func ExtractAuthFromHeader(c *core.RequestEvent, app core.App) error {
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

	return nil
}

// GetAuthRecord retrieves the authenticated record from request context
// Returns nil if not authenticated
func GetAuthRecord(c *core.RequestEvent) *core.Record {
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return nil
	}
	return authRecord.(*core.Record)
}

// GetAuthUserID retrieves the user ID from authenticated record
// Returns empty string if not authenticated
func GetAuthUserID(c *core.RequestEvent) string {
	record := GetAuthRecord(c)
	if record == nil {
		return ""
	}
	return record.Id
}
