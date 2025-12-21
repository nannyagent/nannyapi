package security

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name      string
		password1 string
		password2 string
		wantSame  bool
	}{
		{
			name:      "same password produces same hash",
			password1: "TestPassword123!",
			password2: "TestPassword123!",
			wantSame:  true,
		},
		{
			name:      "different passwords produce different hashes",
			password1: "TestPassword123!",
			password2: "DifferentPass456!",
			wantSame:  false,
		},
		{
			name:      "similar passwords produce different hashes",
			password1: "TestPassword123!",
			password2: "TestPassword123!!",
			wantSame:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := HashPassword(tt.password1)
			hash2 := HashPassword(tt.password2)

			if tt.wantSame && hash1 != hash2 {
				t.Errorf("HashPassword() expected same hashes for identical passwords, got %s and %s", hash1, hash2)
			}

			if !tt.wantSame && hash1 == hash2 {
				t.Errorf("HashPassword() expected different hashes for different passwords, got same hash: %s", hash1)
			}

			// Verify hash format (SHA-256 produces 64 hex characters)
			if len(hash1) != 64 {
				t.Errorf("HashPassword() hash length = %d, want 64", len(hash1))
			}
		})
	}
}

func TestHashPasswordConsistency(t *testing.T) {
	password := "ConsistencyTest123!"

	// Hash same password multiple times
	hash1 := HashPassword(password)
	hash2 := HashPassword(password)
	hash3 := HashPassword(password)

	if hash1 != hash2 || hash2 != hash3 {
		t.Errorf("HashPassword() not consistent: got %s, %s, %s", hash1, hash2, hash3)
	}
}

func TestHashPasswordEmpty(t *testing.T) {
	hash := HashPassword("")

	// Empty string should still produce a valid hash
	if len(hash) != 64 {
		t.Errorf("HashPassword(\"\") hash length = %d, want 64", len(hash))
	}

	// Empty string should produce consistent hash
	hash2 := HashPassword("")
	if hash != hash2 {
		t.Errorf("HashPassword(\"\") not consistent")
	}
}
