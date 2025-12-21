package tests

import (
	"testing"

	"github.com/nannyagent/nannyapi/internal/hooks"
	"github.com/nannyagent/nannyapi/internal/patches"
	"github.com/nannyagent/nannyapi/internal/types"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/tests"
)

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
				AgentID:   agentID,
				Mode:      "dry-run",
				ScriptURL: "https://example.com/scripts/update.sh",
			},
			expectErr: false,
		},
		{
			name:   "Create patch with apply mode",
			userID: userID,
			request: types.PatchRequest{
				AgentID:   agentID,
				Mode:      "apply",
				ScriptURL: "https://example.com/scripts/apply.sh",
			},
			expectErr: false,
		},
		{
			name:   "Fail with invalid mode",
			userID: userID,
			request: types.PatchRequest{
				AgentID:   agentID,
				Mode:      "invalid",
				ScriptURL: "https://example.com/scripts/test.sh",
			},
			expectErr: true,
		},
		{
			name:   "Fail with missing agent_id",
			userID: userID,
			request: types.PatchRequest{
				Mode:      "dry-run",
				ScriptURL: "https://example.com/scripts/test.sh",
			},
			expectErr: true,
		},
		{
			name:   "Fail with missing script_url",
			userID: userID,
			request: types.PatchRequest{
				AgentID: agentID,
				Mode:    "apply",
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
		})
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

	// Create multiple patch operations
	for i := 0; i < 2; i++ {
		_, err := patches.CreatePatchOperation(app, userID, types.PatchRequest{
			AgentID:   agentID,
			Mode:      "dry-run",
			ScriptURL: "https://example.com/scripts/update.sh",
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

	// Create patch operation
	created, err := patches.CreatePatchOperation(app, userID, types.PatchRequest{
		AgentID:   agentID,
		Mode:      "apply",
		ScriptURL: "https://example.com/scripts/update.sh",
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

	// Create patch operation
	created, err := patches.CreatePatchOperation(app, userID, types.PatchRequest{
		AgentID:   agentID,
		Mode:      "dry-run",
		ScriptURL: "https://example.com/scripts/update.sh",
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

// TestCreatePackageUpdate tests creating package update records
func TestCreatePackageUpdate(t *testing.T) {
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
	user := createTestUser(app, t, "patch.package@example.com", "Password123!@#")
	userID := user.Id
	agent := createTestAgent(app, t, userID, "test-hostname")
	agentID := agent.Id

	// Create patch operation
	created, err := patches.CreatePatchOperation(app, userID, types.PatchRequest{
		AgentID:   agentID,
		Mode:      "apply",
		ScriptURL: "https://example.com/scripts/update.sh",
	})
	if err != nil {
		t.Fatalf("Failed to create patch operation: %v", err)
	}

	// Create package update
	pkg := types.PatchPackageInfo{
		Name:       "openssl",
		Version:    "3.0.0",
		UpdateType: "security",
	}

	err = patches.CreatePackageUpdate(app, created.ID, pkg)
	if err != nil {
		t.Fatalf("Failed to create package update: %v", err)
	}
}
