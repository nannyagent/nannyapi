package tests

import (
	"os"
	"testing"

	"github.com/nannyagent/nannyapi/internal/hooks"
	"github.com/nannyagent/nannyapi/internal/investigations"
	"github.com/nannyagent/nannyapi/internal/types"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/tests"
)

// setupInvestigationTestApp creates test app with migrations and registered hooks
func setupInvestigationTestApp(t *testing.T) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}

	// Run migrations to create collections
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register hooks
	app.OnRecordCreate("users").BindFunc(hooks.OnUserCreate(app))
	app.OnRecordUpdate("users").BindFunc(hooks.OnUserUpdate(app))

	return app
}

// TestCreateInvestigation tests investigation creation with various scenarios
func TestCreateInvestigation(t *testing.T) {
	app := setupInvestigationTestApp(t)
	defer app.Cleanup()

	// Create a test user first
	user := createTestUser(app, t, "create.investigation@example.com", "Password123!@#")
	userID := user.Id

	// Create a test agent
	agent := createTestAgent(app, t, userID, "test-hostname")
	agentID := agent.Id

	// Test cases
	tests := []struct {
		name      string
		userID    string
		request   types.InvestigationRequest
		expectErr bool
		errMsg    string
	}{
		{
			name:   "Create investigation successfully",
			userID: userID,
			request: types.InvestigationRequest{
				AgentID:  agentID,
				Issue:    "System CPU usage is consistently above 90%",
				Priority: "high",
			},
			expectErr: false,
		},
		{
			name:   "Create investigation with default priority",
			userID: userID,
			request: types.InvestigationRequest{
				AgentID: agentID,
				Issue:   "Memory leak detected in application",
			},
			expectErr: false,
		},
		{
			name:   "Fail with missing agent_id",
			userID: userID,
			request: types.InvestigationRequest{
				Issue:    "Test issue",
				Priority: "high",
			},
			expectErr: true,
		},
		{
			name:   "Fail with missing issue",
			userID: userID,
			request: types.InvestigationRequest{
				AgentID:  agentID,
				Priority: "high",
			},
			expectErr: true,
		},
		{
			name:      "Fail with invalid user ID",
			userID:    "invalid-user-id",
			request:   types.InvestigationRequest{AgentID: agentID, Issue: "Test"},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := investigations.CreateInvestigation(app, tc.userID, tc.request, "user")

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

			if resp.UserID != tc.userID {
				t.Errorf("UserID mismatch: expected %s, got %s", tc.userID, resp.UserID)
			}
			if resp.AgentID != tc.request.AgentID {
				t.Errorf("AgentID mismatch: expected %s, got %s", tc.request.AgentID, resp.AgentID)
			}
			if resp.UserPrompt != tc.request.Issue {
				t.Errorf("UserPrompt mismatch: expected %s, got %s", tc.request.Issue, resp.UserPrompt)
			}
			if resp.Status != types.InvestigationStatusPending {
				t.Errorf("Status should be pending, got %s", resp.Status)
			}

			// Check priority defaults to medium if not provided
			expectedPriority := tc.request.Priority
			if expectedPriority == "" {
				expectedPriority = "medium"
			}
			if resp.Priority != expectedPriority {
				t.Errorf("Priority mismatch: expected %s, got %s", expectedPriority, resp.Priority)
			}
		})
	}
}

// TestGetInvestigations tests listing investigations for a user
func TestGetInvestigations(t *testing.T) {
	app := setupInvestigationTestApp(t)
	defer app.Cleanup()

	// Create a test user first
	user := createTestUser(app, t, "get.investigations@example.com", "Password123!@#")
	userID := user.Id

	// Create a test agent
	agent := createTestAgent(app, t, userID, "test-hostname")
	agentID := agent.Id

	// Create multiple investigations
	for i := 0; i < 2; i++ {
		_, err := investigations.CreateInvestigation(app, userID, types.InvestigationRequest{
			AgentID:  agentID,
			Issue:    "Test issue " + string(rune(i)),
			Priority: "medium",
		}, "user")
		if err != nil {
			t.Fatalf("Failed to create investigation: %v", err)
		}
	}

	// Get all investigations
	list, err := investigations.GetInvestigations(app, userID)
	if err != nil {
		t.Fatalf("Failed to list investigations: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 investigations, got %d", len(list))
	}
}

// TestGetInvestigation tests retrieving a single investigation
func TestGetInvestigation(t *testing.T) {
	LoadEnv(t)
	app := setupInvestigationTestApp(t)
	defer app.Cleanup()

	// Create test user and agent
	user := createTestUser(app, t, "get.investigation@example.com", "Password123!@#")
	userID := user.Id
	agent := createTestAgent(app, t, userID, "test-hostname")
	agentID := agent.Id

	// Create investigation
	created, err := investigations.CreateInvestigation(app, userID, types.InvestigationRequest{
		AgentID:  agentID,
		Issue:    "Test issue for retrieval",
		Priority: "high",
	}, "user")
	if err != nil {
		t.Fatalf("Failed to create investigation: %v", err)
	}

	// Retrieve it
	retrieved, err := investigations.GetInvestigation(app, userID, created.ID)
	if err != nil {
		t.Fatalf("Failed to get investigation: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: expected %s, got %s", created.ID, retrieved.ID)
	}
	if retrieved.UserPrompt != "Test issue for retrieval" {
		t.Errorf("Issue mismatch: expected 'Test issue for retrieval', got %s", retrieved.UserPrompt)
	}

	// Test unauthorized access - different user trying to access
	otherUser := createTestUser(app, t, "other.user@example.com", "Password123!@#")
	_, err = investigations.GetInvestigation(app, otherUser.Id, created.ID)
	if err == nil {
		t.Fatal("expected error when accessing investigation from different user")
	}
}

// TestUpdateInvestigationStatus tests updating investigation status
func TestUpdateInvestigationStatus(t *testing.T) {
	LoadEnv(t)
	app := setupInvestigationTestApp(t)
	defer app.Cleanup()

	// Create test user and agent
	user := createTestUser(app, t, "update.status@example.com", "Password123!@#")
	userID := user.Id
	agent := createTestAgent(app, t, userID, "test-hostname")
	agentID := agent.Id

	// Create investigation
	created, err := investigations.CreateInvestigation(app, userID, types.InvestigationRequest{
		AgentID:  agentID,
		Issue:    "Test issue",
		Priority: "high",
	}, "user")
	if err != nil {
		t.Fatalf("Failed to create investigation: %v", err)
	}

	// Update status to in_progress
	err = investigations.UpdateInvestigationStatus(app, userID, created.ID, types.InvestigationStatusInProgress, "")
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Verify update
	updated, err := investigations.GetInvestigation(app, userID, created.ID)
	if err != nil {
		t.Fatalf("Failed to get updated investigation: %v", err)
	}

	if updated.Status != types.InvestigationStatusInProgress {
		t.Errorf("Status not updated: expected in_progress, got %s", updated.Status)
	}

	// Update to completed with plan
	resolutionPlan := "Restarted the affected service"
	err = investigations.UpdateInvestigationStatus(app, userID, created.ID, types.InvestigationStatusCompleted, resolutionPlan)
	if err != nil {
		t.Fatalf("Failed to update status to completed: %v", err)
	}

	// Verify completion
	completed, err := investigations.GetInvestigation(app, userID, created.ID)
	if err != nil {
		t.Fatalf("Failed to get completed investigation: %v", err)
	}

	if completed.Status != types.InvestigationStatusCompleted {
		t.Errorf("Status not set to completed: got %s", completed.Status)
	}
}

// TestSetEpisodeID tests setting episode ID for investigation
func TestSetEpisodeID(t *testing.T) {
	// Check if CLICKHOUSE_URL is already set in environment
	if os.Getenv("CLICKHOUSE_URL") == "" {
		// Check if .env file exists before calling LoadEnv
		if _, err := os.Stat(".env"); os.IsNotExist(err) {
			// No .env file, skip the test
			t.Skip("CLICKHOUSE_URL not set and no .env file")
		}
		// .env exists, try to load it
		LoadEnv(t)
		// Check again after loading .env
		if os.Getenv("CLICKHOUSE_URL") == "" {
			t.Skip("CLICKHOUSE_URL not set")
		}
	}
	app := setupInvestigationTestApp(t)
	defer app.Cleanup()

	// Create test user and agent
	user := createTestUser(app, t, "set.episode@example.com", "Password123!@#")
	userID := user.Id
	agent := createTestAgent(app, t, userID, "test-hostname")
	agentID := agent.Id

	// Create investigation
	created, err := investigations.CreateInvestigation(app, userID, types.InvestigationRequest{
		AgentID:  agentID,
		Issue:    "Test issue",
		Priority: "high",
	}, "user")
	if err != nil {
		t.Fatalf("Failed to create investigation: %v", err)
	}

	// Set episode ID
	episodeID := "ep_12345abcde"
	err = investigations.SetEpisodeID(app, userID, created.ID, episodeID)
	if err != nil {
		t.Fatalf("Failed to set episode ID: %v", err)
	}

	// Verify episode ID is set
	updated, err := investigations.GetInvestigation(app, userID, created.ID)
	if err != nil {
		t.Fatalf("Failed to get updated investigation: %v", err)
	}

	if updated.EpisodeID != episodeID {
		t.Errorf("EpisodeID not set correctly: expected %s, got %s", episodeID, updated.EpisodeID)
	}
}
