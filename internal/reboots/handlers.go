package reboots

import (
	"fmt"
	"time"

	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
)

// CreateReboot creates a new reboot operation record
// Only authenticated users can create reboots (not agents)
func CreateReboot(app core.App, userID string, req types.RebootRequest) (*types.RebootResponse, error) {
	// Validation
	if req.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	// Get reboot_operations collection
	collection, err := app.FindCollectionByNameOrId("reboot_operations")
	if err != nil {
		return nil, fmt.Errorf("reboot_operations collection not found: %w", err)
	}

	// Verify agent exists and belongs to user
	agent, err := app.FindRecordById("agents", req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	if agent.GetString("user_id") != userID {
		return nil, fmt.Errorf("unauthorized: agent does not belong to user")
	}

	// Check if agent already has a pending reboot
	if agent.GetString("pending_reboot_id") != "" {
		return nil, fmt.Errorf("agent already has a pending reboot")
	}

	// Validate LXC if provided
	vmid := 0
	if req.LxcID != "" {
		lxcRecord, err := app.FindRecordById("proxmox_lxc", req.LxcID)
		if err != nil {
			return nil, fmt.Errorf("lxc container not found: %w", err)
		}

		// Verify LXC belongs to this agent
		if lxcRecord.GetString("agent_id") != req.AgentID {
			return nil, fmt.Errorf("lxc container does not belong to the specified agent")
		}

		vmid = lxcRecord.GetInt("vmid")
	}

	// Set default timeout
	timeout := req.TimeoutSeconds
	if timeout == 0 {
		timeout = 300 // Default 5 minutes
	}

	now := time.Now().UTC()

	// Create reboot operation record
	record := core.NewRecord(collection)
	record.Set("user_id", userID)
	record.Set("agent_id", req.AgentID)
	record.Set("lxc_id", req.LxcID)
	record.Set("status", string(types.RebootStatusPending))
	record.Set("reason", req.Reason)
	record.Set("requested_at", now)
	record.Set("timeout_seconds", timeout)

	if vmid > 0 {
		record.Set("vmid", vmid)
	}

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save reboot operation: %w", err)
	}

	return &types.RebootResponse{
		ID:             record.Id,
		UserID:         userID,
		AgentID:        req.AgentID,
		LxcID:          req.LxcID,
		Vmid:           vmid,
		Status:         types.RebootStatusSent, // After save, hook sets to sent
		Reason:         req.Reason,
		TimeoutSeconds: timeout,
		RequestedAt:    now,
		CreatedAt:      record.GetDateTime("created").Time(),
		UpdatedAt:      record.GetDateTime("updated").Time(),
	}, nil
}

// ListReboots retrieves reboot operations for a user with optional filters
func ListReboots(app core.App, userID, agentID, status string) (*types.RebootListResponse, error) {
	filter := "user_id = {:userId}"
	params := map[string]interface{}{"userId": userID}

	if agentID != "" {
		filter += " && agent_id = {:agentId}"
		params["agentId"] = agentID
	}
	if status != "" {
		filter += " && status = {:status}"
		params["status"] = status
	}

	records, err := app.FindRecordsByFilter("reboot_operations", filter, "-created", 50, 0, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reboot operations: %w", err)
	}

	var reboots []types.RebootListItem
	for _, r := range records {
		item := types.RebootListItem{
			ID:             r.Id,
			AgentID:        r.GetString("agent_id"),
			LxcID:          r.GetString("lxc_id"),
			Vmid:           r.GetInt("vmid"),
			Status:         types.RebootStatus(r.GetString("status")),
			Reason:         r.GetString("reason"),
			RequestedAt:    r.GetDateTime("requested_at").Time(),
			ErrorMessage:   r.GetString("error_message"),
			TimeoutSeconds: r.GetInt("timeout_seconds"),
		}

		// Handle optional timestamps
		if ackAt := r.GetDateTime("acknowledged_at"); !ackAt.IsZero() {
			t := ackAt.Time()
			item.AcknowledgedAt = &t
		}
		if compAt := r.GetDateTime("completed_at"); !compAt.IsZero() {
			t := compAt.Time()
			item.CompletedAt = &t
		}

		reboots = append(reboots, item)
	}

	return &types.RebootListResponse{Reboots: reboots}, nil
}

// AcknowledgeReboot acknowledges a reboot operation (agent only)
func AcknowledgeReboot(app core.App, agentID, rebootID string) (*types.RebootAcknowledgeResponse, error) {
	rebootOp, err := app.FindRecordById("reboot_operations", rebootID)
	if err != nil {
		return nil, fmt.Errorf("reboot operation not found")
	}

	// Verify agent owns this reboot
	if rebootOp.GetString("agent_id") != agentID {
		return nil, fmt.Errorf("not your reboot operation")
	}

	// Update status to rebooting
	rebootOp.Set("status", string(types.RebootStatusRebooting))
	rebootOp.Set("acknowledged_at", time.Now().UTC())

	if err := app.Save(rebootOp); err != nil {
		return nil, fmt.Errorf("failed to update reboot status: %w", err)
	}

	return &types.RebootAcknowledgeResponse{
		Success: true,
		Message: "Reboot acknowledged. Agent should proceed with reboot.",
	}, nil
}

// GetReboot retrieves a single reboot operation
func GetReboot(app core.App, userID, rebootID string) (*types.RebootResponse, error) {
	record, err := app.FindRecordById("reboot_operations", rebootID)
	if err != nil {
		return nil, fmt.Errorf("reboot operation not found")
	}

	// Verify ownership
	if record.GetString("user_id") != userID {
		return nil, fmt.Errorf("unauthorized: reboot operation does not belong to user")
	}

	return &types.RebootResponse{
		ID:             record.Id,
		UserID:         userID,
		AgentID:        record.GetString("agent_id"),
		LxcID:          record.GetString("lxc_id"),
		Vmid:           record.GetInt("vmid"),
		Status:         types.RebootStatus(record.GetString("status")),
		Reason:         record.GetString("reason"),
		TimeoutSeconds: record.GetInt("timeout_seconds"),
		RequestedAt:    record.GetDateTime("requested_at").Time(),
		CreatedAt:      record.GetDateTime("created").Time(),
		UpdatedAt:      record.GetDateTime("updated").Time(),
	}, nil
}
