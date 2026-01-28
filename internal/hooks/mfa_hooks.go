package hooks

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/nannyagent/nannyapi/internal/mfa"
)

// MFASessionKey is the key used to store MFA session info in the request context
const MFASessionKey = "mfa_session"

// MFAHooks provides MFA-related hooks for the authentication system
type MFAHooks struct {
	app core.App
}

// NewMFAHooks creates a new MFA hooks instance
func NewMFAHooks(app core.App) *MFAHooks {
	return &MFAHooks{app: app}
}

// OnAuthSuccess is called after successful primary authentication
// It checks if the user has MFA enabled and returns appropriate response
func (h *MFAHooks) OnAuthSuccess(e *core.RecordAuthRequestEvent) error {
	// Check if user has MFA enabled
	mfaEnabled := e.Record.GetBool("mfa_enabled")

	if !mfaEnabled {
		// No MFA required, continue with normal auth
		return e.Next()
	}

	// Find user's verified MFA factor
	factorsCollection, err := h.app.FindCollectionByNameOrId("mfa_factors")
	if err != nil {
		// If can't find factors collection, allow auth but log warning
		return e.Next()
	}

	factors, err := h.app.FindRecordsByFilter(
		factorsCollection,
		"user_id = {:userId} AND status = 'verified'",
		"",
		1,
		0,
		dbx.Params{"userId": e.Record.Id},
	)

	if err != nil || len(factors) == 0 {
		// No verified factors, MFA flag might be stale - allow auth
		e.Record.Set("mfa_enabled", false)
		h.app.Save(e.Record)
		return e.Next()
	}

	// User has MFA enabled - we need to return a partial auth response
	// The client must complete MFA verification before getting the full token

	// Create an MFA challenge automatically
	challengesCollection, err := h.app.FindCollectionByNameOrId("mfa_challenges")
	if err != nil {
		return apis.NewApiError(500, "MFA system error", err)
	}

	expiresAt := time.Now().Add(mfa.ChallengeExpiryMinutes * time.Minute)

	challengeRecord := core.NewRecord(challengesCollection)
	challengeRecord.Set("factor_id", factors[0].Id)
	challengeRecord.Set("status", "pending")
	challengeRecord.Set("expires_at", expiresAt)

	if err := h.app.Save(challengeRecord); err != nil {
		return apis.NewApiError(500, "Failed to create MFA challenge", err)
	}

	// Return a response indicating MFA is required
	// The token returned is a temporary token that only allows MFA verification
	return e.JSON(http.StatusOK, map[string]any{
		"mfa_required": true,
		"factor_id":    factors[0].Id,
		"challenge_id": challengeRecord.Id,
		"expires_at":   expiresAt,
		"message":      "MFA verification required. Please provide your TOTP code.",
		// Include a limited token that only allows MFA verification
		"token": e.Token,
		"record": map[string]any{
			"id":    e.Record.Id,
			"email": e.Record.Email(),
		},
	})
}

// RequireMFAMiddleware is a middleware that requires MFA verification for certain routes
func (h *MFAHooks) RequireMFAMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a placeholder for future implementation
		// In a full implementation, we'd check if the session has been MFA-verified
		next.ServeHTTP(w, r)
	})
}

// CheckMFARequired checks if the user has MFA enabled and needs to verify
func (h *MFAHooks) CheckMFARequired(userID string) (bool, string, error) {
	// Find user
	usersCollection, err := h.app.FindCollectionByNameOrId("users")
	if err != nil {
		return false, "", err
	}

	userRecords, err := h.app.FindRecordsByFilter(
		usersCollection,
		"id = {:userId}",
		"",
		1,
		0,
		dbx.Params{"userId": userID},
	)
	if err != nil || len(userRecords) == 0 {
		return false, "", fmt.Errorf("user not found")
	}

	userRecord := userRecords[0]
	mfaEnabled := userRecord.GetBool("mfa_enabled")

	if !mfaEnabled {
		return false, "", nil
	}

	// Find verified factor
	factorsCollection, err := h.app.FindCollectionByNameOrId("mfa_factors")
	if err != nil {
		return false, "", err
	}

	factors, err := h.app.FindRecordsByFilter(
		factorsCollection,
		"user_id = {:userId} AND status = 'verified'",
		"",
		1,
		0,
		dbx.Params{"userId": userID},
	)

	if err != nil || len(factors) == 0 {
		return false, "", nil
	}

	return true, factors[0].Id, nil
}

// VerifyMFAForLogin verifies MFA during the login process
func (h *MFAHooks) VerifyMFAForLogin(userID, factorID, challengeID, code string) error {
	// Find the challenge
	challengeRecord, err := h.app.FindRecordById("mfa_challenges", challengeID)
	if err != nil {
		return fmt.Errorf("challenge not found")
	}

	// Check challenge status
	if challengeRecord.GetString("status") != "pending" {
		return fmt.Errorf("challenge is not pending")
	}

	// Check challenge hasn't expired
	expiresAt := challengeRecord.GetDateTime("expires_at").Time()
	if time.Now().After(expiresAt) {
		challengeRecord.Set("status", "expired")
		h.app.Save(challengeRecord)
		return fmt.Errorf("challenge has expired")
	}

	// Find the factor
	factorRecord, err := h.app.FindRecordById("mfa_factors", factorID)
	if err != nil {
		return fmt.Errorf("factor not found")
	}

	// Verify ownership
	if factorRecord.GetString("user_id") != userID {
		return fmt.Errorf("not authorized")
	}

	// Verify the TOTP code
	secret := factorRecord.GetString("secret")
	valid, err := mfa.VerifyTOTP(secret, code)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// If TOTP failed, try backup code
	if !valid {
		valid, err = h.verifyBackupCode(userID, code)
		if err != nil {
			return fmt.Errorf("backup code verification failed: %w", err)
		}
	}

	if !valid {
		return fmt.Errorf("invalid verification code")
	}

	// Check for replay attack (only for TOTP)
	if len(code) == mfa.TOTPDigits {
		if h.isTokenUsed(factorID, code) {
			return fmt.Errorf("this code has already been used")
		}
		h.markTokenUsed(factorID, code)
	}

	// Update challenge status
	challengeRecord.Set("status", "verified")
	challengeRecord.Set("verified_at", time.Now())
	if err := h.app.Save(challengeRecord); err != nil {
		return fmt.Errorf("failed to update challenge: %w", err)
	}

	return nil
}

// Helper methods

func (h *MFAHooks) verifyBackupCode(userID, code string) (bool, error) {
	backupCodesCollection, err := h.app.FindCollectionByNameOrId("mfa_backup_codes")
	if err != nil {
		return false, err
	}

	codes, err := h.app.FindRecordsByFilter(
		backupCodesCollection,
		"user_id = {:userId} AND used = false",
		"",
		100,
		0,
		dbx.Params{"userId": userID},
	)
	if err != nil {
		return false, nil
	}

	for _, codeRecord := range codes {
		hash := codeRecord.GetString("code_hash")
		if mfa.VerifyBackupCode(code, hash) {
			// Mark as used
			codeRecord.Set("used", true)
			codeRecord.Set("used_at", time.Now())
			if err := h.app.Save(codeRecord); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}

func (h *MFAHooks) isTokenUsed(factorID, token string) bool {
	tokenHash := mfa.HashToken(token)

	usedTokensCollection, err := h.app.FindCollectionByNameOrId("mfa_used_tokens")
	if err != nil {
		return false
	}

	records, err := h.app.FindRecordsByFilter(
		usedTokensCollection,
		"factor_id = {:factorId} AND token_hash = {:tokenHash}",
		"",
		1,
		0,
		dbx.Params{"factorId": factorID, "tokenHash": tokenHash},
	)

	return err == nil && len(records) > 0
}

func (h *MFAHooks) markTokenUsed(factorID, token string) {
	tokenHash := mfa.HashToken(token)

	usedTokensCollection, err := h.app.FindCollectionByNameOrId("mfa_used_tokens")
	if err != nil {
		return
	}

	record := core.NewRecord(usedTokensCollection)
	record.Set("factor_id", factorID)
	record.Set("token_hash", tokenHash)
	record.Set("used_at", time.Now())

	h.app.Save(record)
}
