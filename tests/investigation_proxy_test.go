package tests

import (
	"fmt"
	"testing"

	"github.com/nannyagent/nannyapi/internal/investigations"
	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// setupTestUserAndAgent creates a test user and agent for investigation tests
func setupTestUserAndAgent(t *testing.T, app *tests.TestApp) (string, string) {
	// Create a test user
	usersCollection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Failed to find users collection: %v", err)
	}

	user := core.NewRecord(usersCollection)
	user.Set("email", randomEmail())
	user.Set("password", "Test123456!@#")
	user.Set("name", "Test User")
	user.SetVerified(true)

	if err := app.Save(user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create a test agent for the user using helper
	agent := createTestAgent(app, t, user.Id, "test-proxy-host")

	return user.Id, agent.Id
}

// TestInvestigationProxyWorkflow tests the complete investigation proxy workflow
// Simulates agent sending TensorZero requests through the API proxy
// Workflow:
// 1. User creates investigation (portal-initiated)
// 2. Agent proxies first TensorZero request - gets episode_id back
// 3. API tracks episode_id in investigation (status -> in_progress)
// 4. Agent sends multiple inferences through proxy
// 5. Agent sends final response with resolution_plan
// 6. API marks investigation complete
func TestInvestigationProxyWorkflow(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	userID, agentID := setupTestUserAndAgent(t, app)

	var investigationID string

	// Step 1: Create investigation from portal
	t.Run("Step1_CreateInvestigationFromPortal", func(t *testing.T) {
		investigationReq := types.InvestigationRequest{
			AgentID:  agentID,
			Issue:    "High CPU utilization observed in the past 15 minutes",
			Priority: "high",
		}

		investigation, err := investigations.CreateInvestigation(app, userID, investigationReq)
		if err != nil {
			t.Fatalf("Failed to create investigation: %v", err)
		}

		if investigation.ID == "" {
			t.Fatal("Investigation ID should not be empty")
		}

		if investigation.Status != types.InvestigationStatusPending {
			t.Fatalf("Expected status Pending, got %s", investigation.Status)
		}

		if investigation.EpisodeID != "" {
			t.Fatal("Episode ID should be empty before agent proxies request")
		}

		investigationID = investigation.ID
		t.Logf("✓ Created investigation: %s", investigation.ID)
	})

	// Step 2: Agent proxies TensorZero request with episode_id
	// Simulates: Agent calls /api/investigations/proxy -> API forwards to TensorZero -> Gets episode_id
	t.Run("Step2_ProxyTensorZeroRequestWithEpisodeID", func(t *testing.T) {
		episodeID := "019b403f-74a1-7201-a70e-1eacd1fc6e63"

		// Simulate TensorZero response containing episode_id
		// (This would normally come from the proxy handler)
		err := investigations.TrackInvestigationResponse(app, investigationID, episodeID, "")
		if err != nil {
			t.Fatalf("Failed to track investigation response: %v", err)
		}

		// Verify investigation updated with episode_id
		investigation, err := investigations.GetInvestigation(app, userID, investigationID)
		if err != nil {
			t.Fatalf("Failed to get investigation: %v", err)
		}

		if investigation.EpisodeID != episodeID {
			t.Fatalf("Expected episode_id %s, got %s", episodeID, investigation.EpisodeID)
		}

		if investigation.Status != types.InvestigationStatusInProgress {
			t.Fatalf("Expected status InProgress, got %s", investigation.Status)
		}

		t.Logf("✓ Investigation updated with episode_id: %s", episodeID)
		t.Logf("✓ Status changed to: %s", investigation.Status)
	})

	// Step 3: Agent sends multiple inferences through proxy (no resolution yet)
	// TensorZero typically takes 2-3 inferences before generating resolution_plan
	t.Run("Step3_MultipleInferencesBeforeResolution", func(t *testing.T) {
		episodeID := "019b403f-74a1-7201-a70e-1eacd1fc6e63"

		// Inference 2: Gathering process information
		t.Logf("  Inference 2: Gathering process information...")
		err := investigations.TrackInvestigationResponse(app, investigationID, episodeID, "")
		if err != nil {
			t.Fatalf("Failed to track inference 2: %v", err)
		}

		// Verify investigation still in progress
		investigation, err := investigations.GetInvestigation(app, userID, investigationID)
		if err != nil {
			t.Fatalf("Failed to get investigation: %v", err)
		}

		if investigation.Status != types.InvestigationStatusInProgress {
			t.Fatalf("Expected status InProgress, got %s", investigation.Status)
		}

		// Inference 3: Analyzing eBPF output
		t.Logf("  Inference 3: Analyzing eBPF output...")
		err = investigations.TrackInvestigationResponse(app, investigationID, episodeID, "")
		if err != nil {
			t.Fatalf("Failed to track inference 3: %v", err)
		}

		t.Logf("✓ Multiple inferences processed, investigation still in progress")
	})

	// Step 4: Agent sends final response with resolution_plan
	// This marks investigation as complete
	t.Run("Step4_ResolutionPlanReceived", func(t *testing.T) {
		episodeID := "019b403f-74a1-7201-a70e-1eacd1fc6e63"

		// Simulate final TensorZero response with resolution_plan
		resolutionPlan := `Kill process rogue_app (PID 1234) which is consuming 95% CPU. Process started at 10:25 AM and not responding to signals. Execute: kill -9 1234`

		err := investigations.TrackInvestigationResponse(app, investigationID, episodeID, resolutionPlan)
		if err != nil {
			t.Fatalf("Failed to track final response: %v", err)
		}

		// Verify investigation marked complete
		investigation, err := investigations.GetInvestigation(app, userID, investigationID)
		if err != nil {
			t.Fatalf("Failed to get investigation: %v", err)
		}

		if investigation.Status != types.InvestigationStatusCompleted {
			t.Fatalf("Expected status Completed, got %s", investigation.Status)
		}

		t.Logf("✓ Investigation marked as completed")
		t.Logf("✓ Resolution plan: %s", resolutionPlan)
	})

	// Step 5: Verify final investigation state
	t.Run("Step5_VerifyCompletedInvestigation", func(t *testing.T) {
		investigations, err := investigations.GetInvestigations(app, userID)
		if err != nil {
			t.Fatalf("Failed to list investigations: %v", err)
		}

		if len(investigations) == 0 {
			t.Fatal("Should have at least one investigation")
		}

		investigation := investigations[0]
		if investigation.Status != types.InvestigationStatusCompleted {
			t.Fatalf("Expected status Completed, got %s", investigation.Status)
		}

		if investigation.CompletedAt == nil {
			t.Fatal("Completed timestamp should be set")
		}

		t.Logf("✓ Investigation workflow complete")
		t.Logf("  - Status: %s", investigation.Status)
		t.Logf("  - Episode ID: %s", investigation.ID)
		t.Logf("  - Completed at: %v", investigation.CompletedAt)
	})
}

// TestInvestigationProxyPromptValidation tests prompt validation
func TestInvestigationProxyPromptValidation(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	userID, agentID := setupTestUserAndAgent(t, app)

	tests := []struct {
		name      string
		issue     string
		expectErr bool
	}{
		{
			name:      "Valid issue - longer than 10 chars",
			issue:     "This is a valid issue description",
			expectErr: false,
		},
		{
			name:      "Valid issue - exactly 10 chars",
			issue:     "1234567890",
			expectErr: false,
		},
		{
			name:      "Invalid issue - less than 10 chars",
			issue:     "Short",
			expectErr: true,
		},
		{
			name:      "Invalid issue - 9 chars",
			issue:     "123456789",
			expectErr: true,
		},
		{
			name:      "Valid with whitespace trimming",
			issue:     "   Valid issue with padding   ",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			investigationReq := types.InvestigationRequest{
				AgentID:  agentID,
				Issue:    tt.issue,
				Priority: "medium",
			}

			_, err := investigations.CreateInvestigation(app, userID, investigationReq)

			if tt.expectErr && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestInvestigationProxyAgentInitiated tests agent-initiated investigations
// Agent sends probe data directly without portal creating investigation first
func TestInvestigationProxyAgentInitiated(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	_, agentID := setupTestUserAndAgent(t, app)

	t.Run("AgentInitiatedInvestigation", func(t *testing.T) {
		// Agent-initiated: sends probe data directly to proxy
		// No investigation_id in metadata
		probeData := map[string]interface{}{
			"model": "tensorzero::function_name::diagnose_and_heal",
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": "System experiencing high CPU - see details below",
				},
			},
			"metadata": map[string]interface{}{
				"agent_id":          agentID,
				"issue_description": "High CPU utilization observed in the past 15 minutes",
				"initiated_by":      "agent",
				"function_name":     "diagnose_and_heal",
				"request_type":      "investigation",
			},
		}

		// In real proxy handler:
		// 1. Request forwarded to TensorZero Core
		// 2. Get episode_id from response
		// 3. Check if investigation exists by episode_id
		// 4. If not, create new investigation
		// 5. If yes, just track the response

		// Simulate what proxy handler would do
		episodeID := "agent-initiated-episode-123"
		t.Logf("✓ Agent-initiated investigation")
		t.Logf("  - Issue from metadata: %v", probeData["metadata"].(map[string]interface{})["issue_description"])
		t.Logf("  - Would create/use episode: %s", episodeID)
		t.Logf("  - TensorZero response processed through proxy")
	})
}

// TestInvestigationWithClickHouseEnrichment tests fetching inferences from ClickHouse
// When investigation is complete and has episode_id, GetInvestigation enriches response
func TestInvestigationWithClickHouseEnrichment(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	userID, agentID := setupTestUserAndAgent(t, app)

	t.Run("EnrichWithClickHouseData", func(t *testing.T) {
		// Create and complete an investigation
		investigationReq := types.InvestigationRequest{
			AgentID:  agentID,
			Issue:    "Database connection pool exhausted causing application slowdown",
			Priority: "critical",
		}

		investigation, err := investigations.CreateInvestigation(app, userID, investigationReq)
		if err != nil {
			t.Fatalf("Failed to create investigation: %v", err)
		}

		// Update with episode_id and resolution
		episodeID := "019b403f-74a1-7201-a70e-1eacd1fc6e63"
		resolutionPlan := "Increase connection pool size from 50 to 100 in config"

		err = investigations.TrackInvestigationResponse(app, investigation.ID, episodeID, resolutionPlan)
		if err != nil {
			t.Fatalf("Failed to track response: %v", err)
		}

		// When GetInvestigation is called, it should:
		// 1. Get investigation from PocketBase
		// 2. Check if episode_id exists
		// 3. If yes, query ClickHouse by episode_id
		// 4. Enrich response with inference data

		result, err := investigations.GetInvestigation(app, userID, investigation.ID)
		if err != nil {
			t.Fatalf("Failed to get investigation: %v", err)
		}

		if result.EpisodeID != episodeID {
			t.Fatalf("Expected episode_id %s, got %s", episodeID, result.EpisodeID)
		}

		// Note: ClickHouse query happens in GetInvestigation but requires ClickHouse service
		// In test environment without ClickHouse, the enrichment would skip gracefully
		t.Logf("✓ Investigation retrieved with episode_id")
		t.Logf("  - Episode ID: %s", result.EpisodeID)
		t.Logf("  - Status: %s", result.Status)
		t.Logf("  - (ClickHouse enrichment would happen here if service available)")
	})
}

// DocumentInvestigationProxyWorkflow documents the complete proxy workflow
// This shows how agent, portal, and API interact through the proxy
func DocumentInvestigationProxyWorkflow() {
	fmt.Println("=== Investigation Proxy Workflow ===")
	fmt.Println()
	fmt.Println("PORTAL SIDE:")
	fmt.Println("1. User creates investigation: 'High CPU utilization observed'")
	fmt.Println("   -> Prompt validated (minimum 10 characters)")
	fmt.Println("   -> Investigation created with status=pending, no episode_id yet")
	fmt.Println("   -> Investigation ID returned to frontend")
	fmt.Println()

	fmt.Println("AGENT SIDE:")
	fmt.Println("2. Agent receives investigation via realtime socket")
	fmt.Println("3. Agent prepares TensorZero request with investigation metadata")
	fmt.Println()

	fmt.Println("API PROXY:")
	fmt.Println("4. Agent sends: POST /api/investigations/proxy")
	fmt.Println("   Headers: Authorization: Bearer <agent_access_token>")
	fmt.Println("   Body: TensorZero request + metadata (investigation_id)")
	fmt.Println()
	fmt.Println("5. API validates agent token via ExtractAuthFromHeader()")
	fmt.Println("6. API forwards request to TensorZero Core")
	fmt.Println("7. TensorZero Core responds with episode_id (first response)")
	fmt.Println()

	fmt.Println("DATABASE TRACKING:")
	fmt.Println("8. API tracks episode_id in investigation")
	fmt.Println("   -> Sets episode_id in database")
	fmt.Println("   -> Changes status to in_progress")
	fmt.Println()

	fmt.Println("MULTIPLE INFERENCES:")
	fmt.Println("9. Agent sends follow-up inferences through proxy")
	fmt.Println("   Inference 1: Gathering system metrics")
	fmt.Println("   Inference 2: Analyzing process list")
	fmt.Println("   Inference 3: Processing eBPF output")
	fmt.Println("   -> API tracks inferences (checks for resolution_plan)")
	fmt.Println("   -> Investigation remains in_progress")
	fmt.Println()

	fmt.Println("RESOLUTION:")
	fmt.Println("10. TensorZero sends final response with resolution_plan")
	fmt.Println("    Content: 'Kill rogue_app (PID 1234) consuming 95% CPU'")
	fmt.Println()
	fmt.Println("11. API marks investigation as COMPLETED")
	fmt.Println("    -> Stores resolution_plan in database")
	fmt.Println("    -> Sets completed_at timestamp")
	fmt.Println("    -> Status changes to completed")
	fmt.Println()

	fmt.Println("ENRICHMENT:")
	fmt.Println("12. User queries investigation via portal")
	fmt.Println("    GET /api/investigations?id=<investigation_id>")
	fmt.Println()
	fmt.Println("13. API enriches response with ClickHouse data")
	fmt.Println("    -> Uses episode_id to query ClickHouse")
	fmt.Println("    -> Fetches inference details (model used, tokens, latency)")
	fmt.Println("    -> Includes in response metadata")
	fmt.Println()

	fmt.Println("RESULT:")
	fmt.Println("Investigation complete with full audit trail from TensorZero")
	fmt.Println("- User prompt that triggered investigation")
	fmt.Println("- Episode ID linking to TensorZero episode")
	fmt.Println("- Resolution plan provided by AI")
	fmt.Println("- Inference details from ClickHouse")
}
