package hooks

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/nannyagent/nannyapi/internal/security"
	"github.com/nannyagent/nannyapi/internal/validators"
)

// App interface for PocketBase operations
type App interface {
	FindCollectionByNameOrId(collectionNameOrId string) (*core.Collection, error)
	FindRecordsByFilter(collection any, filter string, sort string, limit int, offset int, params ...dbx.Params) ([]*core.Record, error)
	Save(record core.Model) error
}

// OnUserCreate validates password when creating a new user
func OnUserCreate(app App) func(*core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		password := e.Record.GetString("password")
		if password == "" {
			return e.Next()
		}

		if err := validators.ValidatePasswordInput(password); err != nil {
			return fmt.Errorf("password validation failed: %w", err)
		}

		result := validators.ValidatePasswordRequirements(password)
		if !result.IsValid {
			return fmt.Errorf("password validation failed: %s", result.Errors[0])
		}

		security.TrackPasswordChange(app, e.Record.Id, password, true)
		return e.Next()
	}
}

// OnUserUpdate validates password and checks security constraints when updating user
func OnUserUpdate(app App) func(*core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		password := e.Record.GetString("password")
		if password == "" {
			return e.Next()
		}

		if e.Record.GetBool("oauth_signup") {
			return fmt.Errorf("OAuth users cannot set passwords")
		}

		if err := security.CheckAccountLockout(app, e.Record.Id); err != nil {
			return err
		}

		if err := security.CheckPasswordChangeFrequency(app, e.Record.Id); err != nil {
			return err
		}

		if err := validators.ValidatePasswordInput(password); err != nil {
			return fmt.Errorf("password validation failed: %w", err)
		}

		result := validators.ValidatePasswordRequirements(password)
		if !result.IsValid {
			return fmt.Errorf("password validation failed: %s", result.Errors[0])
		}

		if err := security.CheckPasswordReuse(app, e.Record.Id, password); err != nil {
			return err
		}

		security.TrackPasswordChange(app, e.Record.Id, password, true)
		return e.Next()
	}
}

// OnAuthWithPasswordRequest checks account lockout and tracks failures
func OnAuthWithPasswordRequest(app App) func(*core.RecordAuthWithPasswordRequestEvent) error {
	return func(e *core.RecordAuthWithPasswordRequestEvent) error {
		// Find user by identity first
		usersCollection, _ := app.FindCollectionByNameOrId("users")
		records, err := app.FindRecordsByFilter(usersCollection, "email = {:email}", "", 1, 0, map[string]any{"email": e.Identity})

		var userId string
		if err == nil && len(records) > 0 {
			userId = records[0].Id
			// Check if account is locked BEFORE auth
			if lockErr := security.CheckAccountLockout(app, userId); lockErr != nil {
				return lockErr
			}
		}

		// Call next (actual password verification)
		nextErr := e.Next()

		// If auth failed AND we found a user, track the failure
		if nextErr != nil && userId != "" {
			// Track failed attempt (this may lock the account)
			err := security.TrackFailedAuthAttempt(app, userId)
			if err != nil {
				return err
			}
		}

		return nextErr
	}
}

// OnAuthRequest checks account lockout before allowing auth
func OnAuthRequest(app App) func(*core.RecordAuthRequestEvent) error {
	return func(e *core.RecordAuthRequestEvent) error {
		// Check if account is locked
		if err := security.CheckAccountLockout(app, e.Record.Id); err != nil {
			return err
		}
		return e.Next()
	}
}

// OnAuthFailed tracks failed auth attempts and locks account after 5 failures
func OnAuthFailed(app App) func(*core.RecordAuthRequestEvent) error {
	return func(e *core.RecordAuthRequestEvent) error {
		// Track failed attempt and potentially lock account
		if err := security.TrackFailedAuthAttempt(app, e.Record.Id); err != nil {
			return err
		}
		return e.Next()
	}
}
