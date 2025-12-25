package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/hooks"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestAgentRegistrationPlatformFallback(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Run migrations
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register hooks
	hooks.RegisterAgentHooks(app)

	// 1. Create User
	usersCollection, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(usersCollection)
	email := fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())
	user.Set("email", email)
	user.Set("password", "TestPass123!")
	if err := app.Save(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 2. Create and Authorize Device Code
	deviceCodesCollection, _ := app.FindCollectionByNameOrId("device_codes")
	deviceCode := core.NewRecord(deviceCodesCollection)
	code := "DEV-123"
	deviceCode.Set("device_code", code)
	deviceCode.Set("user_code", "USR123")
	deviceCode.Set("authorized", true)
	deviceCode.Set("user_id", user.Id)
	deviceCode.Set("consumed", false)
	deviceCode.Set("expires_at", time.Now().Add(time.Hour))
	if err := app.Save(deviceCode); err != nil {
		t.Fatalf("Failed to create device code: %v", err)
	}

	// 3. Test Registration with missing platform_family but valid OSInfo
	reqBody := map[string]interface{}{
		"action":      "register",
		"device_code": code,
		"hostname":    "test-agent",
		"os_type":     "linux",
		"os_info":     "Ubuntu 22.04 LTS",
		"os_version":  "22.04",
		"version":     "1.0.0",
		"arch":        "amd64",
		// platform_family is intentionally missing
	}
	jsonBody, _ := json.Marshal(reqBody)

	testName := "Agent registration with platform fallback"

	(&tests.ApiScenario{
		Name:           testName,
		Method:         http.MethodPost,
		URL:            "/api/agent",
		Body:           strings.NewReader(string(jsonBody)),
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"access_token"`,
			`"agent_id"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify agent record in DB
	agentsCollection, _ := app.FindCollectionByNameOrId("agents")
	record, err := app.FindFirstRecordByData(agentsCollection, "hostname", "test-agent")
	if err != nil {
		t.Fatalf("Failed to find created agent: %v", err)
	}

	platformFamily := record.GetString("platform_family")
	if platformFamily != "debian" {
		t.Errorf("Expected platform_family 'debian' for Ubuntu, got '%s'", platformFamily)
	}
}

func TestAgentRegistrationPlatformFallbackRHEL(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Run migrations
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register hooks
	hooks.RegisterAgentHooks(app)

	// 1. Create User
	usersCollection, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(usersCollection)
	email := fmt.Sprintf("test_rhel_%d@example.com", time.Now().UnixNano())
	user.Set("email", email)
	user.Set("password", "TestPass123!")
	if err := app.Save(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 2. Create and Authorize Device Code
	deviceCodesCollection, _ := app.FindCollectionByNameOrId("device_codes")
	deviceCode := core.NewRecord(deviceCodesCollection)
	code := "DEV-RHEL"
	deviceCode.Set("device_code", code)
	deviceCode.Set("user_code", "USR-RHEL")
	deviceCode.Set("authorized", true)
	deviceCode.Set("user_id", user.Id)
	deviceCode.Set("consumed", false)
	deviceCode.Set("expires_at", time.Now().Add(time.Hour))
	if err := app.Save(deviceCode); err != nil {
		t.Fatalf("Failed to create device code: %v", err)
	}

	// 3. Test Registration with missing platform_family but valid OSInfo for RHEL
	reqBody := map[string]interface{}{
		"action":      "register",
		"device_code": code,
		"hostname":    "test-agent-rhel",
		"os_type":     "linux",
		"os_info":     "CentOS Linux 7 (Core)",
		"os_version":  "7",
		"version":     "1.0.0",
		"arch":        "amd64",
	}
	jsonBody, _ := json.Marshal(reqBody)

	(&tests.ApiScenario{
		Name:           "Agent registration with RHEL fallback",
		Method:         http.MethodPost,
		URL:            "/api/agent",
		Body:           strings.NewReader(string(jsonBody)),
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"access_token"`,
			`"agent_id"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify agent record in DB
	agentsCollection, _ := app.FindCollectionByNameOrId("agents")
	record, err := app.FindFirstRecordByData(agentsCollection, "hostname", "test-agent-rhel")
	if err != nil {
		t.Fatalf("Failed to find created agent: %v", err)
	}

	platformFamily := record.GetString("platform_family")
	if platformFamily != "rhel" {
		t.Errorf("Expected platform_family 'rhel' for CentOS, got '%s'", platformFamily)
	}
}
