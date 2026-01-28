package mfa

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/nannyagent/nannyapi/internal/types"
)

const (
	// IssuerName is the name shown in authenticator apps
	IssuerName = "NannyAPI"
)

// Handler manages MFA operations
type Handler struct {
	app core.App
}

// NewHandler creates a new MFA handler
func NewHandler(app core.App) *Handler {
	return &Handler{app: app}
}

// RegisterRoutes registers MFA API routes
func (h *Handler) RegisterRoutes(e *core.ServeEvent) {
	// MFA enrollment
	e.Router.POST("/api/mfa/enroll", h.Enroll).Bind(apis.RequireAuth())

	// Complete enrollment (verify factor)
	e.Router.POST("/api/mfa/enroll/verify", h.VerifyEnrollment).Bind(apis.RequireAuth())

	// List enrolled factors
	e.Router.GET("/api/mfa/factors", h.ListFactors).Bind(apis.RequireAuth())

	// Create challenge for verification
	e.Router.POST("/api/mfa/challenge", h.CreateChallenge).Bind(apis.RequireAuth())

	// Verify MFA (used during login)
	e.Router.POST("/api/mfa/verify", h.Verify).Bind(apis.RequireAuth())

	// Disable MFA
	e.Router.POST("/api/mfa/unenroll", h.Unenroll).Bind(apis.RequireAuth())

	// Get backup codes
	e.Router.POST("/api/mfa/backup-codes", h.GenerateBackupCodes).Bind(apis.RequireAuth())

	// Regenerate backup codes
	e.Router.POST("/api/mfa/backup-codes/regenerate", h.RegenerateBackupCodes).Bind(apis.RequireAuth())

	// Get assurance level
	e.Router.GET("/api/mfa/assurance-level", h.GetAssuranceLevel).Bind(apis.RequireAuth())

	// Verify MFA for sensitive operations (future-ready endpoint)
	e.Router.POST("/api/mfa/verify-sensitive", h.VerifySensitiveOperation).Bind(apis.RequireAuth())

	// Check if sensitive operation verification is valid
	e.Router.GET("/api/mfa/verify-sensitive/{verificationId}", h.CheckSensitiveVerification).Bind(apis.RequireAuth())
}

// Enroll starts MFA enrollment for the authenticated user
func (h *Handler) Enroll(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	var req types.MFAEnrollRequest
	if err := e.BindBody(&req); err != nil {
		return apis.NewBadRequestError("Invalid request body", err)
	}

	// Only TOTP supported for now
	if req.FactorType != "" && req.FactorType != types.MFAFactorTypeTOTP {
		return apis.NewBadRequestError("Only TOTP factor type is supported", nil)
	}

	// Check if user already has a verified factor
	factorsCollection, err := h.app.FindCollectionByNameOrId("mfa_factors")
	if err != nil {
		return apis.NewApiError(500, "Failed to find factors collection", err)
	}

	existingFactors, err := h.app.FindRecordsByFilter(
		factorsCollection,
		"user_id = {:userId} && status = 'verified'",
		"",
		1,
		0,
		dbx.Params{"userId": authRecord.Id},
	)
	if err == nil && len(existingFactors) > 0 {
		return apis.NewBadRequestError("MFA is already enabled. Disable it first before re-enrolling.", nil)
	}

	// Delete any unverified factors for this user
	unverifiedFactors, _ := h.app.FindRecordsByFilter(
		factorsCollection,
		"user_id = {:userId} && status = 'unverified'",
		"",
		100,
		0,
		dbx.Params{"userId": authRecord.Id},
	)
	for _, f := range unverifiedFactors {
		h.app.Delete(f)
	}

	// Generate new TOTP secret
	secret, err := GenerateSecret()
	if err != nil {
		return apis.NewApiError(500, "Failed to generate secret", err)
	}

	// Get user email for the TOTP URI
	userEmail := authRecord.Email()
	if userEmail == "" {
		userEmail = authRecord.Id
	}

	// Create TOTP URI
	config := TOTPConfig{
		Issuer:      IssuerName,
		AccountName: userEmail,
		Secret:      secret,
	}
	totpURI := GenerateTOTPURI(config)

	// Generate QR code
	qrCode, err := GenerateQRCode(totpURI)
	if err != nil {
		return apis.NewApiError(500, "Failed to generate QR code", err)
	}

	// Create unverified factor record
	factorRecord := core.NewRecord(factorsCollection)
	factorRecord.Set("user_id", authRecord.Id)
	factorRecord.Set("factor_type", "totp")
	factorRecord.Set("friendly_name", req.FriendlyName)
	factorRecord.Set("status", "unverified")
	factorRecord.Set("secret", secret)

	if err := h.app.Save(factorRecord); err != nil {
		return apis.NewApiError(500, "Failed to save factor", err)
	}

	response := types.MFAEnrollResponse{
		FactorID:     factorRecord.Id,
		FactorType:   string(types.MFAFactorTypeTOTP),
		TOTPURI:      totpURI,
		TOTPSecret:   secret,
		QRCodeBase64: qrCode,
		FriendlyName: req.FriendlyName,
	}

	return e.JSON(http.StatusOK, response)
}

// VerifyEnrollment completes MFA enrollment by verifying the TOTP code
func (h *Handler) VerifyEnrollment(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	var req types.MFAVerifyRequest
	if err := e.BindBody(&req); err != nil {
		return apis.NewBadRequestError("Invalid request body", err)
	}

	if req.FactorID == "" || req.Code == "" {
		return apis.NewBadRequestError("factor_id and code are required", nil)
	}

	// Find the factor
	factorRecord, err := h.app.FindRecordById("mfa_factors", req.FactorID)
	if err != nil {
		return apis.NewNotFoundError("Factor not found", err)
	}

	// Verify ownership
	if factorRecord.GetString("user_id") != authRecord.Id {
		return apis.NewForbiddenError("Not authorized to access this factor", nil)
	}

	// Check if already verified
	if factorRecord.GetString("status") == "verified" {
		return apis.NewBadRequestError("Factor is already verified", nil)
	}

	// Verify the TOTP code
	secret := factorRecord.GetString("secret")
	valid, err := VerifyTOTP(secret, req.Code)
	if err != nil {
		return apis.NewApiError(500, "Failed to verify code", err)
	}
	if !valid {
		return apis.NewBadRequestError("Invalid verification code", nil)
	}

	// Check for replay attack
	if h.isTokenUsed(req.FactorID, req.Code) {
		return apis.NewBadRequestError("This code has already been used", nil)
	}

	// Mark token as used
	h.markTokenUsed(req.FactorID, req.Code)

	// Update factor status to verified
	factorRecord.Set("status", "verified")
	if err := h.app.Save(factorRecord); err != nil {
		return apis.NewApiError(500, "Failed to update factor", err)
	}

	// Update user's mfa_enabled flag
	authRecord.Set("mfa_enabled", true)
	if err := h.app.Save(authRecord); err != nil {
		return apis.NewApiError(500, "Failed to update user", err)
	}

	// Generate backup codes
	codes, err := GenerateBackupCodes()
	if err != nil {
		return apis.NewApiError(500, "Failed to generate backup codes", err)
	}

	// Store hashed backup codes
	if err := h.storeBackupCodes(authRecord.Id, codes); err != nil {
		return apis.NewApiError(500, "Failed to store backup codes", err)
	}

	response := types.MFABackupCodesResponse{
		Codes: codes,
		Message: "MFA has been enabled. Please save these backup codes in a secure location. " +
			"Each code can only be used once, and generating new codes will invalidate these.",
	}

	return e.JSON(http.StatusOK, response)
}

// ListFactors returns all MFA factors for the authenticated user
func (h *Handler) ListFactors(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	factorsCollection, err := h.app.FindCollectionByNameOrId("mfa_factors")
	if err != nil {
		return apis.NewApiError(500, "Failed to find factors collection", err)
	}

	factors, err := h.app.FindRecordsByFilter(
		factorsCollection,
		"user_id = {:userId} && status = 'verified'",
		"",
		100,
		0,
		dbx.Params{"userId": authRecord.Id},
	)
	if err != nil {
		return apis.NewApiError(500, "Failed to fetch factors", err)
	}

	var totpFactors []types.MFAFactorInfo
	for _, f := range factors {
		totpFactors = append(totpFactors, types.MFAFactorInfo{
			ID:           f.Id,
			FactorType:   types.MFAFactorType(f.GetString("factor_type")),
			FriendlyName: f.GetString("friendly_name"),
			Status:       types.MFAFactorStatus(f.GetString("status")),
			Created:      f.GetDateTime("created").Time(),
		})
	}

	response := types.MFAListFactorsResponse{
		TOTP: totpFactors,
	}

	return e.JSON(http.StatusOK, response)
}

// CreateChallenge creates an MFA challenge for verification
func (h *Handler) CreateChallenge(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	var req types.MFAChallengeRequest
	if err := e.BindBody(&req); err != nil {
		return apis.NewBadRequestError("Invalid request body", err)
	}

	// Find the factor
	factorRecord, err := h.app.FindRecordById("mfa_factors", req.FactorID)
	if err != nil {
		return apis.NewNotFoundError("Factor not found", err)
	}

	// Verify ownership
	if factorRecord.GetString("user_id") != authRecord.Id {
		return apis.NewForbiddenError("Not authorized to access this factor", nil)
	}

	// Factor must be verified
	if factorRecord.GetString("status") != "verified" {
		return apis.NewBadRequestError("Factor is not verified", nil)
	}

	// Create challenge
	challengesCollection, err := h.app.FindCollectionByNameOrId("mfa_challenges")
	if err != nil {
		return apis.NewApiError(500, "Failed to find challenges collection", err)
	}

	expiresAt := time.Now().Add(ChallengeExpiryMinutes * time.Minute)

	challengeRecord := core.NewRecord(challengesCollection)
	challengeRecord.Set("factor_id", req.FactorID)
	challengeRecord.Set("status", "pending")
	challengeRecord.Set("expires_at", expiresAt)

	if err := h.app.Save(challengeRecord); err != nil {
		return apis.NewApiError(500, "Failed to create challenge", err)
	}

	response := types.MFAChallengeResponse{
		ChallengeID: challengeRecord.Id,
		FactorID:    req.FactorID,
		ExpiresAt:   expiresAt,
	}

	return e.JSON(http.StatusOK, response)
}

// Verify verifies an MFA challenge
func (h *Handler) Verify(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	var req types.MFAVerifyRequest
	if err := e.BindBody(&req); err != nil {
		return apis.NewBadRequestError("Invalid request body", err)
	}

	if req.FactorID == "" || req.ChallengeID == "" || req.Code == "" {
		return apis.NewBadRequestError("factor_id, challenge_id, and code are required", nil)
	}

	// Find the challenge
	challengeRecord, err := h.app.FindRecordById("mfa_challenges", req.ChallengeID)
	if err != nil {
		return apis.NewNotFoundError("Challenge not found", err)
	}

	// Check challenge status
	if challengeRecord.GetString("status") != "pending" {
		return apis.NewBadRequestError("Challenge is not pending", nil)
	}

	// Check challenge hasn't expired
	expiresAt := challengeRecord.GetDateTime("expires_at").Time()
	if time.Now().After(expiresAt) {
		challengeRecord.Set("status", "expired")
		h.app.Save(challengeRecord)
		return apis.NewBadRequestError("Challenge has expired", nil)
	}

	// Find the factor
	factorRecord, err := h.app.FindRecordById("mfa_factors", req.FactorID)
	if err != nil {
		return apis.NewNotFoundError("Factor not found", err)
	}

	// Verify ownership
	if factorRecord.GetString("user_id") != authRecord.Id {
		return apis.NewForbiddenError("Not authorized to access this factor", nil)
	}

	// Try TOTP verification first
	secret := factorRecord.GetString("secret")
	valid, err := VerifyTOTP(secret, req.Code)
	if err != nil {
		return apis.NewApiError(500, "Failed to verify code", err)
	}

	// If TOTP failed, try backup code
	if !valid {
		valid, err = h.verifyAndConsumeBackupCode(authRecord.Id, req.Code)
		if err != nil {
			return apis.NewApiError(500, "Failed to verify backup code", err)
		}
	}

	if !valid {
		return apis.NewBadRequestError("Invalid verification code", nil)
	}

	// Check for replay attack (only for TOTP, backup codes are already single-use)
	if len(req.Code) == TOTPDigits {
		if h.isTokenUsed(req.FactorID, req.Code) {
			return apis.NewBadRequestError("This code has already been used", nil)
		}
		h.markTokenUsed(req.FactorID, req.Code)
	}

	// Update challenge status
	challengeRecord.Set("status", "verified")
	challengeRecord.Set("verified_at", time.Now())
	if err := h.app.Save(challengeRecord); err != nil {
		return apis.NewApiError(500, "Failed to update challenge", err)
	}

	response := types.MFAVerifyResponse{
		Success: true,
		AAL:     "aal2",
	}

	return e.JSON(http.StatusOK, response)
}

// Unenroll disables MFA for the user
func (h *Handler) Unenroll(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	var req types.MFAUnenrollRequest
	if err := e.BindBody(&req); err != nil {
		return apis.NewBadRequestError("Invalid request body", err)
	}

	if req.FactorID == "" || req.Code == "" {
		return apis.NewBadRequestError("factor_id and code are required", nil)
	}

	// Find the factor
	factorRecord, err := h.app.FindRecordById("mfa_factors", req.FactorID)
	if err != nil {
		return apis.NewNotFoundError("Factor not found", err)
	}

	// Verify ownership
	if factorRecord.GetString("user_id") != authRecord.Id {
		return apis.NewForbiddenError("Not authorized to access this factor", nil)
	}

	// Verify the TOTP code before allowing unenroll
	secret := factorRecord.GetString("secret")
	valid, err := VerifyTOTP(secret, req.Code)
	if err != nil {
		return apis.NewApiError(500, "Failed to verify code", err)
	}

	// Also allow backup code for unenroll
	if !valid {
		valid, _ = h.verifyAndConsumeBackupCode(authRecord.Id, req.Code)
	}

	if !valid {
		return apis.NewBadRequestError("Invalid verification code", nil)
	}

	// Delete the factor
	if err := h.app.Delete(factorRecord); err != nil {
		return apis.NewApiError(500, "Failed to delete factor", err)
	}

	// Delete all backup codes for this user
	h.deleteBackupCodes(authRecord.Id)

	// Update user's mfa_enabled flag
	authRecord.Set("mfa_enabled", false)
	if err := h.app.Save(authRecord); err != nil {
		return apis.NewApiError(500, "Failed to update user", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"success": true,
		"message": "MFA has been disabled",
	})
}

// GenerateBackupCodes generates new backup codes (requires verified MFA)
func (h *Handler) GenerateBackupCodes(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	// Check if user has MFA enabled
	if !authRecord.GetBool("mfa_enabled") {
		return apis.NewBadRequestError("MFA is not enabled", nil)
	}

	// Generate backup codes
	codes, err := GenerateBackupCodes()
	if err != nil {
		return apis.NewApiError(500, "Failed to generate backup codes", err)
	}

	// Delete existing backup codes and store new ones
	h.deleteBackupCodes(authRecord.Id)
	if err := h.storeBackupCodes(authRecord.Id, codes); err != nil {
		return apis.NewApiError(500, "Failed to store backup codes", err)
	}

	response := types.MFABackupCodesResponse{
		Codes: codes,
		Message: "New backup codes have been generated. Previous backup codes are now invalid. " +
			"Please save these in a secure location.",
	}

	return e.JSON(http.StatusOK, response)
}

// RegenerateBackupCodes regenerates backup codes with verification
func (h *Handler) RegenerateBackupCodes(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	var req types.MFARegenerateBackupCodesRequest
	if err := e.BindBody(&req); err != nil {
		return apis.NewBadRequestError("Invalid request body", err)
	}

	if req.Code == "" {
		return apis.NewBadRequestError("code is required", nil)
	}

	// Check if user has MFA enabled
	if !authRecord.GetBool("mfa_enabled") {
		return apis.NewBadRequestError("MFA is not enabled", nil)
	}

	// Find user's factor
	factorsCollection, err := h.app.FindCollectionByNameOrId("mfa_factors")
	if err != nil {
		return apis.NewApiError(500, "Failed to find factors collection", err)
	}

	factors, err := h.app.FindRecordsByFilter(
		factorsCollection,
		"user_id = {:userId} && status = 'verified'",
		"",
		1,
		0,
		dbx.Params{"userId": authRecord.Id},
	)
	if err != nil || len(factors) == 0 {
		return apis.NewBadRequestError("No verified MFA factor found", nil)
	}

	// Verify the TOTP code
	secret := factors[0].GetString("secret")
	valid, err := VerifyTOTP(secret, req.Code)
	if err != nil {
		return apis.NewApiError(500, "Failed to verify code", err)
	}
	if !valid {
		return apis.NewBadRequestError("Invalid verification code", nil)
	}

	// Generate backup codes
	codes, err := GenerateBackupCodes()
	if err != nil {
		return apis.NewApiError(500, "Failed to generate backup codes", err)
	}

	// Delete existing backup codes and store new ones
	h.deleteBackupCodes(authRecord.Id)
	if err := h.storeBackupCodes(authRecord.Id, codes); err != nil {
		return apis.NewApiError(500, "Failed to store backup codes", err)
	}

	response := types.MFABackupCodesResponse{
		Codes: codes,
		Message: "New backup codes have been generated. Previous backup codes are now invalid. " +
			"Please save these in a secure location.",
	}

	return e.JSON(http.StatusOK, response)
}

// GetAssuranceLevel returns the current authenticator assurance level
func (h *Handler) GetAssuranceLevel(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	mfaEnabled := authRecord.GetBool("mfa_enabled")

	currentLevel := "aal1"
	nextLevel := "aal1"

	if mfaEnabled {
		nextLevel = "aal2"
		// In a real implementation, we'd check if the session has been MFA-verified
		// For now, this indicates the user needs to complete MFA
	}

	response := types.MFAAssuranceLevelResponse{
		CurrentLevel: currentLevel,
		NextLevel:    nextLevel,
	}

	return e.JSON(http.StatusOK, response)
}

// VerifySensitiveOperation verifies MFA for a sensitive operation
func (h *Handler) VerifySensitiveOperation(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	var req types.MFASensitiveOperationVerifyRequest
	if err := e.BindBody(&req); err != nil {
		return apis.NewBadRequestError("Invalid request body", err)
	}

	if req.Code == "" || req.Operation == "" {
		return apis.NewBadRequestError("code and operation are required", nil)
	}

	// Check if user has MFA enabled
	if !authRecord.GetBool("mfa_enabled") {
		return apis.NewBadRequestError("MFA is not enabled for this user", nil)
	}

	// Find user's factor
	factorsCollection, err := h.app.FindCollectionByNameOrId("mfa_factors")
	if err != nil {
		return apis.NewApiError(500, "Failed to find factors collection", err)
	}

	factors, err := h.app.FindRecordsByFilter(
		factorsCollection,
		"user_id = {:userId} && status = 'verified'",
		"",
		1,
		0,
		dbx.Params{"userId": authRecord.Id},
	)
	if err != nil || len(factors) == 0 {
		return apis.NewBadRequestError("No verified MFA factor found", nil)
	}

	// Try TOTP verification
	secret := factors[0].GetString("secret")
	valid, err := VerifyTOTP(secret, req.Code)
	if err != nil {
		return apis.NewApiError(500, "Failed to verify code", err)
	}

	// If TOTP failed, try backup code
	if !valid {
		valid, err = h.verifyAndConsumeBackupCode(authRecord.Id, req.Code)
		if err != nil {
			return apis.NewApiError(500, "Failed to verify backup code", err)
		}
	}

	if !valid {
		return apis.NewBadRequestError("Invalid verification code", nil)
	}

	// Check for replay attack (only for TOTP)
	if len(req.Code) == TOTPDigits {
		if h.isTokenUsed(factors[0].Id, req.Code) {
			return apis.NewBadRequestError("This code has already been used", nil)
		}
		h.markTokenUsed(factors[0].Id, req.Code)
	}

	// Create sensitive operation verification record
	sensitiveOpsCollection, err := h.app.FindCollectionByNameOrId("mfa_sensitive_verifications")
	if err != nil {
		return apis.NewApiError(500, "Failed to find sensitive ops collection", err)
	}

	validUntil := time.Now().Add(SensitiveOpExpiryMinutes * time.Minute)

	verificationRecord := core.NewRecord(sensitiveOpsCollection)
	verificationRecord.Set("user_id", authRecord.Id)
	verificationRecord.Set("operation", req.Operation)
	verificationRecord.Set("valid_until", validUntil)
	verificationRecord.Set("used", false)

	if err := h.app.Save(verificationRecord); err != nil {
		return apis.NewApiError(500, "Failed to create verification", err)
	}

	response := types.MFASensitiveOperationVerifyResponse{
		Success:        true,
		VerificationID: verificationRecord.Id,
		ValidUntil:     validUntil,
		Operation:      req.Operation,
		AllowedEndpoints: []string{
			fmt.Sprintf("/api/%s/*", req.Operation),
		},
	}

	return e.JSON(http.StatusOK, response)
}

// CheckSensitiveVerification checks if a sensitive operation verification is valid
func (h *Handler) CheckSensitiveVerification(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	verificationId := e.Request.PathValue("verificationId")
	if verificationId == "" {
		return apis.NewBadRequestError("verificationId is required", nil)
	}

	// Find the verification
	verificationRecord, err := h.app.FindRecordById("mfa_sensitive_verifications", verificationId)
	if err != nil {
		return apis.NewNotFoundError("Verification not found", err)
	}

	// Verify ownership
	if verificationRecord.GetString("user_id") != authRecord.Id {
		return apis.NewForbiddenError("Not authorized to access this verification", nil)
	}

	// Check if used
	if verificationRecord.GetBool("used") {
		return e.JSON(http.StatusOK, map[string]any{
			"valid":  false,
			"reason": "Verification has already been used",
		})
	}

	// Check if expired
	validUntil := verificationRecord.GetDateTime("valid_until").Time()
	if time.Now().After(validUntil) {
		return e.JSON(http.StatusOK, map[string]any{
			"valid":  false,
			"reason": "Verification has expired",
		})
	}

	return e.JSON(http.StatusOK, map[string]any{
		"valid":       true,
		"operation":   verificationRecord.GetString("operation"),
		"valid_until": validUntil,
	})
}

// Helper methods

func (h *Handler) storeBackupCodes(userID string, codes []string) error {
	backupCodesCollection, err := h.app.FindCollectionByNameOrId("mfa_backup_codes")
	if err != nil {
		return err
	}

	batchID, err := GenerateBatchID()
	if err != nil {
		return err
	}

	for _, code := range codes {
		hash, err := HashBackupCode(code)
		if err != nil {
			return err
		}

		record := core.NewRecord(backupCodesCollection)
		record.Set("user_id", userID)
		record.Set("code_hash", hash)
		record.Set("used", false)
		record.Set("batch_id", batchID)

		if err := h.app.Save(record); err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) deleteBackupCodes(userID string) error {
	backupCodesCollection, err := h.app.FindCollectionByNameOrId("mfa_backup_codes")
	if err != nil {
		return err
	}

	codes, err := h.app.FindRecordsByFilter(
		backupCodesCollection,
		"user_id = {:userId}",
		"",
		1000,
		0,
		dbx.Params{"userId": userID},
	)
	if err != nil {
		return nil // No codes to delete
	}

	for _, code := range codes {
		h.app.Delete(code)
	}

	return nil
}

func (h *Handler) verifyAndConsumeBackupCode(userID, code string) (bool, error) {
	backupCodesCollection, err := h.app.FindCollectionByNameOrId("mfa_backup_codes")
	if err != nil {
		return false, err
	}

	codes, err := h.app.FindRecordsByFilter(
		backupCodesCollection,
		"user_id = {:userId} && used = false",
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
		if VerifyBackupCode(code, hash) {
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

func (h *Handler) isTokenUsed(factorID, token string) bool {
	tokenHash := HashToken(token)

	usedTokensCollection, err := h.app.FindCollectionByNameOrId("mfa_used_tokens")
	if err != nil {
		return false
	}

	records, err := h.app.FindRecordsByFilter(
		usedTokensCollection,
		"factor_id = {:factorId} && token_hash = {:tokenHash}",
		"",
		1,
		0,
		dbx.Params{"factorId": factorID, "tokenHash": tokenHash},
	)

	return err == nil && len(records) > 0
}

func (h *Handler) markTokenUsed(factorID, token string) {
	tokenHash := HashToken(token)

	usedTokensCollection, err := h.app.FindCollectionByNameOrId("mfa_used_tokens")
	if err != nil {
		return
	}

	record := core.NewRecord(usedTokensCollection)
	record.Set("factor_id", factorID)
	record.Set("token_hash", tokenHash)
	record.Set("used_at", time.Now())

	h.app.Save(record)

	// Clean up old tokens (older than 5 minutes)
	h.cleanupOldTokens(factorID)
}

func (h *Handler) cleanupOldTokens(factorID string) {
	usedTokensCollection, err := h.app.FindCollectionByNameOrId("mfa_used_tokens")
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-5 * time.Minute)

	oldTokens, err := h.app.FindRecordsByFilter(
		usedTokensCollection,
		"factor_id = {:factorId} && used_at < {:cutoff}",
		"",
		100,
		0,
		dbx.Params{"factorId": factorID, "cutoff": cutoff},
	)
	if err != nil {
		return
	}

	for _, token := range oldTokens {
		h.app.Delete(token)
	}
}
