package tests

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// createTestUser creates a test user with the given email and password
func createTestUser(app *tests.TestApp, t *testing.T, email, password string) *core.Record {
	usersCollection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Users collection not found: %v", err)
	}

	record := core.NewRecord(usersCollection)
	record.Set("email", email)
	record.Set("password", password)
	record.SetVerified(true)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return record
}

// createTestAgent creates a test agent for the given user
func createTestAgent(app *tests.TestApp, t *testing.T, userID, hostname string) *core.Record {
	agentsCollection, err := app.FindCollectionByNameOrId("agents")
	if err != nil {
		t.Fatalf("Agents collection not found: %v", err)
	}

	// Create a device code first
	deviceCodesCollection, err := app.FindCollectionByNameOrId("device_codes")
	if err != nil {
		t.Fatalf("Device codes collection not found: %v", err)
	}

	deviceCodeRecord := core.NewRecord(deviceCodesCollection)
	deviceCodeRecord.Set("device_code", "test-device-"+time.Now().Format("20060102150405"))
	deviceCodeRecord.Set("user_code", "TEST123")
	deviceCodeRecord.Set("expires_at", time.Now().Add(10*time.Minute))
	if err := app.Save(deviceCodeRecord); err != nil {
		t.Fatalf("Failed to create device code: %v", err)
	}

	// Create agent
	agentRecord := core.NewRecord(agentsCollection)
	agentRecord.Set("user_id", userID)
	agentRecord.Set("hostname", hostname)
	agentRecord.Set("os_type", "linux")
	agentRecord.Set("platform_family", "debian")
	agentRecord.Set("version", "1.0.0")
	agentRecord.Set("device_code_id", deviceCodeRecord.Id)
	agentRecord.SetPassword("testpass123")

	if err := app.Save(agentRecord); err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	return agentRecord
}
