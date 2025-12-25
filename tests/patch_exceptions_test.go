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
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestPatchExceptionsAndScriptResolution(t *testing.T) {
	testApp, _ := tests.NewTestApp()
	defer testApp.Cleanup()

	// Run migrations
	if err := testApp.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register hooks
	hooks.RegisterPatchHooks(testApp)

	// 1. Setup Data
	usersCollection, err := testApp.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Failed to find users collection: %v", err)
	}
	user := core.NewRecord(usersCollection)
	user.Set("email", "test-exceptions@example.com")
	user.Set("password", "Test123456!@#")
	user.SetVerified(true)
	if err := testApp.Save(user); err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	// Create Agent
	agentsCollection, err := testApp.FindCollectionByNameOrId("agents")
	if err != nil {
		t.Fatalf("Failed to find agents collection: %v", err)
	}
	deviceCodesCollection, err := testApp.FindCollectionByNameOrId("device_codes")
	if err != nil {
		t.Fatalf("Failed to find device_codes collection: %v", err)
	}

	deviceCode := core.NewRecord(deviceCodesCollection)
	deviceCode.Set("device_code", "dev-123")
	deviceCode.Set("user_code", "USR123")
	deviceCode.Set("expires_at", time.Now().Add(time.Hour))
	if err := testApp.Save(deviceCode); err != nil {
		t.Fatalf("Failed to save device code: %v", err)
	}

	agent := core.NewRecord(agentsCollection)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", deviceCode.Id)
	agent.Set("hostname", "test-agent")
	agent.Set("platform_family", "debian")
	agent.Set("version", "1.0.0")
	agent.SetPassword("AgentPass123!")
	if err := testApp.Save(agent); err != nil {
		t.Fatalf("Failed to save agent: %v", err)
	}

	// Create Script
	scriptsCollection, _ := testApp.FindCollectionByNameOrId("scripts")
	script := core.NewRecord(scriptsCollection)
	script.Set("name", "update-packages")
	script.Set("platform_family", "debian")

	scriptContent := []byte("#!/bin/bash\necho 'updating...'")
	f, err := filesystem.NewFileFromBytes(scriptContent, "update.sh")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	script.Set("file", f)
	script.Set("sha256", "dummy") // Hook will calculate real one

	if err := testApp.Save(script); err != nil {
		t.Fatalf("Failed to save script: %v", err)
	}

	// Create Package Exception
	pkgExceptionsCollection, err := testApp.FindCollectionByNameOrId("package_exceptions")
	if err != nil {
		t.Fatalf("Failed to find package_exceptions collection: %v", err)
	}

	exception := core.NewRecord(pkgExceptionsCollection)
	exception.Set("agent_id", agent.Id)
	exception.Set("user_id", user.Id)
	exception.Set("package_name", "vim")
	exception.Set("is_active", true)
	if err := testApp.Save(exception); err != nil {
		t.Fatalf("Failed to save exception: %v", err)
	}

	userToken, err := user.NewAuthToken()
	if err != nil {
		t.Fatalf("Failed to generate user token: %v", err)
	}

	// 2. Create Patch Operation via Standard API
	(&tests.ApiScenario{
		Name:   "Create patch operation without script_id",
		Method: "POST",
		URL:    "/api/collections/patch_operations/records",
		Body: strings.NewReader(fmt.Sprintf(`{
"agent_id": "%s",
"mode": "apply",
"status": "pending",
"user_id": "%s"
}`, agent.Id, user.Id)),
		Headers: map[string]string{
			"Authorization": "Bearer " + userToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"id"`,
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			var result map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Verify script_id was populated
			if result["script_id"] != script.Id {
				t.Errorf("Expected script_id %s, got %s", script.Id, result["script_id"])
			}

			// Verify exclusions were populated
			exclusions, ok := result["exclusions"].([]interface{})
			if !ok {
				t.Fatalf("Expected exclusions to be a list, got %T", result["exclusions"])
			}

			found := false
			for _, ex := range exclusions {
				if ex.(string) == "vim" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected 'vim' in exclusions, got %v", exclusions)
			}
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)
}
