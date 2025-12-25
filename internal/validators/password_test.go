package validators

import (
	"strings"
	"testing"
)

func TestValidatePasswordRequirements(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password",
			password: "ValidPass123!@#",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Test1!",
			wantErr:  true,
			errMsg:   "at least 8 characters",
		},
		{
			name:     "no uppercase",
			password: "test123!@#",
			wantErr:  true,
			errMsg:   "uppercase",
		},
		{
			name:     "no lowercase",
			password: "TEST123!@#",
			wantErr:  true,
			errMsg:   "lowercase",
		},
		{
			name:     "no number",
			password: "TestTest!@#",
			wantErr:  true,
			errMsg:   "number",
		},
		{
			name:     "no special char",
			password: "TestTest123",
			wantErr:  true,
			errMsg:   "special character",
		},
		{
			name:     "all requirements met",
			password: "MySecure123!Pass",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePasswordRequirements(tt.password)

			if result.IsValid == tt.wantErr {
				t.Errorf("ValidatePasswordRequirements() isValid = %v, wantErr %v", result.IsValid, tt.wantErr)
			}

			if tt.wantErr && len(result.Errors) > 0 {
				hasExpectedError := false
				for _, err := range result.Errors {
					if strings.Contains(strings.ToLower(err), strings.ToLower(tt.errMsg)) {
						hasExpectedError = true
						break
					}
				}
				if !hasExpectedError {
					t.Errorf("Expected error containing '%s', got errors: %v", tt.errMsg, result.Errors)
				}
			}

			if !tt.wantErr && len(result.Errors) > 0 {
				t.Errorf("Expected no errors, got: %v", result.Errors)
			}
		})
	}
}

func TestValidatePasswordInput(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
		{
			name:     "too long password",
			password: strings.Repeat("a", 300),
			wantErr:  true,
		},
		{
			name:     "valid length password",
			password: "ValidPass123!@#",
			wantErr:  false,
		},
		{
			name:     "max length password",
			password: strings.Repeat("A1!", 85) + "A", // 256 chars
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordInput(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPasswordRequirements(t *testing.T) {
	// Test individual requirement flags
	tests := []struct {
		name         string
		password     string
		requirements map[string]bool
	}{
		{
			name:     "all requirements",
			password: "ValidPass123!@#",
			requirements: map[string]bool{
				"minLength":      true,
				"hasUppercase":   true,
				"hasLowercase":   true,
				"hasNumber":      true,
				"hasSpecialChar": true,
			},
		},
		{
			name:     "missing uppercase only",
			password: "validpass123!@#",
			requirements: map[string]bool{
				"minLength":      true,
				"hasUppercase":   false,
				"hasLowercase":   true,
				"hasNumber":      true,
				"hasSpecialChar": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePasswordRequirements(tt.password)

			for req, expected := range tt.requirements {
				if actual, ok := result.Requirements[req]; !ok {
					t.Errorf("Requirement %s not found in result", req)
				} else if actual != expected {
					t.Errorf("Requirement %s = %v, want %v", req, actual, expected)
				}
			}
		})
	}
}
