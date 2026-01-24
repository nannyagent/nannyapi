package hooks

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nannyagent/nannyapi/internal/reboots"
	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRebootHooks registers all reboot management endpoints and hooks
// Note: Reboot schedule cron management is handled by schedules.RegisterRebootScheduler
func RegisterRebootHooks(app core.App) {
	// Hook to enforce unique reboot schedule per agent/lxc (same as patch schedules)
	validateUniqueRebootSchedule := func(e *core.RecordEvent) error {
		agentID := e.Record.GetString("agent_id")
		lxcID := e.Record.GetString("lxc_id")

		filter := ""
		params := map[string]interface{}{
			"agent": agentID,
			"id":    e.Record.Id,
		}

		if lxcID == "" {
			filter = "agent_id = {:agent} && lxc_id = '' && id != {:id}"
		} else {
			filter = "agent_id = {:agent} && lxc_id = {:lxc} && id != {:id}"
			params["lxc"] = lxcID
		}

		records, err := app.FindRecordsByFilter("reboot_schedules", filter, "", 1, 0, params)
		if err != nil {
			return err
		}

		if len(records) > 0 {
			if lxcID != "" {
				return fmt.Errorf("a reboot schedule already exists for this LXC container (agent: %s, lxc: %s)", agentID, lxcID)
			}
			return fmt.Errorf("a reboot schedule already exists for this agent (%s)", agentID)
		}
		return e.Next()
	}

	app.OnRecordCreate("reboot_schedules").BindFunc(validateUniqueRebootSchedule)
	app.OnRecordUpdate("reboot_schedules").BindFunc(validateUniqueRebootSchedule)

	// Hook to set requested_at and validate on reboot operation create
	app.OnRecordCreate("reboot_operations").BindFunc(func(e *core.RecordEvent) error {
		// Set requested_at if not provided
		if e.Record.GetDateTime("requested_at").IsZero() {
			e.Record.Set("requested_at", time.Now().UTC())
		}

		// Set default timeout if not provided (300 seconds = 5 minutes)
		if e.Record.GetInt("timeout_seconds") == 0 {
			e.Record.Set("timeout_seconds", 300)
		}

		// Set status to "sent" directly - this ensures the CREATE event has the correct status
		// Previously we set "pending" here and updated to "sent" in AfterCreateSuccess,
		// but that second save doesn't properly broadcast realtime events from cron context
		e.Record.Set("status", "sent")

		// vmid is populated by the handler before save (from lxc_id lookup), similar to patches

		return e.Next()
	})

	// After create hook - only update agent's pending_reboot_id (don't modify e.Record again)
	app.OnRecordAfterCreateSuccess("reboot_operations").BindFunc(func(e *core.RecordEvent) error {
		agentID := e.Record.GetString("agent_id")

		// Update agent's pending_reboot_id
		agent, err := app.FindRecordById("agents", agentID)
		if err != nil {
			app.Logger().Error("Failed to find agent for reboot", "agent_id", agentID, "error", err)
			return e.Next()
		}

		agent.Set("pending_reboot_id", e.Record.Id)
		if err := app.Save(agent); err != nil {
			app.Logger().Error("Failed to update agent pending_reboot_id", "agent_id", agentID, "error", err)
		}

		return e.Next()
	})

	// Register API endpoints on startup
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Register reboot API endpoints
		withAuth := func(handler func(*core.RequestEvent) error) func(*core.RequestEvent) error {
			return LoadAuthContext(app)(RequireAuth()(handler))
		}

		// POST /api/reboot - Create reboot operation (user only)
		e.Router.POST("/api/reboot", withAuth(handleCreateReboot(app)))

		// GET /api/reboot - List reboot operations
		e.Router.GET("/api/reboot", withAuth(handleListReboots(app)))

		// GET /api/reboot/{id} - Get single reboot operation
		e.Router.GET("/api/reboot/{id}", withAuth(handleGetReboot(app)))

		// POST /api/reboot/{id}/acknowledge - Agent acknowledges reboot
		e.Router.POST("/api/reboot/{id}/acknowledge", withAuth(handleRebootAcknowledge(app)))

		// Start background job to check for reconnected agents after reboot
		go startRebootMonitor(app)

		return e.Next()
	})
}

// handleCreateReboot handles POST /api/reboot (user-initiated reboot)
func handleCreateReboot(app core.App) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		// Only authenticated users can create reboots
		user := c.Get("authRecord").(*core.Record)
		if user == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		// Check if this is a user (not an agent)
		if user.Collection().Name == "agents" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "agents cannot initiate reboots"})
		}

		var req types.RebootRequest
		if err := c.BindBody(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		resp, err := reboots.CreateReboot(app, user.Id, req)
		if err != nil {
			// Determine appropriate status code
			errMsg := err.Error()
			switch {
			case errMsg == "agent not found":
				return c.JSON(http.StatusNotFound, map[string]string{"error": errMsg})
			case errMsg == "unauthorized: agent does not belong to user":
				return c.JSON(http.StatusForbidden, map[string]string{"error": errMsg})
			case errMsg == "agent already has a pending reboot":
				return c.JSON(http.StatusConflict, map[string]string{"error": errMsg})
			default:
				return c.JSON(http.StatusBadRequest, map[string]string{"error": errMsg})
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":   true,
			"reboot_id": resp.ID,
			"status":    resp.Status,
			"message":   "Reboot command sent. Agent will receive via realtime subscription.",
		})
	}
}

// handleListReboots handles GET /api/reboot
func handleListReboots(app core.App) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		authRecord := c.Get("authRecord").(*core.Record)
		if authRecord == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		var userID string
		if authRecord.Collection().Name == "users" {
			userID = authRecord.Id
		} else if authRecord.Collection().Name == "agents" {
			userID = authRecord.GetString("user_id")
			if userID == "" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "agent has no owner"})
			}
		} else {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "invalid authentication type"})
		}

		agentID := c.Request.URL.Query().Get("agent_id")
		status := c.Request.URL.Query().Get("status")

		resp, err := reboots.ListReboots(app, userID, agentID, status)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, resp)
	}
}

// handleGetReboot handles GET /api/reboot/{id}
func handleGetReboot(app core.App) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		authRecord := c.Get("authRecord").(*core.Record)
		if authRecord == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		var userID string
		if authRecord.Collection().Name == "users" {
			userID = authRecord.Id
		} else if authRecord.Collection().Name == "agents" {
			userID = authRecord.GetString("user_id")
			if userID == "" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "agent has no owner"})
			}
		} else {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "invalid authentication type"})
		}

		rebootID := c.Request.PathValue("id")
		if rebootID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "reboot_id required"})
		}

		resp, err := reboots.GetReboot(app, userID, rebootID)
		if err != nil {
			errMsg := err.Error()
			if errMsg == "reboot operation not found" {
				return c.JSON(http.StatusNotFound, map[string]string{"error": errMsg})
			}
			if errMsg == "unauthorized: reboot operation does not belong to user" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": errMsg})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": errMsg})
		}

		return c.JSON(http.StatusOK, resp)
	}
}

// handleRebootAcknowledge handles POST /api/reboot/{id}/acknowledge (agent acknowledges)
func handleRebootAcknowledge(app core.App) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		authRecord := c.Get("authRecord")
		if authRecord == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		agent, ok := authRecord.(*core.Record)
		if !ok || agent.Collection().Name != "agents" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "only agents can acknowledge reboots"})
		}

		rebootID := c.Request.PathValue("id")
		if rebootID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "reboot_id required"})
		}

		resp, err := reboots.AcknowledgeReboot(app, agent.Id, rebootID)
		if err != nil {
			errMsg := err.Error()
			if errMsg == "reboot operation not found" {
				return c.JSON(http.StatusNotFound, map[string]string{"error": errMsg})
			}
			if errMsg == "not your reboot operation" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": errMsg})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": errMsg})
		}

		return c.JSON(http.StatusOK, resp)
	}
}

// startRebootMonitor periodically checks for agents that have reconnected after reboot
func startRebootMonitor(app core.App) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		CheckRebootCompletions(app)
	}
}

// CheckRebootCompletions checks for agents with pending_reboot_id that have reconnected
// Exported for testing
func CheckRebootCompletions(app core.App) {
	// Find agents with pending_reboot_id set
	agents, err := app.FindRecordsByFilter("agents", "pending_reboot_id != ''", "", 100, 0, nil)
	if err != nil {
		return
	}

	for _, agent := range agents {
		rebootID := agent.GetString("pending_reboot_id")
		if rebootID == "" {
			continue
		}

		rebootOp, err := app.FindRecordById("reboot_operations", rebootID)
		if err != nil {
			// Reboot operation deleted, clear the reference
			agent.Set("pending_reboot_id", "")
			_ = app.Save(agent)
			continue
		}

		status := rebootOp.GetString("status")

		// If status is rebooting, check if agent has reconnected (last_seen updated)
		if status == "rebooting" {
			acknowledgedAt := rebootOp.GetDateTime("acknowledged_at").Time()
			lastSeen := agent.GetDateTime("last_seen").Time()

			// If last_seen is after acknowledged_at, agent has reconnected
			if lastSeen.After(acknowledgedAt) {
				rebootOp.Set("status", "completed")
				rebootOp.Set("completed_at", time.Now().UTC())
				if err := app.Save(rebootOp); err != nil {
					app.Logger().Error("Failed to mark reboot as completed", "reboot_id", rebootID, "error", err)
					continue
				}

				// Clear pending_reboot_id
				agent.Set("pending_reboot_id", "")
				if err := app.Save(agent); err != nil {
					app.Logger().Error("Failed to clear pending_reboot_id", "agent_id", agent.Id, "error", err)
				}
			} else {
				// Check for timeout
				timeoutSeconds := rebootOp.GetInt("timeout_seconds")
				if timeoutSeconds == 0 {
					timeoutSeconds = 300
				}
				if time.Since(acknowledgedAt) > time.Duration(timeoutSeconds)*time.Second {
					rebootOp.Set("status", "timeout")
					rebootOp.Set("error_message", "Agent did not reconnect within timeout period")
					_ = app.Save(rebootOp)

					// Clear pending_reboot_id
					agent.Set("pending_reboot_id", "")
					_ = app.Save(agent)
				}
			}
		} else if status == "sent" {
			// Check if sent but never acknowledged (stale)
			requestedAt := rebootOp.GetDateTime("requested_at").Time()
			timeoutSeconds := rebootOp.GetInt("timeout_seconds")
			if timeoutSeconds == 0 {
				timeoutSeconds = 300
			}
			if time.Since(requestedAt) > time.Duration(timeoutSeconds)*time.Second {
				rebootOp.Set("status", "timeout")
				rebootOp.Set("error_message", "Agent did not acknowledge reboot command")
				_ = app.Save(rebootOp)

				agent.Set("pending_reboot_id", "")
				_ = app.Save(agent)
			}
		}
	}
}
