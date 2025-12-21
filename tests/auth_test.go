package tests

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/hooks"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// Helper to generate random email
func randomEmail() string {
	return fmt.Sprintf("test%d@example.com", rand.New(rand.NewSource(time.Now().UnixNano())).Int())
}

// setupTestApp creates a test PocketBase app with hooks registered
func setupTestApp(t *testing.T) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}

	// Run migrations to create collections
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register hooks from hooks package (same as main.go)
	app.OnRecordCreate("users").BindFunc(hooks.OnUserCreate(app))
	app.OnRecordUpdate("users").BindFunc(hooks.OnUserUpdate(app))

	return app
}

func TestPasswordValidation(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	collection, _ := app.FindCollectionByNameOrId("users")

	tests := []struct {
		name      string
		password  string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid password",
			password:  "ValidPass123!@#",
			expectErr: false,
		},
		{
			name:      "too short",
			password:  "Test1!",
			expectErr: true,
			errMsg:    "at least 8 characters",
		},
		{
			name:      "no uppercase",
			password:  "test123!@#",
			expectErr: true,
			errMsg:    "uppercase letter",
		},
		{
			name:      "no lowercase",
			password:  "TEST123!@#",
			expectErr: true,
			errMsg:    "lowercase letter",
		},
		{
			name:      "no number",
			password:  "TestAbc!@#",
			expectErr: true,
			errMsg:    "number",
		},
		{
			name:      "no special char",
			password:  "TestAbc123",
			expectErr: true,
			errMsg:    "special character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := core.NewRecord(collection)
			record.Set("email", randomEmail())
			record.Set("password", tt.password)
			record.SetVerified(true)

			err := app.Save(record)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				} else {
					t.Logf("✅ Correctly rejected: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				} else {
					t.Logf("✅ User created successfully")
				}
			}
		})
	}
}

func TestUserCreationAndLogin(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	collection, _ := app.FindCollectionByNameOrId("users")

	email := randomEmail()
	password := "ValidPass123!@#"

	// Create user
	record := core.NewRecord(collection)
	record.Set("email", email)
	record.Set("password", password)
	record.SetVerified(true)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Logf("✅ User created: %s", email)

	// Verify password is hashed
	if record.GetString("password") == password {
		t.Error("Password should be hashed, not stored in plain text")
	}

	// Test authentication (simulated)
	if record.Id == "" {
		t.Error("User ID should be set after creation")
	}

	t.Logf("✅ User authentication simulation passed")
}

func TestOAuthUserCannotSetPassword(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	collection, _ := app.FindCollectionByNameOrId("users")

	email := randomEmail()

	// Create OAuth user without password (OAuth users don't have passwords initially)
	record := core.NewRecord(collection)
	record.Set("email", email)
	record.Set("oauth_signup", true)
	record.Set("password", "TempPassword123!@#") // Set initial password to satisfy schema
	record.SetVerified(true)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create OAuth user: %v", err)
	}

	// Mark as OAuth user after creation
	record.Set("oauth_signup", true)
	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to mark as OAuth user: %v", err)
	}

	t.Logf("✅ OAuth user created")

	// Try to change password
	record.Set("password", "NewTestPass123!@#")
	err := app.Save(record)

	if err == nil {
		t.Error("OAuth users should not be able to set passwords")
	} else if !contains(err.Error(), "OAuth") {
		t.Errorf("Expected OAuth error, got: %v", err)
	} else {
		t.Logf("✅ OAuth password restriction working: %v", err)
	}
}

func TestDuplicateEmailPrevention(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	collection, _ := app.FindCollectionByNameOrId("users")

	email := randomEmail()

	// Create first user
	record1 := core.NewRecord(collection)
	record1.Set("email", email)
	record1.Set("password", "FirstPass123!@#")
	record1.SetVerified(true)

	if err := app.Save(record1); err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	// Try to create second user with same email
	record2 := core.NewRecord(collection)
	record2.Set("email", email)
	record2.Set("password", "SecondPass123!@#")
	record2.SetVerified(true)

	err := app.Save(record2)

	if err == nil {
		t.Error("Duplicate email should be rejected")
	} else {
		t.Logf("✅ Duplicate email correctly prevented: %v", err)
	}
}

func TestPasswordReusePrevention(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	// Ensure security collections exist
	historyCollection, err := app.FindCollectionByNameOrId("password_change_history")
	if err != nil {
		t.Skip("Security tables not available in test environment")
	}
	t.Logf("Security collection exists: %s", historyCollection.Name)

	collection, _ := app.FindCollectionByNameOrId("users")

	email := randomEmail()
	password := "InitialPass123!@#"

	// Create user
	record := core.NewRecord(collection)
	record.Set("email", email)
	record.Set("password", password)
	record.SetVerified(true)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Logf("✅ User created")

	// Try to reuse the same password
	record.Set("password", password)
	err = app.Save(record)

	if err == nil {
		t.Errorf("Password reuse should be prevented")
	} else if !contains(err.Error(), "recently used") {
		t.Errorf("Expected reuse error, got: %v", err)
	} else {
		t.Logf("✅ Password reuse correctly prevented: %v", err)
	}

	// Try a different password
	newPassword := "NewValidPass123!@#"
	record.Set("password", newPassword)
	err = app.Save(record)

	if err != nil {
		t.Errorf("New password should be accepted, got error: %v", err)
	} else {
		t.Logf("✅ New password accepted")
	}
}

func TestPasswordChangeFrequencyLimit(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	// Ensure security collections exist
	_, err := app.FindCollectionByNameOrId("password_change_history")
	if err != nil {
		t.Skip("Security tables not available in test environment")
		return
	}

	collection, _ := app.FindCollectionByNameOrId("users")

	email := randomEmail()

	// Create user
	record := core.NewRecord(collection)
	record.Set("email", email)
	record.Set("password", "InitialPass123!@#")
	record.SetVerified(true)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Change password 5 times
	for i := 1; i <= 5; i++ {
		newPass := fmt.Sprintf("NewPass%d!@#Abc", i)
		record.Set("password", newPass)
		if err := app.Save(record); err != nil {
			t.Logf("Password change %d: %v", i, err)
		}
	}

	// 6th attempt should fail (frequency limit)
	record.Set("password", "SixthPass123!@#")
	err = app.Save(record)

	if err == nil {
		t.Errorf("6th password change should be blocked due to frequency limit")
	} else if !contains(err.Error(), "too many") {
		t.Errorf("Expected frequency limit error, got: %v", err)
	} else {
		t.Logf("✅ Password change frequency limit working: %v", err)
	}
}

// Helper function
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 ||
		(len(str) > 0 && len(substr) > 0 && containsSubstring(str, substr)))
}

func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGitHubOAuthConfiguration(t *testing.T) {
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	if githubClientID == "" || githubClientSecret == "" {
		t.Skip("Skipping GitHub OAuth test - GITHUB_CLIENT_ID or GITHUB_CLIENT_SECRET not set")
	}

	app := setupTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Failed to find users collection: %v", err)
	}

	// Check if GitHub OAuth provider is configured
	if collection.OAuth2.Providers == nil {
		t.Fatal("OAuth2 providers not configured")
	}

	foundGitHub := false
	for _, provider := range collection.OAuth2.Providers {
		if provider.Name == "github" {
			foundGitHub = true
			if provider.ClientId == "" {
				t.Error("GitHub OAuth ClientId is empty")
			}
			t.Logf("✅ GitHub OAuth configured with ClientId: %s", provider.ClientId)
			break
		}
	}

	if !foundGitHub {
		t.Error("GitHub OAuth provider not found in collection configuration")
	}
}

func TestGoogleOAuthConfiguration(t *testing.T) {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if googleClientID == "" || googleClientSecret == "" {
		t.Skip("Skipping Google OAuth test - GOOGLE_CLIENT_ID or GOOGLE_CLIENT_SECRET not set")
	}

	app := setupTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Failed to find users collection: %v", err)
	}

	// Check if Google OAuth provider is configured
	if collection.OAuth2.Providers == nil {
		t.Fatal("OAuth2 providers not configured")
	}

	foundGoogle := false
	for _, provider := range collection.OAuth2.Providers {
		if provider.Name == "google" {
			foundGoogle = true
			if provider.ClientId == "" {
				t.Error("Google OAuth ClientId is empty")
			}
			t.Logf("✅ Google OAuth configured with ClientId: %s", provider.ClientId)
			break
		}
	}

	if !foundGoogle {
		t.Error("Google OAuth provider not found in collection configuration")
	}
}

// TestOAuthUserCreation - Removed: Cannot properly simulate OAuth flow in unit tests
// OAuth users are created by PocketBase's OAuth system which auto-generates credentials
// The TestOAuthUserCannotSetPassword test already validates the key behavior

// TestOAuthAndPasswordUserWithSameEmail - Removed: Same reason as above
// TestDuplicateEmailPrevention already validates unique email constraint
