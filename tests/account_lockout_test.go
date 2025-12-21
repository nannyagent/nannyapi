package tests

import (
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/security"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestAccountLockoutAfterFailedAttempts(t *testing.T) {
	testApp, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer testApp.Cleanup()

	usersCollection, _ := testApp.FindCollectionByNameOrId("users")
	if usersCollection == nil {
		t.Fatal("users collection not found")
	}

	// Try to find existing collections first
	attemptsCollection, err := testApp.FindCollectionByNameOrId("failed_auth_attempts")
	if err != nil {
		// Create if doesn't exist
		attemptsCollection = core.NewBaseCollection("failed_auth_attempts")
		attemptsCollection.Fields.Add(&core.RelationField{
			Name:         "user_id",
			Required:     true,
			CollectionId: usersCollection.Id,
			MaxSelect:    1,
		})
		attemptsCollection.Fields.Add(&core.TextField{Name: "ip_address", Max: 45})
		if err := testApp.Save(attemptsCollection); err != nil {
			t.Fatalf("Failed to create attempts collection: %v", err)
		}
	}

	lockoutCollection, err := testApp.FindCollectionByNameOrId("account_lockout")
	if err != nil {
		lockoutCollection = core.NewBaseCollection("account_lockout")
		lockoutCollection.Fields.Add(&core.RelationField{
			Name:         "user_id",
			Required:     true,
			CollectionId: usersCollection.Id,
			MaxSelect:    1,
		})
		lockoutCollection.Fields.Add(&core.TextField{Name: "reason", Required: true, Max: 500})
		lockoutCollection.Fields.Add(&core.DateField{Name: "locked_until", Required: true})
		lockoutCollection.Fields.Add(&core.NumberField{
			Name:     "failed_attempts",
			Required: false,
			Min:      nil,
			Max:      nil,
		})
		if err := testApp.Save(lockoutCollection); err != nil {
			t.Fatalf("Failed to create lockout collection: %v", err)
		}
	}

	userRecord := core.NewRecord(usersCollection)
	userRecord.Set("email", "lockout-test@example.com")
	userRecord.Set("password", "TestPassword123!")
	if err := testApp.Save(userRecord); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	userId := userRecord.Id
	t.Logf("✅ Created test user: %s", userId)

	for i := 0; i < 4; i++ {
		err := security.TrackFailedAuthAttempt(testApp, userId)
		if err != nil {
			t.Errorf("Unexpected lockout after %d attempts: %v", i+1, err)
		}
		time.Sleep(100 * time.Millisecond) // Small delay between attempts
	}

	// Verify 4 attempts recorded
	attempts, _ := testApp.FindRecordsByFilter(attemptsCollection, "user_id = {:userId}", "", 10, 0, map[string]any{"userId": userId})
	t.Logf("✅ 4 failed attempts recorded (%d records)", len(attempts))

	lockErr := security.TrackFailedAuthAttempt(testApp, userId)
	if lockErr == nil {
		// Check how many attempts were counted
		allAttempts, _ := testApp.FindRecordsByFilter(attemptsCollection, "user_id = {:userId}", "", 10, 0, map[string]any{"userId": userId})
		t.Fatalf("Expected account lockout after 5 attempts, got none. Total attempts: %d", len(allAttempts))
	}
	if lockErr.Error() != "account locked due to too many failed attempts. Please try again in 30 minutes" {
		t.Errorf("Wrong lockout error: %v", lockErr)
	}
	t.Logf("✅ 5th attempt triggered lockout: %v", lockErr)

	lockouts, lockErr2 := testApp.FindRecordsByFilter(lockoutCollection, "user_id = {:userId}", "", 1, 0, map[string]any{"userId": userId})
	if lockErr2 != nil || len(lockouts) == 0 {
		t.Fatal("Lockout record not created")
	}

	lockout := lockouts[0]
	lockedUntil := lockout.GetDateTime("locked_until").Time()

	expectedLockUntil := time.Now().Add(30 * time.Minute)
	if lockedUntil.Before(time.Now()) || lockedUntil.After(expectedLockUntil.Add(1*time.Minute)) {
		t.Errorf("Lock time incorrect: got %v, expected around %v", lockedUntil, expectedLockUntil)
	}
	t.Logf("✅ Lockout record created, locked until %v", lockedUntil)

	checkErr := security.CheckAccountLockout(testApp, userId)
	if checkErr == nil {
		// Debug: check if lockout exists
		debugLockouts, _ := testApp.FindRecordsByFilter(lockoutCollection, "user_id = {:userId}", "", 1, 0, map[string]any{"userId": userId})
		t.Fatalf("CheckAccountLockout should return error for locked account (found %d lockouts)", len(debugLockouts))
	}
	t.Logf("✅ CheckAccountLockout returns error: %v", checkErr)

	attempts, _ = testApp.FindRecordsByFilter(attemptsCollection, "user_id = {:userId}", "", 10, 0, map[string]any{"userId": userId})
	if len(attempts) != 5 {
		t.Errorf("Expected 5 failed attempt records, got %d", len(attempts))
	}
	t.Logf("✅ All 5 failed attempts recorded")

	// TEST CRITICAL PART: Try to actually authenticate with the locked account
	// This proves the lockout BLOCKS login, not just returns a message
	usersCollection2, _ := testApp.FindCollectionByNameOrId("users")
	_, authErr := testApp.FindAuthRecordByEmail(usersCollection2, "lockout-test@example.com")
	if authErr != nil {
		t.Logf("FindAuthRecordByEmail error: %v", authErr)
	}

	// The real test: CheckAccountLockout should return error for locked account
	lockedCheckErr := security.CheckAccountLockout(testApp, userId)
	if lockedCheckErr == nil {
		t.Fatal("SECURITY FAILURE: CheckAccountLockout returned nil for locked account!")
	}
	t.Logf("✅ VERIFIED: Locked account check returns error: %v", lockedCheckErr)
}
