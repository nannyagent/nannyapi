package types

import "time"

// MFAFactorStatus represents the status of an MFA factor
type MFAFactorStatus string

const (
	MFAFactorStatusUnverified MFAFactorStatus = "unverified"
	MFAFactorStatusVerified   MFAFactorStatus = "verified"
)

// MFAFactorType represents the type of MFA factor
type MFAFactorType string

const (
	MFAFactorTypeTOTP MFAFactorType = "totp"
)

// MFAChallengeStatus represents the status of an MFA challenge
type MFAChallengeStatus string

const (
	MFAChallengeStatusPending  MFAChallengeStatus = "pending"
	MFAChallengeStatusVerified MFAChallengeStatus = "verified"
	MFAChallengeStatusExpired  MFAChallengeStatus = "expired"
)

// MFAFactor represents an enrolled MFA factor for a user
type MFAFactor struct {
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	FactorType   MFAFactorType   `json:"factor_type"`
	FriendlyName string          `json:"friendly_name,omitempty"`
	Status       MFAFactorStatus `json:"status"`
	Secret       string          `json:"-"` // Never expose in JSON
	Created      time.Time       `json:"created"`
	Updated      time.Time       `json:"updated"`
}

// MFABackupCode represents a single-use backup code for MFA recovery
type MFABackupCode struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	CodeHash string    `json:"-"` // Hashed backup code, never expose
	Used     bool      `json:"used"`
	UsedAt   time.Time `json:"used_at,omitempty"`
	Created  time.Time `json:"created"`
}

// MFAChallenge represents an MFA verification challenge
type MFAChallenge struct {
	ID        string             `json:"id"`
	FactorID  string             `json:"factor_id"`
	Status    MFAChallengeStatus `json:"status"`
	ExpiresAt time.Time          `json:"expires_at"`
	Created   time.Time          `json:"created"`
	Verified  time.Time          `json:"verified_at,omitempty"`
}

// MFAUsedToken tracks used TOTP tokens to prevent replay attacks
type MFAUsedToken struct {
	ID       string    `json:"id"`
	FactorID string    `json:"factor_id"`
	Token    string    `json:"-"` // The hash of the token that was used
	UsedAt   time.Time `json:"used_at"`
}

// --- API Request/Response Types ---

// MFAEnrollRequest is the request to start MFA enrollment
type MFAEnrollRequest struct {
	FactorType   MFAFactorType `json:"factor_type"`   // Currently only "totp"
	FriendlyName string        `json:"friendly_name"` // User-provided name for the factor
}

// MFAEnrollResponse is returned when starting MFA enrollment
type MFAEnrollResponse struct {
	FactorID     string `json:"factor_id"`
	FactorType   string `json:"factor_type"`
	TOTPURI      string `json:"totp_uri"`       // otpauth:// URI for authenticator apps
	TOTPSecret   string `json:"totp_secret"`    // Base32-encoded secret for manual entry
	QRCodeBase64 string `json:"qr_code_base64"` // Base64-encoded PNG QR code image
	FriendlyName string `json:"friendly_name,omitempty"`
}

// MFAChallengeRequest is the request to create a challenge
type MFAChallengeRequest struct {
	FactorID string `json:"factor_id"`
}

// MFAChallengeResponse is returned when a challenge is created
type MFAChallengeResponse struct {
	ChallengeID string    `json:"challenge_id"`
	FactorID    string    `json:"factor_id"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// MFAVerifyRequest is the request to verify an MFA challenge
type MFAVerifyRequest struct {
	FactorID    string `json:"factor_id"`
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"` // 6-digit TOTP code or backup code
}

// MFAVerifyResponse is returned after successful MFA verification
type MFAVerifyResponse struct {
	Success bool   `json:"success"`
	AAL     string `json:"aal"` // Authenticator Assurance Level: "aal1" or "aal2"
}

// MFAUnenrollRequest is the request to disable MFA
type MFAUnenrollRequest struct {
	FactorID string `json:"factor_id"`
	Code     string `json:"code"` // Requires valid TOTP code to disable
}

// MFAListFactorsResponse lists all MFA factors for the user
type MFAListFactorsResponse struct {
	TOTP []MFAFactorInfo `json:"totp"`
}

// MFAFactorInfo is a safe representation of an MFA factor (no secrets)
type MFAFactorInfo struct {
	ID           string          `json:"id"`
	FactorType   MFAFactorType   `json:"factor_type"`
	FriendlyName string          `json:"friendly_name"`
	Status       MFAFactorStatus `json:"status"`
	Created      time.Time       `json:"created"`
}

// MFABackupCodesResponse returns newly generated backup codes
type MFABackupCodesResponse struct {
	Codes   []string `json:"codes"`   // Plain text codes (shown only once)
	Message string   `json:"message"` // Warning about storing securely
}

// MFARegenerateBackupCodesRequest is for regenerating backup codes
type MFARegenerateBackupCodesRequest struct {
	Code string `json:"code"` // Requires valid TOTP code to regenerate
}

// MFAAssuranceLevelResponse returns the current AAL
type MFAAssuranceLevelResponse struct {
	CurrentLevel string `json:"current_level"` // "aal1" or "aal2"
	NextLevel    string `json:"next_level"`    // What level user could reach
}

// MFAVerifyBackupCodeRequest verifies using a backup code
type MFAVerifyBackupCodeRequest struct {
	Code string `json:"code"` // The backup code
}

// MFASensitiveOperationVerifyRequest is for verifying MFA before sensitive operations
type MFASensitiveOperationVerifyRequest struct {
	Code      string `json:"code"`      // TOTP code or backup code
	Operation string `json:"operation"` // Name of the operation being performed
}

// MFASensitiveOperationVerifyResponse confirms MFA for sensitive operation
type MFASensitiveOperationVerifyResponse struct {
	Success          bool      `json:"success"`
	VerificationID   string    `json:"verification_id"`   // ID to pass with subsequent request
	ValidUntil       time.Time `json:"valid_until"`       // When this verification expires
	Operation        string    `json:"operation"`         // The operation this verification is for
	AllowedEndpoints []string  `json:"allowed_endpoints"` // Endpoints this verification applies to
}

// MFASessionInfo stores MFA session state
type MFASessionInfo struct {
	UserID        string    `json:"user_id"`
	AAL           string    `json:"aal"` // "aal1" or "aal2"
	MFAVerified   bool      `json:"mfa_verified"`
	MFAVerifiedAt time.Time `json:"mfa_verified_at,omitempty"`
	MFARequired   bool      `json:"mfa_required"` // User has MFA enabled
	FactorID      string    `json:"factor_id,omitempty"`
}
