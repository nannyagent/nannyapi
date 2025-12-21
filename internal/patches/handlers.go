package patches

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
)

// CreatePatchOperation creates a new patch operation record
func CreatePatchOperation(app core.App, userID string, req types.PatchRequest) (*types.PatchResponse, error) {
	// Basic validation
	if req.AgentID == "" || req.ScriptURL == "" {
		return nil, fmt.Errorf("agent_id and script_url are required")
	}

	// Get patch_operations collection
	collection, err := app.FindCollectionByNameOrId("patch_operations")
	if err != nil {
		return nil, fmt.Errorf("patch_operations collection not found: %w", err)
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

	// Validate mode
	mode := types.PatchMode(req.Mode)
	if mode != types.PatchModeDryRun && mode != types.PatchModeApply {
		return nil, fmt.Errorf("invalid mode: must be dry-run or apply")
	}

	// Create new patch operation record
	record := core.NewRecord(collection)
	record.Set("user_id", userID)
	record.Set("agent_id", req.AgentID)
	record.Set("mode", string(mode))
	record.Set("status", string(types.PatchStatusPending))
	record.Set("script_url", req.ScriptURL)

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save patch operation: %w", err)
	}

	return &types.PatchResponse{
		ID:        record.Id,
		UserID:    userID,
		AgentID:   req.AgentID,
		Mode:      mode,
		Status:    types.PatchStatusPending,
		ScriptURL: req.ScriptURL,
		CreatedAt: record.GetDateTime("created").Time(),
		UpdatedAt: record.GetDateTime("updated").Time(),
	}, nil
}

// GetPatchOperations retrieves all patch operations for a user
func GetPatchOperations(app core.App, userID string) ([]*types.PatchResponse, error) {
	collection, err := app.FindCollectionByNameOrId("patch_operations")
	if err != nil {
		return nil, fmt.Errorf("patch_operations collection not found: %w", err)
	}

	// List patch operations for this user
	records, err := app.FindRecordsByFilter(collection, "user_id = {:uid}", "", 0, 0, map[string]interface{}{"uid": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch patch operations: %w", err)
	}

	var operations []*types.PatchResponse
	for _, record := range records {
		operations = append(operations, &types.PatchResponse{
			ID:        record.Id,
			UserID:    userID,
			AgentID:   record.GetString("agent_id"),
			Mode:      types.PatchMode(record.GetString("mode")),
			Status:    types.PatchStatus(record.GetString("status")),
			ScriptURL: record.GetString("script_url"),
			CreatedAt: record.GetDateTime("created").Time(),
			UpdatedAt: record.GetDateTime("updated").Time(),
		})
	}

	return operations, nil
}

// GetPatchOperation retrieves a single patch operation by ID
func GetPatchOperation(app core.App, userID, operationID string) (*types.PatchResponse, error) {
	collection, err := app.FindCollectionByNameOrId("patch_operations")
	if err != nil {
		return nil, fmt.Errorf("patch_operations collection not found: %w", err)
	}

	record, err := app.FindRecordById(collection.Id, operationID)
	if err != nil {
		return nil, fmt.Errorf("patch operation not found: %w", err)
	}

	// Verify ownership
	if record.GetString("user_id") != userID {
		return nil, fmt.Errorf("unauthorized: operation does not belong to user")
	}

	return &types.PatchResponse{
		ID:        record.Id,
		UserID:    userID,
		AgentID:   record.GetString("agent_id"),
		Mode:      types.PatchMode(record.GetString("mode")),
		Status:    types.PatchStatus(record.GetString("status")),
		ScriptURL: record.GetString("script_url"),
		CreatedAt: record.GetDateTime("created").Time(),
		UpdatedAt: record.GetDateTime("updated").Time(),
	}, nil
}

// UpdatePatchStatus updates patch operation status
func UpdatePatchStatus(
	app core.App,
	userID, operationID string,
	status types.PatchStatus,
	outputPath string,
	errorMsg string,
) error {
	collection, err := app.FindCollectionByNameOrId("patch_operations")
	if err != nil {
		return fmt.Errorf("patch_operations collection not found: %w", err)
	}

	record, err := app.FindRecordById(collection.Id, operationID)
	if err != nil {
		return fmt.Errorf("patch operation not found: %w", err)
	}

	// Verify ownership
	if record.GetString("user_id") != userID {
		return fmt.Errorf("unauthorized: operation does not belong to user")
	}

	record.Set("status", string(status))
	if status == types.PatchStatusRunning {
		record.Set("started_at", time.Now())
	}
	if status == types.PatchStatusCompleted || status == types.PatchStatusFailed {
		record.Set("completed_at", time.Now())
	}
	if outputPath != "" {
		record.Set("output_path", outputPath)
	}
	if errorMsg != "" {
		record.Set("error_msg", errorMsg)
	}

	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to update patch operation: %w", err)
	}

	return nil
}

// CreatePackageUpdate creates a package update record from patch result
func CreatePackageUpdate(
	app core.App,
	patchOpID string,
	pkg types.PatchPackageInfo,
) error {
	collection, err := app.FindCollectionByNameOrId("package_updates")
	if err != nil {
		return fmt.Errorf("package_updates collection not found: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("patch_op_id", patchOpID)
	record.Set("package_name", pkg.Name)
	record.Set("target_ver", pkg.Version)
	record.Set("update_type", pkg.UpdateType)
	record.Set("status", "applied")

	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to save package update: %w", err)
	}

	return nil
}

// HandlePatchOperations handles patch operation API endpoints
func HandlePatchOperations(app core.App, c *core.RequestEvent) error {
	// Get authenticated user
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
	}
	user := authRecord.(*core.Record)

	// Determine action based on method
	switch c.Request.Method {
	case http.MethodPost:
		return handleCreatePatchOperation(app, c, user.Id)
	case http.MethodGet:
		// Check if getting specific operation or list
		pathID := c.Request.URL.Query().Get("id")
		if pathID != "" {
			return handleGetPatchOperation(app, c, user.Id, pathID)
		}
		return handleListPatchOperations(app, c, user.Id)
	case http.MethodPatch:
		pathID := c.Request.URL.Query().Get("id")
		if pathID == "" {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "id parameter required"})
		}
		return handleUpdatePatchOperation(app, c, user.Id, pathID)
	}

	return c.JSON(http.StatusMethodNotAllowed, types.ErrorResponse{Error: "method not allowed"})
}

func handleCreatePatchOperation(app core.App, c *core.RequestEvent, userID string) error {
	var req types.PatchRequest
	if err := c.BindBody(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request"})
	}

	resp, err := CreatePatchOperation(app, userID, req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, resp)
}

func handleListPatchOperations(app core.App, c *core.RequestEvent, userID string) error {
	operations, err := GetPatchOperations(app, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, operations)
}

func handleGetPatchOperation(app core.App, c *core.RequestEvent, userID, operationID string) error {
	operation, err := GetPatchOperation(app, userID, operationID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, operation)
}

func handleUpdatePatchOperation(app core.App, c *core.RequestEvent, userID, operationID string) error {
	var updateReq struct {
		Status     string `json:"status"`
		OutputPath string `json:"output_path"`
		ErrorMsg   string `json:"error_msg"`
	}
	if err := c.BindBody(&updateReq); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request"})
	}

	if updateReq.Status != "" {
		err := UpdatePatchStatus(app, userID, operationID, types.PatchStatus(updateReq.Status), updateReq.OutputPath, updateReq.ErrorMsg)
		if err != nil {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"success": "true"})
}
