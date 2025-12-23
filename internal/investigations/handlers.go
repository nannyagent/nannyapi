package investigations

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nannyagent/nannyapi/internal/clickhouse"
	"github.com/nannyagent/nannyapi/internal/tensorzero"
	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
)

// CreateInvestigation creates investigation record (portal-initiated only)
// Called by: User via `/api/investigations` POST
// Does: Validate prompt (10+ chars), create DB record, return investigation_id
// Then: Agent receives via realtime, sends back to same endpoint as proxy
func CreateInvestigation(app core.App, userID string, req types.InvestigationRequest, initiatedBy string) (*types.InvestigationResponse, error) {
	// Validation
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

	// Verify agent exists and belongs to user
	agentsCollection, err := app.FindCollectionByNameOrId("agents")
	if err != nil {
		return nil, fmt.Errorf("agents collection not found: %w", err)
	}

	agentRecord, err := app.FindRecordById(agentsCollection.Id, req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	if agentRecord.GetString("user_id") != userID {
		return nil, fmt.Errorf("unauthorized: agent does not belong to user")
	}

	// Set default priority
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	// Create investigation record (status: pending, no episode_id)
	record := core.NewRecord(collection)
	record.Set("user_id", userID)
	record.Set("agent_id", req.AgentID)
	record.Set("user_prompt", trimmedIssue)
	record.Set("priority", priority)
	record.Set("status", string(types.InvestigationStatusPending))
	record.Set("initiated_at", time.Now())
	record.Set("metadata", map[string]interface{}{"initiated_by": initiatedBy})

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save investigation: %w", err)
	}

	return &types.InvestigationResponse{
		ID:             record.Id,
		UserID:         userID,
		AgentID:        req.AgentID,
		EpisodeID:      "",
		UserPrompt:     trimmedIssue,
		Priority:       priority,
		Status:         types.InvestigationStatusPending,
		ResolutionPlan: "",
		InitiatedAt:    record.GetDateTime("initiated_at").Time(),
		CreatedAt:      record.GetDateTime("created").Time(),
		UpdatedAt:      record.GetDateTime("updated").Time(),
		Metadata:       map[string]interface{}{"initiated_by": initiatedBy},
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
		ID:             record.Id,
		UserID:         userID,
		AgentID:        record.GetString("agent_id"),
		EpisodeID:      record.GetString("episode_id"),
		UserPrompt:     record.GetString("user_prompt"),
		Priority:       record.GetString("priority"),
		Status:         types.InvestigationStatus(record.GetString("status")),
		ResolutionPlan: record.GetString("resolution_plan"),
		InitiatedAt:    record.GetDateTime("initiated_at").Time(),
		CreatedAt:      record.GetDateTime("created").Time(),
		UpdatedAt:      record.GetDateTime("updated").Time(),
		Metadata:       getMetadata(record),
	}

	// Add CompletedAt if it exists
	completedAt := record.GetDateTime("completed_at").Time()
	if !completedAt.IsZero() {
		response.CompletedAt = &completedAt
	}

	// If episode_id exists, query ClickHouse for inference data (ESSENTIAL for production)
	if response.EpisodeID != "" {
		chClient := clickhouse.NewClient()
		// ClickHouse is configured and required
		inferences, err := chClient.FetchInferencesByEpisode(response.EpisodeID)
		if err != nil {
			// ClickHouse configured but failed - log and continue
			app.Logger().Error("failed to fetch inferences from ClickHouse", "error", err)
		} else {
			// Add inference count to response metadata
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

// TrackInvestigationResponse tracks TensorZero responses
// Called when agent proxies TensorZero responses back to API
// Updates episode_id on first response, marks complete when resolution_plan arrives
func TrackInvestigationResponse(
	app core.App,
	investigationID string,
	episodeID string,
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

	// Update episode_id if provided and not already set
	if episodeID != "" && record.GetString("episode_id") == "" {
		record.Set("episode_id", episodeID)
		record.Set("status", string(types.InvestigationStatusInProgress))
	}

	// Mark investigation complete if resolution_plan provided
	if resolutionPlan != "" {
		record.Set("resolution_plan", resolutionPlan)
		record.Set("status", string(types.InvestigationStatusCompleted))
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
	// Get authenticated user or agent
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
	}
	record := authRecord.(*core.Record)

	var userID string
	if record.Collection().Name == "users" {
		userID = record.Id
	} else if record.Collection().Name == "agents" {
		userID = record.GetString("user_id")
		if userID == "" {
			return c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "agent has no owner"})
		}
	} else {
		return c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "invalid authentication type"})
	}

	// Determine action based on method
	switch c.Request.Method {
	case http.MethodPost:
		return handleCreateInvestigation(app, c, userID)
	case http.MethodGet:
		// Check if getting specific investigation or list
		pathID := c.Request.URL.Query().Get("id")
		if pathID != "" {
			return handleGetInvestigation(app, c, userID, pathID)
		}
		return handleListInvestigations(app, c, userID)
	case http.MethodPatch:
		pathID := c.Request.URL.Query().Get("id")
		if pathID == "" {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "id parameter required"})
		}
		return handleUpdateInvestigation(app, c, userID, pathID)
	}

	return c.JSON(http.StatusMethodNotAllowed, types.ErrorResponse{Error: "method not allowed"})
}

func handleCreateInvestigation(app core.App, c *core.RequestEvent, userID string) error {
	// Read request body to determine if this is a proxy or portal-initiated request
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "failed to read request"})
	}
	defer c.Request.Body.Close()

	// Parse raw JSON to check for investigation_id
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid JSON"})
	}

	investigationID, hasInvestigationID := bodyMap["investigation_id"].(string)

	// Determine initiated_by
	initiatedBy := "user"
	authRecord := c.Get("authRecord")
	if authRecord != nil {
		rec := authRecord.(*core.Record)
		if rec.Collection().Name == "agents" {
			initiatedBy = "agent"
		}
	}

	// CASE 1: Portal-initiated investigation (no investigation_id in body)
	if !hasInvestigationID {
		var req types.InvestigationRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request format"})
		}

		resp, err := CreateInvestigation(app, userID, req, initiatedBy)
		if err != nil {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
		}

		return c.JSON(http.StatusCreated, resp)
	}

	// CASE 2: Agent proxy request (has investigation_id in body)
	// Forward request UNCHANGED to TensorZero Core, parse response for episode_id/resolution_plan
	return proxyToTensorZero(app, c, userID, investigationID, bodyBytes)
}

// proxyToTensorZero forwards investigation request to TensorZero Core unchanged
func proxyToTensorZero(app core.App, c *core.RequestEvent, userID, investigationID string, bodyBytes []byte) error {
	// Verify investigation exists and belongs to user
	collection, err := app.FindCollectionByNameOrId("investigations")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "investigations collection not found"})
	}

	record, err := app.FindRecordById(collection.Id, investigationID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "investigation not found"})
	}

	// Verify user ownership
	if record.GetString("user_id") != userID {
		return c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "unauthorized"})
	}

	// Parse request as TensorZero core request
	var tzRequest types.TensorZeroCoreRequest
	if err := json.Unmarshal(bodyBytes, &tzRequest); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid TensorZero request format"})
	}

	// Forward to TensorZero Core UNCHANGED
	// tzClient will panic if credentials are not set, which is correct behavior
	tzClient := tensorzero.NewClient()

	// Determine model based on initiated_by
	metadata := getMetadata(record)
	initiatedBy := "user"
	if val, ok := metadata["initiated_by"].(string); ok {
		initiatedBy = val
	}

	model := types.TensorZeroModelDiagnoseAndHealApplication
	if initiatedBy == "agent" {
		model = types.TensorZeroModelDiagnoseAndHeal
	}

	// Get episode_id from the investigations table & set it for  next inferences
	investigation, err := GetInvestigation(app, userID, investigationID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.ErrorResponse{Error: err.Error()})
	}

	// if there is no empty episode_id, tensorzero will anyways discard it
	episodeIDForInference := investigation.EpisodeID

	tzResp, err := tzClient.CallChatCompletion(tzRequest.Messages, model, episodeIDForInference)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: fmt.Sprintf("TensorZero error: %v", err)})
	}

	// Parse TensorZero response for episode_id (first response) and resolution_plan (final response)
	episodeID := tzResp.EpisodeID
	var resolutionPlan string

	// Parse response content to extract resolution_plan
	if len(tzResp.Choices) > 0 {
		// Content is a JSON string that needs to be parsed
		content := tzResp.Choices[0].Message.Content

		// Try to parse as ResolutionResponse (final response with resolution_plan)
		var resolutionResp types.ResolutionResponse
		if err := json.Unmarshal([]byte(content), &resolutionResp); err == nil {
			// Successfully parsed - this is a resolution response
			if resolutionResp.ResponseType == "resolution" && resolutionResp.ResolutionPlan != "" {
				resolutionPlan = resolutionResp.ResolutionPlan
			}
		} else {
			// Not a resolution response, try to parse as DiagnosticResponse
			var diagnosticResp types.DiagnosticResponse
			if err := json.Unmarshal([]byte(content), &diagnosticResp); err == nil {
				// Diagnostic response - no resolution plan yet
				// This is normal for intermediate responses
			}
		}
	}

	// Track investigation response in database
	if err := TrackInvestigationResponse(app, investigationID, episodeID, resolutionPlan); err != nil {
		// Log error but continue - response still needs to be returned to agent
		fmt.Printf("Warning: Failed to track investigation response: %v\n", err)
	}

	// Return TensorZero response as-is to agent
	return c.JSON(http.StatusOK, tzResp)
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
