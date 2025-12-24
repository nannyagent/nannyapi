package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
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

func TestPatchEndToEndFlow(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-patch")

	testApp, _ := tests.NewTestApp()
	defer testApp.Cleanup()

	// Run migrations
	if err := testApp.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register hooks
	hooks.RegisterInvestigationHooks(testApp)
	hooks.RegisterPatchHooks(testApp)
	hooks.RegisterAgentHooks(testApp)

	// 1. Setup Data
	usersCollection, err := testApp.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Failed to find users collection: %v", err)
	}
	user := core.NewRecord(usersCollection)
	user.Set("email", "patch-test@example.com")
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
	deviceCode.Set("device_code", "dev-patch-123")
	deviceCode.Set("user_code", "USRPATCH")
	deviceCode.Set("expires_at", time.Now().Add(time.Hour))
	if err := testApp.Save(deviceCode); err != nil {
		t.Fatalf("Failed to save device code: %v", err)
	}

	agent := core.NewRecord(agentsCollection)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", deviceCode.Id)
	agent.Set("hostname", "patch-agent")
	agent.Set("os_type", "linux")
	agent.Set("platform_family", "debian")
	agent.Set("os_version", "ubuntu-22.04")
	agent.Set("version", "1.0.0")
	agent.SetPassword("AgentPass123!")
	if err := testApp.Save(agent); err != nil {
		t.Fatalf("Failed to save agent: %v", err)
	}

	agentToken, err := agent.NewAuthToken()
	if err != nil {
		t.Fatalf("Failed to generate agent token: %v", err)
	}

	userToken, err := user.NewAuthToken()
	if err != nil {
		t.Fatalf("Failed to generate user token: %v", err)
	}

	// Create Script
	scriptsCollection, _ := testApp.FindCollectionByNameOrId("scripts")
	script := core.NewRecord(scriptsCollection)
	script.Set("name", "update-packages")
	script.Set("platform_family", "debian")
	script.Set("os_type", "linux")
	script.Set("os_version", "ubuntu-22.04")

	scriptContent := []byte("#!/bin/bash\necho 'updating...'")
	h := sha256.New()
	h.Write(scriptContent)
	expectedHash := hex.EncodeToString(h.Sum(nil))

	script.Set("sha256", expectedHash)
	f, _ := filesystem.NewFileFromBytes(scriptContent, "script.sh")
	script.Set("file", f)
	testApp.Save(script)

	// --- Scenario 1: Agent reports metrics (Prerequisite) ---
	(&tests.ApiScenario{
		Name:   "Agent reports metrics",
		Method: "POST",
		URL:    "/api/agent",
		Body: strings.NewReader(`{
			"action": "ingest-metrics",
			"platform_family": "debian",
			"system_metrics": {
				"cpu_percent": 10.5,
				"filesystems": []
			}
		}`),
		Headers: map[string]string{
			"Authorization": "Bearer " + agentToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"success":true`,
			`"message":"metrics recorded"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// --- Scenario 2: User initiates patch (Dry Run) ---
	var patchID string
	(&tests.ApiScenario{
		Name:   "User initiates patch",
		Method: "POST",
		URL:    "/api/patches",
		Body: strings.NewReader(fmt.Sprintf(`{
			"agent_id": "%s",
			"mode": "dry-run"
		}`, agent.Id)),
		Headers: map[string]string{
			"Authorization": "Bearer " + userToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus: 201,
		ExpectedContent: []string{
			`"status":"pending"`,
			fmt.Sprintf(`"script_sha256":"%s"`, expectedHash),
			`"mode":"dry-run"`,
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			// Extract Patch ID
			var resp map[string]interface{}
			json.NewDecoder(res.Body).Decode(&resp)
			patchID = resp["id"].(string)
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// --- Scenario 3: Agent validates script ---
	(&tests.ApiScenario{
		Name:   "Agent validates script",
		Method: "GET",
		URL:    fmt.Sprintf("/api/scripts/%s/validate", script.Id),
		Headers: map[string]string{
			"Authorization": "Bearer " + agentToken,
		},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			fmt.Sprintf(`"sha256":"%s"`, expectedHash),
			`"name":"update-packages"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// --- Scenario 4: Agent uploads results (Success) ---
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("exit_code", "0")

	part, _ := writer.CreateFormFile("stdout_file", "stdout.log")
	part.Write([]byte("Dry run successful\nPackage A: 1.0 -> 1.1"))

	part, _ = writer.CreateFormFile("stderr_file", "stderr.log")
	part.Write([]byte(""))

	writer.Close()

	(&tests.ApiScenario{
		Name:   "Agent uploads results (Success)",
		Method: "POST",
		URL:    fmt.Sprintf("/api/patches/%s/result", patchID),
		Body:   body,
		Headers: map[string]string{
			"Authorization": "Bearer " + agentToken,
			"Content-Type":  writer.FormDataContentType(),
		},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"status":"updated"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify DB State
	patchRecord, err := testApp.FindRecordById("patch_operations", patchID)
	if err != nil {
		t.Fatalf("Failed to find patch record: %v", err)
	}
	if patchRecord.GetString("status") != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", patchRecord.GetString("status"))
	}
	if patchRecord.GetString("stdout_file") == "" {
		t.Error("Expected stdout_file to be set")
	}

	// --- Scenario 5: User initiates patch (Apply) ---
	var applyPatchID string
	(&tests.ApiScenario{
		Name:   "User initiates patch (Apply)",
		Method: "POST",
		URL:    "/api/patches",
		Body: strings.NewReader(fmt.Sprintf(`{
			"agent_id": "%s",
			"mode": "apply"
		}`, agent.Id)),
		Headers: map[string]string{
			"Authorization": "Bearer " + userToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus: 201,
		ExpectedContent: []string{
			`"status":"pending"`,
			`"mode":"apply"`,
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			var resp map[string]interface{}
			json.NewDecoder(res.Body).Decode(&resp)
			applyPatchID = resp["id"].(string)
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// --- Scenario 6: Agent uploads results (Failure) ---
	bodyFail := &bytes.Buffer{}
	writerFail := multipart.NewWriter(bodyFail)
	writerFail.WriteField("exit_code", "1")

	partFail, _ := writerFail.CreateFormFile("stdout_file", "stdout.log")
	partFail.Write([]byte("Applying update..."))

	partFail, _ = writerFail.CreateFormFile("stderr_file", "stderr.log")
	partFail.Write([]byte("Error: Package conflict"))

	writerFail.Close()

	(&tests.ApiScenario{
		Name:   "Agent uploads results (Failure)",
		Method: "POST",
		URL:    fmt.Sprintf("/api/patches/%s/result", applyPatchID),
		Body:   bodyFail,
		Headers: map[string]string{
			"Authorization": "Bearer " + agentToken,
			"Content-Type":  writerFail.FormDataContentType(),
		},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"status":"updated"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify DB State for Failure
	patchRecordFail, err := testApp.FindRecordById("patch_operations", applyPatchID)
	if err != nil {
		t.Fatalf("Failed to find patch record: %v", err)
	}
	if patchRecordFail.GetString("status") != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", patchRecordFail.GetString("status"))
	}
	if patchRecordFail.GetString("stderr_file") == "" {
		t.Error("Expected stderr_file to be set")
	}

	// --- Scenario 7: Script Not Found (Exception) ---
	(&tests.ApiScenario{
		Name:   "Agent validates non-existent script",
		Method: "GET",
		URL:    "/api/scripts/non-existent-id/validate",
		Headers: map[string]string{
			"Authorization": "Bearer " + agentToken,
		},
		ExpectedStatus: 404,
		ExpectedContent: []string{
			`"error":"script not found"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// --- Scenario 8: Unauthorized Upload (Exception) ---
	// Create another agent
	agent2 := core.NewRecord(agentsCollection)
	agent2.Set("user_id", user.Id)
	agent2.Set("device_code_id", deviceCode.Id) // Reusing device code for simplicity in test setup
	agent2.Set("hostname", "agent-2")
	agent2.Set("os_type", "linux")
	agent2.Set("platform_family", "debian")
	agent2.Set("version", "1.0.0")
	agent2.SetPassword("AgentPass123!")
	if err := testApp.Save(agent2); err != nil {
		t.Fatalf("Failed to save agent2: %v", err)
	}
	agent2Token, _ := agent2.NewAuthToken()

	(&tests.ApiScenario{
		Name:   "Unauthorized agent uploads results",
		Method: "POST",
		URL:    fmt.Sprintf("/api/patches/%s/result", patchID), // patchID belongs to agent 1
		Body:   body,                                           // Reusing body
		Headers: map[string]string{
			"Authorization": "Bearer " + agent2Token,
			"Content-Type":  writer.FormDataContentType(),
		},
		ExpectedStatus: 403,
		ExpectedContent: []string{
			`"error":"unauthorized: operation does not belong to agent"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)
}
