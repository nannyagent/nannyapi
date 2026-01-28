package mfa

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image/png"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"golang.org/x/crypto/bcrypt"
)

const (
	// TOTPSecretLength is the length of the TOTP secret in bytes
	TOTPSecretLength = 20

	// TOTPDigits is the number of digits in a TOTP code
	TOTPDigits = 6

	// TOTPPeriod is the time step in seconds
	TOTPPeriod = 30

	// TOTPSkew allows for clock drift (1 period before/after)
	TOTPSkew = 1

	// BackupCodeCount is the number of backup codes generated
	BackupCodeCount = 10

	// BackupCodeLength is the length of each backup code
	BackupCodeLength = 8

	// ChallengeExpiryMinutes is how long a challenge remains valid
	ChallengeExpiryMinutes = 5

	// SensitiveOpExpiryMinutes is how long a sensitive operation verification is valid
	SensitiveOpExpiryMinutes = 10

	// QRCodeSize is the size of the QR code image in pixels
	QRCodeSize = 256
)

// TOTPConfig holds configuration for TOTP generation
type TOTPConfig struct {
	Issuer      string
	AccountName string
	Secret      string
}

// GenerateSecret generates a cryptographically secure random secret for TOTP
func GenerateSecret() (string, error) {
	secret := make([]byte, TOTPSecretLength)
	_, err := rand.Read(secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateTOTPURI creates the otpauth:// URI for authenticator apps
func GenerateTOTPURI(config TOTPConfig) string {
	// Format: otpauth://totp/ISSUER:ACCOUNT?secret=SECRET&issuer=ISSUER&algorithm=SHA1&digits=6&period=30
	return fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		config.Issuer,
		config.AccountName,
		config.Secret,
		config.Issuer,
		TOTPDigits,
		TOTPPeriod,
	)
}

// GenerateQRCode generates a QR code PNG image as base64 string
func GenerateQRCode(uri string) (string, error) {
	// Create QR code
	qrCode, err := qr.Encode(uri, qr.M, qr.Auto)
	if err != nil {
		return "", fmt.Errorf("failed to create QR code: %w", err)
	}

	// Scale to desired size
	qrCode, err = barcode.Scale(qrCode, QRCodeSize, QRCodeSize)
	if err != nil {
		return "", fmt.Errorf("failed to scale QR code: %w", err)
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, qrCode); err != nil {
		return "", fmt.Errorf("failed to encode QR code: %w", err)
	}

	// Return as data URI
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// GenerateTOTP generates a TOTP code for the given secret and time
func GenerateTOTP(secret string, timestamp time.Time) (string, error) {
	// Decode the base32 secret
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", fmt.Errorf("invalid secret: %w", err)
	}

	// Calculate time counter
	counter := uint64(timestamp.Unix()) / TOTPPeriod

	// Generate HOTP
	return generateHOTP(secretBytes, counter)
}

// generateHOTP generates an HOTP code
func generateHOTP(secret []byte, counter uint64) (string, error) {
	// Convert counter to bytes
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, counter)

	// Calculate HMAC-SHA1
	h := hmac.New(sha1.New, secret)
	h.Write(counterBytes)
	hash := h.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0f
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	// Get the specified number of digits
	code = code % 1000000 // 10^6 for 6 digits

	return fmt.Sprintf("%06d", code), nil
}

// VerifyTOTP verifies a TOTP code against the secret
// Returns true if the code is valid for current time ± skew
func VerifyTOTP(secret, code string) (bool, error) {
	if len(code) != TOTPDigits {
		return false, nil
	}

	now := time.Now()

	// Check current time and allowed skew
	for i := -TOTPSkew; i <= TOTPSkew; i++ {
		checkTime := now.Add(time.Duration(i*TOTPPeriod) * time.Second)
		expectedCode, err := GenerateTOTP(secret, checkTime)
		if err != nil {
			return false, err
		}
		if hmac.Equal([]byte(code), []byte(expectedCode)) {
			return true, nil
		}
	}

	return false, nil
}

// GenerateBackupCodes generates a set of single-use backup codes
func GenerateBackupCodes() ([]string, error) {
	codes := make([]string, BackupCodeCount)
	for i := 0; i < BackupCodeCount; i++ {
		code, err := generateBackupCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

// generateBackupCode generates a single backup code
func generateBackupCode() (string, error) {
	// Generate random bytes
	b := make([]byte, BackupCodeLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate backup code: %w", err)
	}

	// Convert to alphanumeric string (A-Z, 0-9)
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Removed I, O, 0, 1 for clarity
	code := make([]byte, BackupCodeLength)
	for i, v := range b {
		code[i] = charset[int(v)%len(charset)]
	}

	// Format as XXXX-XXXX for readability
	return string(code[:4]) + "-" + string(code[4:]), nil
}

// HashBackupCode creates a secure hash of a backup code
func HashBackupCode(code string) (string, error) {
	// Normalize the code (remove hyphens, uppercase)
	normalized := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	hash, err := bcrypt.GenerateFromPassword([]byte(normalized), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash backup code: %w", err)
	}
	return string(hash), nil
}

// VerifyBackupCode checks if a provided code matches a stored hash
func VerifyBackupCode(code, hash string) bool {
	// Normalize the code
	normalized := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(normalized))
	return err == nil
}

// HashToken creates a hash of a TOTP token for replay prevention
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(h[:])
}

// GenerateBatchID generates a unique ID for a batch of backup codes
func GenerateBatchID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
