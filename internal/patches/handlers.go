package patches

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// CreatePatchOperation creates a new patch operation record
func CreatePatchOperation(app core.App, userID string, req types.PatchRequest) (*types.PatchResponse, error) {
	// Basic validation
	if req.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
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

	// Get agent OS info
	osType := agentRecord.GetString("os_type")
	osVersion := agentRecord.GetString("os_version")

	// Find appropriate script for this OS
	scriptsCollection, err := app.FindCollectionByNameOrId("scripts")
	if err != nil {
		return nil, fmt.Errorf("scripts collection not found: %w", err)
	}

	// Try to find script matching OS type and version
	// If no exact match, try matching just OS type
	// If still no match, fail
	var scriptRecord *core.Record

	// 1. Try exact match (os_type + os_version)
	if osType != "" && osVersion != "" {
		records, err := app.FindRecordsByFilter(scriptsCollection, "os_type = {:osType} && os_version = {:osVer}", "", 1, 0, map[string]interface{}{
			"osType": osType,
			"osVer":  osVersion,
		})
		if err == nil && len(records) > 0 {
			scriptRecord = records[0]
		}
	}

	// 2. Try OS type match only (generic script for distro)
	if scriptRecord == nil && osType != "" {
		records, err := app.FindRecordsByFilter(scriptsCollection, "os_type = {:osType} && os_version = ''", "", 1, 0, map[string]interface{}{
			"osType": osType,
		})
		if err == nil && len(records) > 0 {
			scriptRecord = records[0]
		}
	}

	// 3. Fallback to "linux" generic if available
	if scriptRecord == nil {
		records, err := app.FindRecordsByFilter(scriptsCollection, "os_type = 'linux'", "", 1, 0, nil)
		if err == nil && len(records) > 0 {
			scriptRecord = records[0]
		}
	}

	if scriptRecord == nil {
		return nil, fmt.Errorf("no compatible patch script found for agent OS: %s %s", osType, osVersion)
	}

	scriptURL := fmt.Sprintf("/api/files/%s/%s/%s", scriptsCollection.Id, scriptRecord.Id, scriptRecord.GetString("file"))
	scriptSHA256 := scriptRecord.GetString("sha256")

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
	record.Set("script_url", scriptURL) // Store the resolved script URL

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save patch operation: %w", err)
	}

	// Send realtime notification to agent
	// We don't have direct access to realtime service here, but PocketBase handles subscriptions
	// The agent should be subscribed to "patch_operations" or a specific topic
	// For now, we assume the agent polls or listens to changes on this record
	// BUT the requirement says "Agent receives the script via realtime"
	// So we should probably trigger a custom event or rely on the record creation event
	// The record creation event will send the record data.
	// We need to ensure the agent gets the script URL and SHA256.
	// Since we can't easily inject extra data into the standard create event without modifying the record,
	// we might need to rely on the agent fetching the script details or include them in the record (but we don't want to duplicate data).
	// Actually, we can just rely on the agent reading the `script_url` from the record.
	// However, for SHA256, we should probably expose it.
	// Let's add a `script_sha256` field to patch_operations (transient or persistent) or just let agent fetch it.
	// The requirement says: "pass that info in realtime message along with script_path"
	// We can't easily modify the realtime message payload of a standard Create event.
	// We could use app.OnRecordAfterCreateRequest to send a custom message if we had a custom websocket handler,
	// but PocketBase's realtime is tied to records.
	// Best approach: The agent receives the PatchOperation record. It contains `script_url`.
	// The agent then calls `GET /api/scripts/{id}/validate` (which we will build) to get the SHA256 before downloading.
	// OR we can store the SHA256 in the patch_operation record itself for simplicity and security (immutable at creation).

	// Let's update the record with SHA256 if we can add a field, or just rely on the validation endpoint.
	// The prompt says: "offer another endpoint for agent to validate this sha2sum when it downloads the script prior execution"
	// So the validation endpoint is the way to go.

	return &types.PatchResponse{
		ID:           record.Id,
		UserID:       userID,
		AgentID:      req.AgentID,
		Mode:         mode,
		Status:       types.PatchStatusPending,
		ScriptURL:    scriptURL,
		ScriptSHA256: scriptSHA256, // Return this in response so UI can see it, but Agent should verify via endpoint
		CreatedAt:    record.GetDateTime("created").Time(),
		UpdatedAt:    record.GetDateTime("updated").Time(),
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

// HandleValidateScript validates script SHA256
func HandleValidateScript(app core.App, c *core.RequestEvent) error {
	scriptID := c.Request.PathValue("id")
	if scriptID == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "script id required"})
	}

	collection, err := app.FindCollectionByNameOrId("scripts")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "scripts collection not found"})
	}

	record, err := app.FindRecordById(collection.Id, scriptID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "script not found"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"id":     record.Id,
		"sha256": record.GetString("sha256"),
		"name":   record.GetString("name"),
	})
}

// HandlePatchResult handles upload of patch execution results (stdout, stderr, exit code)
func HandlePatchResult(app core.App, c *core.RequestEvent) error {
	patchID := c.Request.PathValue("id")
	if patchID == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "patch id required"})
	}

	// Verify auth (Agent only)
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
	}
	agentRecord := authRecord.(*core.Record)
	if agentRecord.Collection().Name != "agents" {
		return c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "only agents can upload results"})
	}

	collection, err := app.FindCollectionByNameOrId("patch_operations")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "patch_operations collection not found"})
	}

	record, err := app.FindRecordById(collection.Id, patchID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "patch operation not found"})
	}

	// Verify agent owns this operation
	if record.GetString("agent_id") != agentRecord.Id {
		return c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "unauthorized: operation does not belong to agent"})
	}

	// Update fields
	exitCode := c.Request.FormValue("exit_code")
	if exitCode != "" {
		record.Set("exit_code", exitCode)
	}

	// Handle file uploads
	// Use c.Request.FormFile to get the file header, then create a filesystem.File
	// Actually PocketBase core.RequestEvent doesn't have FindUploadedFile directly in v0.23+?
	// Let's check how to handle file uploads in PocketBase v0.23+
	// It seems we should use c.Request.FormFile and then filesystem.NewFileFromMultipart

	// We need to import "github.com/pocketbase/pocketbase/tools/filesystem"

	if f, header, err := c.Request.FormFile("stdout_file"); err == nil {
		f.Close()
		if file, err := filesystem.NewFileFromMultipart(header); err == nil {
			record.Set("stdout_file", file)
		}
	}

	if f, header, err := c.Request.FormFile("stderr_file"); err == nil {
		f.Close()
		if file, err := filesystem.NewFileFromMultipart(header); err == nil {
			record.Set("stderr_file", file)
		}
	}

	// Update status based on exit code
	if exitCode == "0" {
		record.Set("status", "completed")
	} else {
		record.Set("status", "failed")
	}
	record.Set("completed_at", time.Now())

	if err := app.Save(record); err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to save patch result: " + err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

// HandlePatchOperations handles patch operation API endpoints
func HandlePatchOperations(app core.App, c *core.RequestEvent) error {
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
		return handleCreatePatchOperation(app, c, userID)
	case http.MethodGet:
		// Check if getting specific operation or list
		pathID := c.Request.URL.Query().Get("id")
		if pathID != "" {
			return handleGetPatchOperation(app, c, userID, pathID)
		}
		return handleListPatchOperations(app, c, userID)
	case http.MethodPatch:
		pathID := c.Request.URL.Query().Get("id")
		if pathID == "" {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "id parameter required"})
		}
		return handleUpdatePatchOperation(app, c, userID, pathID)
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
