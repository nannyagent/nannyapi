package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// App interface for testability
type App interface {
	FindCollectionByNameOrId(collectionNameOrId string) (*core.Collection, error)
	FindRecordsByFilter(collection any, filter string, sort string, limit int, offset int, params ...dbx.Params) ([]*core.Record, error)
	Save(record core.Model) error
}

// HashPassword creates SHA-256 hash of password for reuse checking
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// CheckAccountLockout checks if account is locked
func CheckAccountLockout(app App, userId string) error {
	lockoutCollection, err := app.FindCollectionByNameOrId("account_lockout")
	if err != nil {
		return nil // Collection doesn't exist yet
	}

	// Get all lockouts for this user
	records, err := app.FindRecordsByFilter(lockoutCollection, "user_id = {:userId}", "", 1, 0, dbx.Params{
		"userId": userId,
	})

	if err != nil || len(records) == 0 {
		return nil
	}

	lockout := records[0]
	lockedUntilStr := lockout.GetString("locked_until")
	lockedUntil, err := time.Parse(time.RFC3339, lockedUntilStr)
	if err != nil {
		// Try parsing as datetime field instead
		lockedUntil = lockout.GetDateTime("locked_until").Time()
		if lockedUntil.IsZero() {
			return nil
		}
	}

	if time.Now().Before(lockedUntil) {
		return fmt.Errorf("account is locked until %s due to too many failed attempts", lockedUntil.Format(time.RFC3339))
	}

	return nil
}

// CheckPasswordChangeFrequency enforces max 5 password changes per 24h
func CheckPasswordChangeFrequency(app App, userId string) error {
	historyCollection, err := app.FindCollectionByNameOrId("password_change_history")
	if err != nil {
		return nil // Collection doesn't exist yet
	}

	since := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	records, err := app.FindRecordsByFilter(historyCollection, "user_id = {:userId} && created >= {:since}", "-created", 10, 0, dbx.Params{
		"userId": userId,
		"since":  since,
	})

	if err == nil && len(records) >= 5 {
		return fmt.Errorf("too many password changes. Maximum 5 password changes allowed per 24 hours")
	}

	return nil
}

// CheckPasswordReuse prevents reusing passwords within 24 hours
func CheckPasswordReuse(app App, userId string, newPassword string) error {
	historyCollection, err := app.FindCollectionByNameOrId("password_change_history")
	if err != nil {
		return nil // Collection doesn't exist yet
	}

	passwordHash := HashPassword(newPassword)
	records, err := app.FindRecordsByFilter(historyCollection, "user_id = {:userId}", "-id", 100, 0, dbx.Params{
		"userId": userId,
	})

	if err == nil {
		cutoff := time.Now().Add(-24 * time.Hour)
		for _, record := range records {
			storedHash := record.GetString("password_hash")
			createdTime := record.GetDateTime("created").Time()

			if createdTime.After(cutoff) && storedHash == passwordHash {
				return fmt.Errorf("password was recently used. Choose a different password (24 hour history window)")
			}
		}
	}

	return nil
}

// TrackPasswordChange records password change in history
func TrackPasswordChange(app App, userId string, password string, success bool) {
	historyCollection, err := app.FindCollectionByNameOrId("password_change_history")
	if err == nil {
		rec := core.NewRecord(historyCollection)
		rec.Set("user_id", userId)
		rec.Set("password_hash", HashPassword(password))
		rec.Set("ip_address", "127.0.0.1")
		rec.Set("user_agent", "pocketbase")
		rec.Set("changed_by_agent", false)
		rec.Set("created", time.Now().Format(time.RFC3339))
		err = app.Save(rec)
		if err != nil {
			fmt.Printf("Warning: failed to save password change history: %v\n", err)
		}
	}

	// Track attempt
	attemptsCollection, err := app.FindCollectionByNameOrId("password_change_attempts")
	if err == nil {
		rec := core.NewRecord(attemptsCollection)
		rec.Set("user_id", userId)
		rec.Set("ip_address", "127.0.0.1")
		rec.Set("success", success)
		err = app.Save(rec)
		if err != nil {
			fmt.Printf("Warning: failed to save password change attempt: %v\n", err)
		}
	}
}

// TrackFailedAuthAttempt tracks failed login attempts and locks account after 5 failures
func TrackFailedAuthAttempt(app App, userId string) error {
	attemptsCollection, err := app.FindCollectionByNameOrId("failed_auth_attempts")
	if err != nil {
		return fmt.Errorf("collection not found: %v", err)
	}

	// Record failed attempt
	rec := core.NewRecord(attemptsCollection)
	rec.Set("user_id", userId)
	rec.Set("ip_address", "127.0.0.1")
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("failed to save attempt: %v", err)
	}

	// Get all failed attempts for this user
	records, err := app.FindRecordsByFilter(attemptsCollection, "user_id = {:userId}", "", 100, 0, dbx.Params{
		"userId": userId,
	})
	if err != nil {
		return fmt.Errorf("query error: %v", err)
	}

	// Filter in Go code for last 15 minutes
	var recentFailures []*core.Record
	sinceTime := time.Now().Add(-15 * time.Minute)
	for _, r := range records {
		// Check if created field exists and is valid
		createdField := r.GetDateTime("created")
		if createdField.IsZero() {
			// If no created timestamp, treat as recent (just created)
			recentFailures = append(recentFailures, r)
			continue
		}
		created := createdField.Time()
		if created.After(sinceTime) {
			recentFailures = append(recentFailures, r)
		}
	}

	if len(recentFailures) >= 5 {
		// Lock account for 30 minutes
		lockoutCollection, err := app.FindCollectionByNameOrId("account_lockout")
		if err == nil {
			lockout := core.NewRecord(lockoutCollection)
			lockout.Set("user_id", userId)
			lockout.Set("reason", "Too many failed login attempts")
			lockout.Set("locked_until", time.Now().Add(30*time.Minute).Format(time.RFC3339))
			lockout.Set("failed_attempts", float64(len(recentFailures))) // Try float64
			if saveErr := app.Save(lockout); saveErr != nil {
				// Log save error but still return lockout error
				fmt.Printf("Warning: failed to save lockout record: %v\n", saveErr)
			}

			return fmt.Errorf("account locked due to too many failed attempts. Please try again in 30 minutes")
		}
	}

	return nil
}
