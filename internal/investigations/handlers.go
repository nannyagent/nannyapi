package investigations

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
)

// CreateInvestigation creates a new investigation record
// Called when user initiates investigation via API
func CreateInvestigation(app core.App, userID string, req types.InvestigationRequest) (*types.InvestigationResponse, error) {
	// Basic validation
	if req.AgentID == "" || req.Issue == "" {
		return nil, fmt.Errorf("agent_id and issue are required")
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
	record.Set("user_prompt", req.Issue)
	record.Set("priority", priority)
	record.Set("status", string(types.InvestigationStatusPending))
	record.Set("initiated_at", time.Now())
	record.Set("metadata", map[string]interface{}{})

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save investigation: %w", err)
	}

	// Return response
	return &types.InvestigationResponse{
		ID:         record.Id,
		UserID:     userID,
		AgentID:    req.AgentID,
		EpisodeID:  "",
		UserPrompt: req.Issue,
		Priority:   priority,
		Status:     types.InvestigationStatusPending,
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

// GetInvestigation retrieves a single investigation by ID
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

	return &types.InvestigationResponse{
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
	}, nil
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
	if c.Request.Method == http.MethodPost {
		return handleCreateInvestigation(app, c, user.Id)
	} else if c.Request.Method == http.MethodGet {
		// Check if getting specific investigation or list
		pathID := c.Request.URL.Query().Get("id")
		if pathID != "" {
			return handleGetInvestigation(app, c, user.Id, pathID)
		}
		return handleListInvestigations(app, c, user.Id)
	} else if c.Request.Method == http.MethodPatch {
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
