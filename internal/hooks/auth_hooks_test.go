package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestOnUserCreateHookValid(t *testing.T) {
	app, _ := tests.NewTestApp()
	defer app.Cleanup()

	collection, _ := app.FindCollectionByNameOrId("users")
	record := core.NewRecord(collection)
	record.Set("email", "test-"+t.Name()+"@example.com") // Unique email
	record.Set("password", "ValidPass123!@#")
	record.SetVerified(true)

	// Register the hook
	app.OnRecordCreate("users").BindFunc(OnUserCreate(app))

	// Try to save - should succeed with valid password
	err := app.Save(record)
	if err != nil {
		t.Errorf("Expected no error for valid password, got: %v", err)
	} else {
		t.Logf("Valid password accepted")
	}
}

func TestOnUserCreateHookInvalid(t *testing.T) {
	app, _ := tests.NewTestApp()
	defer app.Cleanup()

	collection, _ := app.FindCollectionByNameOrId("users")
	record := core.NewRecord(collection)
	record.Set("email", "test-"+t.Name()+"@example.com") // Unique email
	record.Set("password", "weak")
	record.SetVerified(true)

	// Register the hook
	app.OnRecordCreate("users").BindFunc(OnUserCreate(app))

	// Try to save - should fail with invalid password
	err := app.Save(record)
	if err == nil {
		t.Error("Expected error for invalid password, got nil")
	} else {
		t.Logf("Invalid password rejected: %v", err)
	}
}

func TestOnUserUpdateOAuthRestriction(t *testing.T) {
	app, _ := tests.NewTestApp()
	defer app.Cleanup()

	collection, _ := app.FindCollectionByNameOrId("users")
	record := core.NewRecord(collection)
	record.Set("email", "oauth@example.com")
	record.Set("password", "Initial123!@#")
	record.SetVerified(true)

	// Create user first
	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Mark as OAuth user
	record.Set("oauth_signup", true)
	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to mark as OAuth: %v", err)
	}

	// Register the update hook
	app.OnRecordUpdate("users").BindFunc(OnUserUpdate(app))

	// Try to change password - should fail
	record.Set("password", "NewPass123!@#")
	err := app.Save(record)

	if err == nil {
		t.Error("Expected error for OAuth user setting password, got nil")
	} else if err.Error() != "OAuth users cannot set passwords" {
		t.Errorf("Expected OAuth error message, got: %v", err)
	} else {
		t.Logf("OAuth restriction working: %v", err)
	}
}
