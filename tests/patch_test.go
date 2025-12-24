package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/hooks"
	"github.com/nannyagent/nannyapi/internal/patches"
	"github.com/nannyagent/nannyapi/internal/types"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func setupPatchPrerequisites(app core.App, t *testing.T, agentID string) {
	// Update Agent with OS info
	agentsCollection, err := app.FindCollectionByNameOrId("agents")
	if err != nil {
		t.Fatalf("Failed to find agents collection: %v", err)
	}
	agent, err := app.FindRecordById(agentsCollection, agentID)
	if err != nil {
		t.Fatalf("Failed to find agent: %v", err)
	}
	agent.Set("os_type", "linux")
	agent.Set("os_version", "ubuntu-22.04")
	if err := app.Save(agent); err != nil {
		t.Fatalf("Failed to update agent OS info: %v", err)
	}

	// Create Script
	scriptsCollection, err := app.FindCollectionByNameOrId("scripts")
	if err != nil {
		t.Fatalf("Failed to find scripts collection: %v", err)
	}
	script := core.NewRecord(scriptsCollection)
	script.Set("name", "update-packages")
	script.Set("platform_family", "debian")
	script.Set("os_type", "linux")
	script.Set("os_version", "ubuntu-22.04")

	scriptContent := "#!/bin/bash\necho 'updating...'"
	// Calculate SHA256 of the script content
	h := sha256.New()
	h.Write([]byte(scriptContent))
	sha256Hash := hex.EncodeToString(h.Sum(nil))

	script.Set("sha256", sha256Hash)
	f, _ := filesystem.NewFileFromBytes([]byte(scriptContent), "script.sh")
	script.Set("file", f)
	if err := app.Save(script); err != nil {
		t.Fatalf("Failed to save script: %v", err)
	}

	// Create Agent Metrics
	metricsCollection, err := app.FindCollectionByNameOrId("agent_metrics")
	if err != nil {
		t.Fatalf("Failed to find agent_metrics collection: %v", err)
	}
	metric := core.NewRecord(metricsCollection)
	metric.Set("agent_id", agentID)
	metric.Set("recorded_at", time.Now())
	if err := app.Save(metric); err != nil {
		t.Fatalf("Failed to save agent metrics: %v", err)
	}
}

// TestCreatePatchOperation tests patch operation creation
func TestCreatePatchOperation(t *testing.T) {
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
	app.OnRecordCreate("users").BindFunc(hooks.OnUserCreate(app))
	app.OnRecordUpdate("users").BindFunc(hooks.OnUserUpdate(app))

	// Create test user and agent
	user := createTestUser(app, t, "patch.create@example.com", "Password123!@#")
	userID := user.Id
	agent := createTestAgent(app, t, userID, "test-hostname")
	agentID := agent.Id

	setupPatchPrerequisites(app, t, agentID)

	tests := []struct {
		name      string
		userID    string
		request   types.PatchRequest
		expectErr bool
		errMsg    string
	}{
		{
			name:   "Create patch with dry-run mode",
			userID: userID,
			request: types.PatchRequest{
				AgentID: agentID,
				Mode:    "dry-run",
			},
			expectErr: false,
		},
		{
			name:   "Create patch with apply mode",
			userID: userID,
			request: types.PatchRequest{
				AgentID: agentID,
				Mode:    "apply",
			},
			expectErr: false,
		},
		{
			name:   "Fail with invalid mode",
			userID: userID,
			request: types.PatchRequest{
				AgentID: agentID,
				Mode:    "invalid",
			},
			expectErr: true,
		},
		{
			name:   "Fail with missing agent_id",
			userID: userID,
			request: types.PatchRequest{
				Mode: "dry-run",
			},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := patches.CreatePatchOperation(app, tc.userID, tc.request)

			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Fatal("Response is nil")
			}

			if resp.Mode != types.PatchMode(tc.request.Mode) {
				t.Errorf("Mode mismatch: expected %s, got %s", tc.request.Mode, resp.Mode)
			}
			if resp.Status != types.PatchStatusPending {
				t.Errorf("Status should be pending, got %s", resp.Status)
			}
			if resp.ScriptURL == "" {
				t.Error("Expected ScriptURL to be set")
			}
			if resp.ScriptSHA256 == "" {
				t.Error("Expected ScriptSHA256 to be set")
			}

			// Verify record in DB
			patchCollection, _ := app.FindCollectionByNameOrId("patch_operations")
			record, err := app.FindRecordById(patchCollection, resp.ID)
			if err != nil {
				t.Fatalf("Failed to find patch record: %v", err)
			}
			if record.GetString("script_url") == "" {
				t.Error("script_url is empty in DB record")
			}
			if record.GetString("script_url") != resp.ScriptURL {
				t.Errorf("DB script_url mismatch: expected %s, got %s", resp.ScriptURL, record.GetString("script_url"))
			}
		})
	}
}

// TestScriptChecksum verifies that the stored SHA256 matches the file content
func TestScriptChecksum(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Create scripts collection if not exists
	scriptsCollection, err := app.FindCollectionByNameOrId("scripts")
	if err != nil {
		scriptsCollection = core.NewBaseCollection("scripts")
		scriptsCollection.Fields.Add(&core.TextField{Name: "name", Required: true})
		scriptsCollection.Fields.Add(&core.FileField{Name: "file", Required: true, MaxSelect: 1, MaxSize: 1024 * 1024})
		scriptsCollection.Fields.Add(&core.TextField{Name: "sha256", Required: true})
		if err := app.Save(scriptsCollection); err != nil {
			t.Fatalf("Failed to create scripts collection: %v", err)
		}
	}

	scriptContent := "#!/bin/bash\necho 'checksum test'"

	// Calculate expected hash
	h := sha256.New()
	h.Write([]byte(scriptContent))
	expectedHash := hex.EncodeToString(h.Sum(nil))

	// Create script record (simulating migration)
	script := core.NewRecord(scriptsCollection)
	script.Set("name", "test-script")
	script.Set("platform_family", "linux") // Required field
	script.Set("sha256", expectedHash)
	f, _ := filesystem.NewFileFromBytes([]byte(scriptContent), "script.sh")
	script.Set("file", f)

	if err := app.Save(script); err != nil {
		t.Fatalf("Failed to save script: %v", err)
	}

	// Retrieve record
	retrievedScript, err := app.FindRecordById(scriptsCollection, script.Id)
	if err != nil {
		t.Fatalf("Failed to retrieve script: %v", err)
	}

	if retrievedScript.GetString("sha256") != expectedHash {
		t.Errorf("Hash mismatch in DB: expected %s, got %s", expectedHash, retrievedScript.GetString("sha256"))
	}

	// Verify file content in storage
	fs, err := app.NewFilesystem()
	if err != nil {
		t.Fatalf("Failed to get filesystem: %v", err)
	}
	defer fs.Close()

	path := retrievedScript.BaseFilesPath() + "/" + retrievedScript.GetString("file")
	fileReader, err := fs.GetFile(path)
	if err != nil {
		t.Fatalf("Failed to read file from storage: %v", err)
	}
	defer fileReader.Close()

	h2 := sha256.New()
	if _, err := io.Copy(h2, fileReader); err != nil {
		t.Fatalf("Failed to hash stored file: %v", err)
	}
	storedHash := hex.EncodeToString(h2.Sum(nil))

	if storedHash != expectedHash {
		t.Errorf("Stored file hash mismatch: expected %s, got %s", expectedHash, storedHash)
	}
}

// TestPatchHook verifies that script_url is populated by hook
func TestPatchHook(t *testing.T) {
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
	hooks.RegisterPatchHooks(app)

	// Create user and agent
	user := createTestUser(app, t, "patch.hook@example.com", "Password123!@#")
	agent := createTestAgent(app, t, user.Id, "hook-agent")

	// Create script
	scriptsCollection, _ := app.FindCollectionByNameOrId("scripts")
	script := core.NewRecord(scriptsCollection)
	script.Set("name", "hook-script")
	script.Set("platform_family", "linux")
	script.Set("sha256", "hash")
	f, _ := filesystem.NewFileFromBytes([]byte("echo hook"), "hook.sh")
	script.Set("file", f)
	app.Save(script)

	// Create patch operation directly (bypassing handler)
	patchCollection, _ := app.FindCollectionByNameOrId("patch_operations")
	patch := core.NewRecord(patchCollection)
	patch.Set("user_id", user.Id)
	patch.Set("agent_id", agent.Id)
	patch.Set("mode", "dry-run")
	patch.Set("status", "pending")
	patch.Set("script_id", script.Id)
	// script_url is NOT set

	if err := app.Save(patch); err != nil {
		t.Fatalf("Failed to save patch: %v", err)
	}

	// Verify script_url is populated
	retrieved, _ := app.FindRecordById(patchCollection, patch.Id)
	if retrieved.GetString("script_url") == "" {
		t.Error("script_url was not populated by hook")
	}

	expectedURL := fmt.Sprintf("/api/files/%s/%s/%s", scriptsCollection.Id, script.Id, script.GetString("file"))
	if retrieved.GetString("script_url") != expectedURL {
		t.Errorf("script_url mismatch: expected %s, got %s", expectedURL, retrieved.GetString("script_url"))
	}
}

// TestGetPatchOperations tests listing patch operations
func TestGetPatchOperations(t *testing.T) {
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
	app.OnRecordCreate("users").BindFunc(hooks.OnUserCreate(app))
	app.OnRecordUpdate("users").BindFunc(hooks.OnUserUpdate(app))

	// Create test user and agent
	user := createTestUser(app, t, "patch.list@example.com", "Password123!@#")
	userID := user.Id
	agent := createTestAgent(app, t, userID, "test-hostname")
	agentID := agent.Id

	setupPatchPrerequisites(app, t, agentID)

	// Create multiple patch operations
	for i := 0; i < 2; i++ {
		_, err := patches.CreatePatchOperation(app, userID, types.PatchRequest{
			AgentID: agentID,
			Mode:    "dry-run",
		})
		if err != nil {
			t.Fatalf("Failed to create patch operation: %v", err)
		}
	}

	// Get all patch operations
	opsList, err := patches.GetPatchOperations(app, userID)
	if err != nil {
		t.Fatalf("Failed to list patch operations: %v", err)
	}

	if len(opsList) != 2 {
		t.Errorf("expected 2 patch operations, got %d", len(opsList))
	}
}

// TestGetPatchOperation tests retrieving a single patch operation
func TestGetPatchOperation(t *testing.T) {
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
	app.OnRecordCreate("users").BindFunc(hooks.OnUserCreate(app))
	app.OnRecordUpdate("users").BindFunc(hooks.OnUserUpdate(app))

	// Create test user and agent
	user := createTestUser(app, t, "patch.get@example.com", "Password123!@#")
	userID := user.Id
	agent := createTestAgent(app, t, userID, "test-hostname")
	agentID := agent.Id

	setupPatchPrerequisites(app, t, agentID)

	// Create patch operation
	created, err := patches.CreatePatchOperation(app, userID, types.PatchRequest{
		AgentID: agentID,
		Mode:    "apply",
	})
	if err != nil {
		t.Fatalf("Failed to create patch operation: %v", err)
	}

	// Retrieve it
	retrieved, err := patches.GetPatchOperation(app, userID, created.ID)
	if err != nil {
		t.Fatalf("Failed to get patch operation: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: expected %s, got %s", created.ID, retrieved.ID)
	}

	// Test unauthorized access
	otherUser := createTestUser(app, t, "patch.other@example.com", "Password123!@#")
	_, err = patches.GetPatchOperation(app, otherUser.Id, created.ID)
	if err == nil {
		t.Fatal("expected error when accessing operation from different user")
	}
}

// TestUpdatePatchStatus tests updating patch operation status
func TestUpdatePatchStatus(t *testing.T) {
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
	app.OnRecordCreate("users").BindFunc(hooks.OnUserCreate(app))
	app.OnRecordUpdate("users").BindFunc(hooks.OnUserUpdate(app))

	// Create test user and agent
	user := createTestUser(app, t, "patch.update@example.com", "Password123!@#")
	userID := user.Id
	agent := createTestAgent(app, t, userID, "test-hostname")
	agentID := agent.Id

	setupPatchPrerequisites(app, t, agentID)

	// Create patch operation
	created, err := patches.CreatePatchOperation(app, userID, types.PatchRequest{
		AgentID: agentID,
		Mode:    "dry-run",
	})
	if err != nil {
		t.Fatalf("Failed to create patch operation: %v", err)
	}

	// Update status to running
	err = patches.UpdatePatchStatus(app, userID, created.ID, types.PatchStatusRunning, "", "")
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Verify update
	updated, err := patches.GetPatchOperation(app, userID, created.ID)
	if err != nil {
		t.Fatalf("Failed to get updated operation: %v", err)
	}

	if updated.Status != types.PatchStatusRunning {
		t.Errorf("Status not updated: expected running, got %s", updated.Status)
	}

	// Update to completed with output
	err = patches.UpdatePatchStatus(app, userID, created.ID, types.PatchStatusCompleted, "/output/path/result.txt", "")
	if err != nil {
		t.Fatalf("Failed to update to completed: %v", err)
	}

	// Verify completion
	completed, err := patches.GetPatchOperation(app, userID, created.ID)
	if err != nil {
		t.Fatalf("Failed to get completed operation: %v", err)
	}

	if completed.Status != types.PatchStatusCompleted {
		t.Errorf("Status not completed: got %s", completed.Status)
	}
}
