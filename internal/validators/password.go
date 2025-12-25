package validators

import (
	"fmt"
	"regexp"
)

var specialCharRegex = regexp.MustCompile(`[!@#$%^&*()_+\-={}[\];':"\\|,.<>/?]`)

// PasswordValidationResult contains validation results
type PasswordValidationResult struct {
	IsValid      bool
	Errors       []string
	Requirements map[string]bool
}

// ValidatePasswordRequirements validates password strength requirements
// Based on supabase/functions/validate-password/index.ts
func ValidatePasswordRequirements(password string) PasswordValidationResult {
	errors := []string{}
	requirements := map[string]bool{
		"minLength":      len(password) >= 8,
		"hasUppercase":   regexp.MustCompile(`[A-Z]`).MatchString(password),
		"hasLowercase":   regexp.MustCompile(`[a-z]`).MatchString(password),
		"hasNumber":      regexp.MustCompile(`[0-9]`).MatchString(password),
		"hasSpecialChar": specialCharRegex.MatchString(password),
	}

	if !requirements["minLength"] {
		errors = append(errors, "Password must be at least 8 characters long")
	}
	if !requirements["hasUppercase"] {
		errors = append(errors, "Password must contain at least one uppercase letter")
	}
	if !requirements["hasLowercase"] {
		errors = append(errors, "Password must contain at least one lowercase letter")
	}
	if !requirements["hasNumber"] {
		errors = append(errors, "Password must contain at least one number")
	}
	if !requirements["hasSpecialChar"] {
		errors = append(errors, "Password must contain at least one special character (!@#$%^&*)")
	}

	return PasswordValidationResult{
		IsValid:      len(errors) == 0,
		Errors:       errors,
		Requirements: requirements,
	}
}

// ValidatePasswordInput performs basic password input validation
func ValidatePasswordInput(password string) error {
	if password == "" {
		return fmt.Errorf("password is required")
	}
	if len(password) > 256 {
		return fmt.Errorf("password is too long (max 256 characters)")
	}
	return nil
}
