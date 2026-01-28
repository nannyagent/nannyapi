package mfa

import (
	"testing"
	"time"
)

func TestGenerateSecret(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}
	if len(secret) == 0 {
		t.Error("Secret should not be empty")
	}
	if len(secret) != 32 {
		t.Errorf("Expected secret length 32, got %d", len(secret))
	}
	secret2, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}
	if secret == secret2 {
		t.Error("Two generated secrets should be different")
	}
}

func TestGenerateTOTPURI(t *testing.T) {
	config := TOTPConfig{
		Issuer:      "NannyAPI",
		AccountName: "test@example.com",
		Secret:      "JBSWY3DPEHPK3PXP",
	}
	uri := GenerateTOTPURI(config)
	expected := "otpauth://totp/NannyAPI:test@example.com?secret=JBSWY3DPEHPK3PXP&issuer=NannyAPI&algorithm=SHA1&digits=6&period=30"
	if uri != expected {
		t.Errorf("Expected URI %s, got %s", expected, uri)
	}
}

func TestGenerateQRCode(t *testing.T) {
	uri := "otpauth://totp/NannyAPI:test@example.com?secret=JBSWY3DPEHPK3PXP&issuer=NannyAPI"
	qrCode, err := GenerateQRCode(uri)
	if err != nil {
		t.Fatalf("GenerateQRCode failed: %v", err)
	}
	prefix := "data:image/png;base64,"
	if len(qrCode) < len(prefix) || qrCode[:len(prefix)] != prefix {
		t.Error("QR code should be a base64 data URI")
	}
}

func TestGenerateTOTP(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	code, err := GenerateTOTP(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateTOTP failed: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("Expected 6-digit code, got %d digits", len(code))
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("Code should only contain digits, found %c", c)
		}
	}
}

func TestVerifyTOTP(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}
	code, err := GenerateTOTP(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateTOTP failed: %v", err)
	}
	valid, err := VerifyTOTP(secret, code)
	if err != nil {
		t.Fatalf("VerifyTOTP failed: %v", err)
	}
	if !valid {
		t.Error("Valid code should be accepted")
	}
	valid, err = VerifyTOTP(secret, "000000")
	if err != nil {
		t.Fatalf("VerifyTOTP failed: %v", err)
	}
	if valid {
		t.Error("Invalid code should be rejected")
	}
}

func TestGenerateBackupCodes(t *testing.T) {
	codes, err := GenerateBackupCodes()
	if err != nil {
		t.Fatalf("GenerateBackupCodes failed: %v", err)
	}
	if len(codes) != BackupCodeCount {
		t.Errorf("Expected %d codes, got %d", BackupCodeCount, len(codes))
	}
	for _, code := range codes {
		if len(code) != 9 {
			t.Errorf("Expected code length 9, got %d for code %s", len(code), code)
		}
		if code[4] != '-' {
			t.Errorf("Expected hyphen at position 4, got %c", code[4])
		}
	}
	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Error("Backup codes should be unique")
		}
		seen[code] = true
	}
}

func TestHashAndVerifyBackupCode(t *testing.T) {
	codes, err := GenerateBackupCodes()
	if err != nil {
		t.Fatalf("GenerateBackupCodes failed: %v", err)
	}
	code := codes[0]
	hash, err := HashBackupCode(code)
	if err != nil {
		t.Fatalf("HashBackupCode failed: %v", err)
	}
	if !VerifyBackupCode(code, hash) {
		t.Error("Exact code should verify")
	}
	codeNoHyphen := code[:4] + code[5:]
	if !VerifyBackupCode(codeNoHyphen, hash) {
		t.Error("Code without hyphen should verify")
	}
	if VerifyBackupCode("ABCD-EFGH", hash) {
		t.Error("Wrong code should not verify")
	}
}

func TestHashToken(t *testing.T) {
	token := "123456"
	hash := HashToken(token)
	if hash == "" {
		t.Error("Hash should not be empty")
	}
	hash2 := HashToken(token)
	if hash != hash2 {
		t.Error("Same token should produce same hash")
	}
	hash3 := HashToken("654321")
	if hash == hash3 {
		t.Error("Different tokens should produce different hashes")
	}
}

func TestGenerateBatchID(t *testing.T) {
	id1, err := GenerateBatchID()
	if err != nil {
		t.Fatalf("GenerateBatchID failed: %v", err)
	}
	if len(id1) != 32 {
		t.Errorf("Expected batch ID length 32, got %d", len(id1))
	}
	id2, err := GenerateBatchID()
	if err != nil {
		t.Fatalf("GenerateBatchID failed: %v", err)
	}
	if id1 == id2 {
		t.Error("Two batch IDs should be different")
	}
}
