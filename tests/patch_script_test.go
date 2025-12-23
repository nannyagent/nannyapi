package tests

import (
	"crypto/sha256"
	"encoding/hex"
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

func ptr(s string) *string {
	return &s
}

func TestPatchScriptRelationAndDownload(t *testing.T) {
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
	hooks.RegisterPatchHooks(app)

	// 1. Create User
	usersCollection, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(usersCollection)
	email := fmt.Sprintf("test_patch_%d@example.com", time.Now().UnixNano())
	user.Set("email", email)
	user.Set("password", "TestPass123!")
	if err := app.Save(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 2. Create Agent
	agentsCollection, _ := app.FindCollectionByNameOrId("agents")
	deviceCodesCollection, _ := app.FindCollectionByNameOrId("device_codes")

	deviceCode := core.NewRecord(deviceCodesCollection)
	deviceCode.Set("device_code", "DEV-PATCH")
	deviceCode.Set("user_code", "USRPATCH") // 8 chars max
	deviceCode.Set("authorized", true)
	deviceCode.Set("user_id", user.Id)
	deviceCode.Set("consumed", false)
	deviceCode.Set("expires_at", time.Now().Add(time.Hour))
	if err := app.Save(deviceCode); err != nil {
		t.Fatalf("Failed to create device code: %v", err)
	}

	agent := core.NewRecord(agentsCollection)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", deviceCode.Id)
	agent.Set("hostname", "test-agent-patch")
	agent.Set("platform_family", "debian")
	agent.Set("os_type", "linux")
	agent.Set("os_version", "22.04")
	agent.Set("version", "1.0.0")
	agent.SetPassword("AgentPass123!")
	if err := app.Save(agent); err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 3. Create a dummy script in scripts collection
	scriptsCollection, _ := app.FindCollectionByNameOrId("scripts")
	scriptContent := []byte("#!/bin/bash\necho 'Hello Patch'\n")
	scriptHash := sha256.Sum256(scriptContent)
	scriptHashStr := hex.EncodeToString(scriptHash[:])

	file, err := filesystem.NewFileFromBytes(scriptContent, "test-script.sh")
	if err != nil {
		t.Fatal(err)
	}

	script := core.NewRecord(scriptsCollection)
	script.Set("name", "test-script.sh")
	script.Set("platform_family", "debian")
	script.Set("os_type", "linux")
	script.Set("file", file)
	script.Set("sha256", scriptHashStr)
	if err := app.Save(script); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	// 4. Create Patch Operation linking to the script
	patchOpsCollection, _ := app.FindCollectionByNameOrId("patch_operations")
	patchOp := core.NewRecord(patchOpsCollection)
	patchOp.Set("user_id", user.Id)
	patchOp.Set("agent_id", agent.Id)
	patchOp.Set("mode", "dry-run")
	patchOp.Set("status", "pending")
	patchOp.Set("script_id", script.Id)
	// Simulate what the handler does
	scriptURL := fmt.Sprintf("/api/files/%s/%s/%s", scriptsCollection.Id, script.Id, script.GetString("file"))
	patchOp.Set("script_url", scriptURL)

	if err := app.Save(patchOp); err != nil {
		t.Fatalf("Failed to create patch operation: %v", err)
	}

	// 5. Verify Relation and Download
	// Fetch the patch operation
	record, err := app.FindRecordById(patchOpsCollection.Id, patchOp.Id)
	if err != nil {
		t.Fatalf("Failed to fetch patch op: %v", err)
	}

	// Verify script_id is set correctly
	if record.GetString("script_id") != script.Id {
		t.Errorf("Expected script_id %s, got %s", script.Id, record.GetString("script_id"))
	}

	// Verify script_url is populated
	savedScriptURL := record.GetString("script_url")
	if savedScriptURL == "" {
		t.Errorf("script_url is empty in the record")
	}
	fmt.Printf("Patch Operation JSON: %v\n", record)
	fmt.Printf("Script URL: %s\n", savedScriptURL)

	// Now try to download
	agentToken, _ := agent.NewAuthToken()

	// Use the URL from the record
	fileURL := savedScriptURL

	(&tests.ApiScenario{
		Name:   "Download script file and verify content",
		Method: http.MethodGet,
		URL:    fileURL,
		Headers: map[string]string{
			"Authorization": agentToken,
		},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			"#!/bin/bash",
			"echo 'Hello Patch'",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
	}).Test(t)
}

func TestCreatePatchOperationViaAPI(t *testing.T) {
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
	hooks.RegisterPatchHooks(app)

	// 1. Create User
	usersCollection, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(usersCollection)
	user.Set("email", "test_api@example.com")
	user.Set("password", "TestPass123!")
	if err := app.Save(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 2. Create Agent
	agentsCollection, _ := app.FindCollectionByNameOrId("agents")
	deviceCodesCollection, _ := app.FindCollectionByNameOrId("device_codes")

	deviceCode := core.NewRecord(deviceCodesCollection)
	deviceCode.Set("device_code", "DEV-API")
	deviceCode.Set("user_code", "USRAPI")
	deviceCode.Set("authorized", true)
	deviceCode.Set("user_id", user.Id)
	deviceCode.Set("consumed", false)
	deviceCode.Set("expires_at", time.Now().Add(time.Hour))
	if err := app.Save(deviceCode); err != nil {
		t.Fatalf("Failed to create device code: %v", err)
	}

	agent := core.NewRecord(agentsCollection)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", deviceCode.Id)
	agent.Set("hostname", "test-agent-api")
	agent.Set("platform_family", "debian")
	agent.Set("os_type", "linux")
	agent.Set("os_version", "22.04")
	agent.Set("version", "1.0.0")
	agent.SetPassword("AgentPass123!")
	if err := app.Save(agent); err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 3. Create Script
	scriptsCollection, _ := app.FindCollectionByNameOrId("scripts")
	scriptContent := []byte("#!/bin/bash\necho 'Hello API'\n")
	file, _ := filesystem.NewFileFromBytes(scriptContent, "api-script.sh")
	script := core.NewRecord(scriptsCollection)
	script.Set("name", "api-script.sh")
	script.Set("platform_family", "debian")
	script.Set("os_type", "linux")
	script.Set("file", file)
	script.Set("sha256", "dummyhash")
	if err := app.Save(script); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	// 4. Call API to create patch operation
	userToken, _ := user.NewAuthToken()

	(&tests.ApiScenario{
		Name:   "Create patch operation via API",
		Method: http.MethodPost,
		URL:    "/api/patches",
		Headers: map[string]string{
			"Authorization": userToken,
		},
		Body:           strings.NewReader(fmt.Sprintf("{\"agent_id\": \"%s\", \"mode\": \"dry-run\"}", agent.Id)),
		ExpectedStatus: 201,
		ExpectedContent: []string{
			"\"script_url\":", // Check if script_url is present in JSON response
			"/api/files/",     // Check if it looks like a file path
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
	}).Test(t)
}
