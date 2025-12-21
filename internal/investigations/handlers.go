package investigations

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nannyagent/nannyapi/internal/clickhouse"
	"github.com/nannyagent/nannyapi/internal/tensorzero"
	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
)

// CreateInvestigation creates a new investigation record and calls TensorZero API
// Called when user initiates investigation via API
func CreateInvestigation(app core.App, userID string, req types.InvestigationRequest) (*types.InvestigationResponse, error) {
	// Basic validation
	if req.AgentID == "" || req.Issue == "" {
		return nil, fmt.Errorf("agent_id and issue are required")
	}

	// Validate prompt length (minimum 10 characters)
	trimmedIssue := strings.TrimSpace(req.Issue)
	if len(trimmedIssue) < 10 {
		return nil, fmt.Errorf("issue must be at least 10 characters long")
	}

	// Get investigations collection
	collection, err := app.FindCollectionByNameOrId("investigations")
	if err != nil {
		return nil, fmt.Errorf("investigations collection not found: %w", err)
	}

	// Get agents collection to verify agent exists and user owns it
	agentsCollection, err := app.FindCollectionByNameOrId("agents")
	if err != nil {
		return nil, fmt.Errorf("agents collection not found: %w", err)
	}

	// Verify agent exists and belongs to user
	agentRecord, err := app.FindRecordById(agentsCollection.Id, req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	agentUserID := agentRecord.GetString("user_id")
	if agentUserID != userID {
		return nil, fmt.Errorf("unauthorized: agent does not belong to user")
	}

	// Set default priority
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	// Create new investigation record
	record := core.NewRecord(collection)
	record.Set("user_id", userID)
	record.Set("agent_id", req.AgentID)
	record.Set("user_prompt", trimmedIssue)
	record.Set("priority", priority)
	record.Set("status", string(types.InvestigationStatusPending))
	record.Set("initiated_at", time.Now())
	record.Set("metadata", map[string]interface{}{})

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save investigation: %w", err)
	}

	// Call TensorZero API for AI analysis (asynchronously in production)
	tzClient := tensorzero.NewClient()
	messages := []types.ChatMessage{
		{
			Role:    "user",
			Content: fmt.Sprintf("System Issue: %s\n\nAgent ID: %s\n\nPlease analyze this issue and provide diagnostic insights and resolution steps.", trimmedIssue, req.AgentID),
		},
	}

	tzResp, err := tzClient.CallChatCompletion(messages)
	if err != nil {
		// Log error but don't fail investigation creation - it can be retried later
		fmt.Printf("Warning: TensorZero API call failed: %v\n", err)
	} else if tzResp != nil && tzResp.EpisodeID != "" {
		// Update investigation with episode_id from TensorZero
		record.Set("episode_id", tzResp.EpisodeID)
		if len(tzResp.Choices) > 0 {
			record.Set("resolution_plan", tzResp.Choices[0].Message.Content)
			record.Set("status", string(types.InvestigationStatusInProgress))
		}
		if err := app.Save(record); err != nil {
			fmt.Printf("Warning: Failed to update investigation with TensorZero response: %v\n", err)
		}
	}

	// Return response
	return &types.InvestigationResponse{
		ID:         record.Id,
		UserID:     userID,
		AgentID:    req.AgentID,
		EpisodeID:  record.GetString("episode_id"),
		UserPrompt: trimmedIssue,
		Priority:   priority,
		Status:     types.InvestigationStatus(record.GetString("status")),
		CreatedAt:  record.GetDateTime("created").Time(),
		UpdatedAt:  record.GetDateTime("updated").Time(),
		Metadata:   make(map[string]interface{}),
	}, nil
}

// GetInvestigations retrieves all investigations for a user
func GetInvestigations(app core.App, userID string) ([]*types.InvestigationListResponse, error) {
	collection, err := app.FindCollectionByNameOrId("investigations")
	if err != nil {
		return nil, fmt.Errorf("investigations collection not found: %w", err)
	}

	// List investigations for this user
	records, err := app.FindRecordsByFilter(collection, "user_id = {:uid}", "", 0, 0, map[string]interface{}{"uid": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch investigations: %w", err)
	}

	var investigations []*types.InvestigationListResponse
	for _, record := range records {
		completedAt := getTimePtr(record, "completed_at")
		investigations = append(investigations, &types.InvestigationListResponse{
			ID:          record.Id,
			AgentID:     record.GetString("agent_id"),
			UserPrompt:  record.GetString("user_prompt"),
			Priority:    record.GetString("priority"),
			Status:      types.InvestigationStatus(record.GetString("status")),
			InitiatedAt: record.GetDateTime("initiated_at").Time(),
			CompletedAt: completedAt,
			CreatedAt:   record.GetDateTime("created").Time(),
		})
	}

	return investigations, nil
}

// GetInvestigation retrieves a single investigation by ID and enriches with ClickHouse inferences
func GetInvestigation(app core.App, userID, investigationID string) (*types.InvestigationResponse, error) {
	collection, err := app.FindCollectionByNameOrId("investigations")
	if err != nil {
		return nil, fmt.Errorf("investigations collection not found: %w", err)
	}

	record, err := app.FindRecordById(collection.Id, investigationID)
	if err != nil {
		return nil, fmt.Errorf("investigation not found: %w", err)
	}

	// Verify ownership
	if record.GetString("user_id") != userID {
		return nil, fmt.Errorf("unauthorized: investigation does not belong to user")
	}

	response := &types.InvestigationResponse{
		ID:         record.Id,
		UserID:     userID,
		AgentID:    record.GetString("agent_id"),
		EpisodeID:  record.GetString("episode_id"),
		UserPrompt: record.GetString("user_prompt"),
		Priority:   record.GetString("priority"),
		Status:     types.InvestigationStatus(record.GetString("status")),
		CreatedAt:  record.GetDateTime("created").Time(),
		UpdatedAt:  record.GetDateTime("updated").Time(),
		Metadata:   getMetadata(record),
	}

	// If episode_id exists, query ClickHouse for inference data
	if response.EpisodeID != "" {
		chClient := clickhouse.NewClient()
		inferences, err := chClient.FetchInferencesByEpisode(response.EpisodeID)
		if err != nil {
			// Log error but don't fail - investigation is still valid
			fmt.Printf("Warning: Failed to fetch inferences from ClickHouse: %v\n", err)
		} else {
			// Add inference count to response metadata if not already set
			if response.Metadata == nil {
				response.Metadata = make(map[string]interface{})
			}
			response.InferenceCount = len(inferences)
			response.Metadata["inferences"] = inferences
		}
	}

	return response, nil
}

// UpdateInvestigationStatus updates investigation status and resolution
func UpdateInvestigationStatus(
	app core.App,
	userID, investigationID string,
	status types.InvestigationStatus,
	resolutionPlan string,
) error {
	collection, err := app.FindCollectionByNameOrId("investigations")
	if err != nil {
		return fmt.Errorf("investigations collection not found: %w", err)
	}

	record, err := app.FindRecordById(collection.Id, investigationID)
	if err != nil {
		return fmt.Errorf("investigation not found: %w", err)
	}

	// Verify ownership
	if record.GetString("user_id") != userID {
		return fmt.Errorf("unauthorized: investigation does not belong to user")
	}

	record.Set("status", string(status))
	if resolutionPlan != "" {
		record.Set("resolution_plan", resolutionPlan)
	}
	if status == types.InvestigationStatusCompleted || status == types.InvestigationStatusFailed {
		record.Set("completed_at", time.Now())
	}

	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to update investigation: %w", err)
	}

	return nil
}

// SetEpisodeID updates investigation with TensorZero episode ID
func SetEpisodeID(app core.App, userID, investigationID, episodeID string) error {
	collection, err := app.FindCollectionByNameOrId("investigations")
	if err != nil {
		return fmt.Errorf("investigations collection not found: %w", err)
	}

	record, err := app.FindRecordById(collection.Id, investigationID)
	if err != nil {
		return fmt.Errorf("investigation not found: %w", err)
	}

	// Verify ownership
	if record.GetString("user_id") != userID {
		return fmt.Errorf("unauthorized: investigation does not belong to user")
	}

	record.Set("episode_id", episodeID)
	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to update episode_id: %w", err)
	}

	return nil
}

// HandleInvestigations handles investigation API endpoints
func HandleInvestigations(app core.App, c *core.RequestEvent) error {
	// Get authenticated user
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
	}
	user := authRecord.(*core.Record)

	// Determine action based on method
	switch c.Request.Method {
	case http.MethodPost:
		return handleCreateInvestigation(app, c, user.Id)
	case http.MethodGet:
		// Check if getting specific investigation or list
		pathID := c.Request.URL.Query().Get("id")
		if pathID != "" {
			return handleGetInvestigation(app, c, user.Id, pathID)
		}
		return handleListInvestigations(app, c, user.Id)
	case http.MethodPatch:
		pathID := c.Request.URL.Query().Get("id")
		if pathID == "" {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "id parameter required"})
		}
		return handleUpdateInvestigation(app, c, user.Id, pathID)
	}

	return c.JSON(http.StatusMethodNotAllowed, types.ErrorResponse{Error: "method not allowed"})
}

func handleCreateInvestigation(app core.App, c *core.RequestEvent, userID string) error {
	var req types.InvestigationRequest
	if err := c.BindBody(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request"})
	}

	resp, err := CreateInvestigation(app, userID, req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, resp)
}

func handleListInvestigations(app core.App, c *core.RequestEvent, userID string) error {
	investigations, err := GetInvestigations(app, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, investigations)
}

func handleGetInvestigation(app core.App, c *core.RequestEvent, userID, investigationID string) error {
	investigation, err := GetInvestigation(app, userID, investigationID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, investigation)
}

func handleUpdateInvestigation(app core.App, c *core.RequestEvent, userID, investigationID string) error {
	var updateReq struct {
		Status         string `json:"status"`
		ResolutionPlan string `json:"resolution_plan"`
		EpisodeID      string `json:"episode_id"`
	}
	if err := c.BindBody(&updateReq); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request"})
	}

	if updateReq.EpisodeID != "" {
		if err := SetEpisodeID(app, userID, investigationID, updateReq.EpisodeID); err != nil {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
		}
	}

	if updateReq.Status != "" {
		if err := UpdateInvestigationStatus(app, userID, investigationID, types.InvestigationStatus(updateReq.Status), updateReq.ResolutionPlan); err != nil {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"success": "true"})
}

// Helper functions

func getTimePtr(record *core.Record, fieldName string) *time.Time {
	val := record.GetDateTime(fieldName)
	t := val.Time()
	if t.IsZero() {
		return nil
	}
	return &t
}

func getMetadata(record *core.Record) map[string]interface{} {
	raw := record.Get("metadata")
	if raw == nil {
		return make(map[string]interface{})
	}
	if m, ok := raw.(map[string]interface{}); ok {
		return m
	}
	return make(map[string]interface{})
}
