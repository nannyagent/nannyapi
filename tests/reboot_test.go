package tests

import (
	"log"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/hooks"
	"github.com/nannyagent/nannyapi/internal/reboots"
	"github.com/nannyagent/nannyapi/internal/types"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestRebootOperations(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}

	// Register hooks
	hooks.RegisterRebootHooks(app)

	// Create a user
	userColl, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(userColl)
	user.Set("email", "reboot_test@example.com")
	user.Set("password", "Pass123!456")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}

	// Create device code for agent
	dcColl, _ := app.FindCollectionByNameOrId("device_codes")
	dc := core.NewRecord(dcColl)
	dc.Set("device_code", "reboot-dc-123")
	dc.Set("user_code", "REBOOT01")
	dc.Set("expires_at", time.Now().Add(time.Hour))
	if err := app.Save(dc); err != nil {
		t.Fatal(err)
	}

	// Create agent
	agentColl, _ := app.FindCollectionByNameOrId("agents")
	agent := core.NewRecord(agentColl)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", dc.Id)
	agent.Set("hostname", "reboot-host")
	agent.Set("platform_family", "debian")
	agent.Set("version", "1.0")
	agent.SetPassword("AgentPass123!")
	if err := app.Save(agent); err != nil {
		t.Fatal(err)
	}

	// Test 1: Create reboot operation
	t.Run("Create reboot operation", func(t *testing.T) {
		rebootColl, _ := app.FindCollectionByNameOrId("reboot_operations")
		reboot := core.NewRecord(rebootColl)
		reboot.Set("user_id", user.Id)
		reboot.Set("agent_id", agent.Id)
		reboot.Set("reason", "Test reboot")
		reboot.Set("status", "pending")

		if err := app.Save(reboot); err != nil {
			t.Fatalf("Failed to create reboot operation: %v", err)
		}

		// Verify requested_at was auto-populated
		requestedAt := reboot.GetDateTime("requested_at")
		if requestedAt.IsZero() {
			t.Error("requested_at should be auto-populated")
		}

		// Verify timeout_seconds default
		timeout := reboot.GetInt("timeout_seconds")
		if timeout != 300 {
			t.Errorf("Expected default timeout 300, got %d", timeout)
		}

		log.Printf("Reboot operation created: %s, requested_at: %v, timeout: %d", reboot.Id, requestedAt, timeout)

		// Verify agent's pending_reboot_id was updated
		updatedAgent, err := app.FindRecordById("agents", agent.Id)
		if err != nil {
			t.Fatal(err)
		}

		pendingRebootID := updatedAgent.GetString("pending_reboot_id")
		if pendingRebootID != reboot.Id {
			t.Errorf("Expected pending_reboot_id %s, got %s", reboot.Id, pendingRebootID)
		}

		log.Printf("Agent pending_reboot_id updated correctly: %s", pendingRebootID)
	})

	// Test 2: Cannot create duplicate pending reboot for same agent via handler
	t.Run("Prevent duplicate pending reboot via handler", func(t *testing.T) {
		// Agent already has a pending reboot from test 1
		// Try creating via the reboots package (simulates API call)
		_, err := reboots.CreateReboot(app, user.Id, types.RebootRequest{
			AgentID: agent.Id,
			Reason:  "Duplicate reboot attempt",
		})

		if err == nil {
			t.Fatal("Expected error when creating duplicate reboot, got nil")
		}

		if err.Error() != "agent already has a pending reboot" {
			t.Errorf("Expected 'agent already has a pending reboot' error, got: %v", err)
		}
		log.Printf("Handler correctly prevented duplicate: %v", err)
	})
}

func TestRebootScheduleUniqueness(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}

	// Register hooks
	hooks.RegisterRebootHooks(app)

	// Create a user
	userColl, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(userColl)
	user.Set("email", "reboot_sched@example.com")
	user.Set("password", "Pass123!456")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}

	// Create device code for agent
	dcColl, _ := app.FindCollectionByNameOrId("device_codes")
	dc := core.NewRecord(dcColl)
	dc.Set("device_code", "reboot-sched-dc")
	dc.Set("user_code", "RBTSCHED")
	dc.Set("expires_at", time.Now().Add(time.Hour))
	if err := app.Save(dc); err != nil {
		t.Fatal(err)
	}

	// Create agent
	agentColl, _ := app.FindCollectionByNameOrId("agents")
	agent := core.NewRecord(agentColl)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", dc.Id)
	agent.Set("hostname", "reboot-sched-host")
	agent.Set("platform_family", "debian")
	agent.Set("version", "1.0")
	agent.SetPassword("AgentPass123!")
	if err := app.Save(agent); err != nil {
		t.Fatal(err)
	}

	schedColl, _ := app.FindCollectionByNameOrId("reboot_schedules")

	// Test 1: Create first reboot schedule for agent (host)
	t.Run("Create first reboot schedule", func(t *testing.T) {
		s1 := core.NewRecord(schedColl)
		s1.Set("user_id", user.Id)
		s1.Set("agent_id", agent.Id)
		s1.Set("lxc_id", "")
		s1.Set("cron_expression", "0 4 * * 0")
		s1.Set("reason", "Weekly maintenance")
		s1.Set("is_active", true)

		if err := app.Save(s1); err != nil {
			t.Fatalf("Failed to save first reboot schedule: %v", err)
		}
		log.Printf("Created first reboot schedule: %s", s1.Id)
	})

	// Test 2: Try to create duplicate schedule for same agent
	t.Run("Prevent duplicate reboot schedule", func(t *testing.T) {
		s2 := core.NewRecord(schedColl)
		s2.Set("user_id", user.Id)
		s2.Set("agent_id", agent.Id)
		s2.Set("lxc_id", "")
		s2.Set("cron_expression", "0 5 * * 1")
		s2.Set("reason", "Another schedule")
		s2.Set("is_active", true)

		err := app.Save(s2)
		if err == nil {
			t.Fatal("Expected error when saving duplicate reboot schedule, got nil")
		}
		log.Printf("Got expected error: %v", err)
	})

	// Test 3: Create LXC-specific schedule (requires LXC setup)
	t.Run("Create LXC reboot schedule", func(t *testing.T) {
		// Create Node first
		nodeColl, _ := app.FindCollectionByNameOrId("proxmox_nodes")
		node := core.NewRecord(nodeColl)
		node.Set("name", "pve-reboot")
		node.Set("agent_id", agent.Id)
		node.Set("px_node_id", 1)
		node.Set("ip", "10.0.0.1")
		node.Set("pve_version", "8.0")
		node.Set("recorded_at", time.Now())
		if err := app.Save(node); err != nil {
			t.Fatal(err)
		}

		// Create LXC
		lxcColl, _ := app.FindCollectionByNameOrId("proxmox_lxc")
		lxc := core.NewRecord(lxcColl)
		lxc.Set("name", "test-lxc-reboot")
		lxc.Set("agent_id", agent.Id)
		lxc.Set("node_id", node.Id)
		lxc.Set("vmid", 100)
		lxc.Set("lxc_id", "100")     // Required field
		lxc.Set("status", "running") // Required field
		lxc.Set("ostype", "ubuntu")
		lxc.Set("recorded_at", time.Now())
		if err := app.Save(lxc); err != nil {
			t.Fatal(err)
		}

		// Create LXC schedule
		s3 := core.NewRecord(schedColl)
		s3.Set("user_id", user.Id)
		s3.Set("agent_id", agent.Id)
		s3.Set("lxc_id", lxc.Id)
		s3.Set("cron_expression", "0 3 * * *")
		s3.Set("reason", "LXC daily reboot")
		s3.Set("is_active", true)

		if err := app.Save(s3); err != nil {
			t.Fatalf("Failed to save LXC reboot schedule: %v", err)
		}
		log.Printf("Created LXC reboot schedule: %s", s3.Id)

		// Try duplicate LXC schedule
		s4 := core.NewRecord(schedColl)
		s4.Set("user_id", user.Id)
		s4.Set("agent_id", agent.Id)
		s4.Set("lxc_id", lxc.Id)
		s4.Set("cron_expression", "0 6 * * *")
		s4.Set("is_active", true)

		err := app.Save(s4)
		if err == nil {
			t.Fatal("Expected error when saving duplicate LXC reboot schedule, got nil")
		}
		log.Printf("Got expected error for LXC: %v", err)
	})
}

func TestRebootCompletionDetection(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}

	hooks.RegisterRebootHooks(app)

	// Create user, agent, reboot operation
	userColl, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(userColl)
	user.Set("email", "reboot_complete@example.com")
	user.Set("password", "Pass123!456")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}

	dcColl, _ := app.FindCollectionByNameOrId("device_codes")
	dc := core.NewRecord(dcColl)
	dc.Set("device_code", "complete-dc")
	dc.Set("user_code", "COMPLETE")
	dc.Set("expires_at", time.Now().Add(time.Hour))
	if err := app.Save(dc); err != nil {
		t.Fatal(err)
	}

	agentColl, _ := app.FindCollectionByNameOrId("agents")
	agent := core.NewRecord(agentColl)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", dc.Id)
	agent.Set("hostname", "complete-host")
	agent.Set("platform_family", "debian")
	agent.Set("version", "1.0")
	agent.Set("last_seen", time.Now().Add(-5*time.Minute)) // Set last_seen to past
	agent.SetPassword("AgentPass123!")
	if err := app.Save(agent); err != nil {
		t.Fatal(err)
	}

	// Create reboot operation first (hooks will set status to "sent")
	rebootColl, _ := app.FindCollectionByNameOrId("reboot_operations")
	reboot := core.NewRecord(rebootColl)
	reboot.Set("user_id", user.Id)
	reboot.Set("agent_id", agent.Id)
	reboot.Set("status", "pending")
	reboot.Set("timeout_seconds", 300)
	if err := app.Save(reboot); err != nil {
		t.Fatal(err)
	}

	// Re-fetch to get the post-hook state
	reboot, _ = app.FindRecordById("reboot_operations", reboot.Id)

	// Manually set to "rebooting" state with acknowledged_at (simulating agent ack)
	reboot.Set("status", "rebooting")
	reboot.Set("acknowledged_at", time.Now().Add(-2*time.Minute))
	if err := app.Save(reboot); err != nil {
		t.Fatal(err)
	}

	// Re-fetch agent to get the pending_reboot_id set by hook
	agent, _ = app.FindRecordById("agents", agent.Id)

	t.Run("Detect reconnection after reboot", func(t *testing.T) {
		// Simulate agent reconnection by updating last_seen to now
		agent.Set("last_seen", time.Now())
		if err := app.Save(agent); err != nil {
			t.Fatal(err)
		}

		// Call the completion check directly
		hooks.CheckRebootCompletions(app)

		// Verify reboot status changed to completed
		updatedReboot, err := app.FindRecordById("reboot_operations", reboot.Id)
		if err != nil {
			t.Fatal(err)
		}

		status := updatedReboot.GetString("status")
		if status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", status)
		}

		completedAt := updatedReboot.GetDateTime("completed_at")
		if completedAt.IsZero() {
			t.Error("completed_at should be set")
		}

		// Verify pending_reboot_id cleared
		updatedAgent, err := app.FindRecordById("agents", agent.Id)
		if err != nil {
			t.Fatal(err)
		}

		pendingID := updatedAgent.GetString("pending_reboot_id")
		if pendingID != "" {
			t.Errorf("Expected pending_reboot_id to be cleared, got '%s'", pendingID)
		}

		log.Printf("Reboot completion detected: status=%s, completed_at=%v", status, completedAt)
	})
}

func TestUserOnlyRebootCreation(t *testing.T) {
	// This test verifies that the API rules prevent agents from creating reboots
	// The actual API-level enforcement is in the handler, but we can verify collection rules
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}

	// Check collection rules
	rebootOps, err := app.FindCollectionByNameOrId("reboot_operations")
	if err != nil {
		t.Fatal(err)
	}

	// Verify CreateRule is user-only
	if rebootOps.CreateRule == nil {
		t.Error("CreateRule should be set")
	} else {
		t.Logf("CreateRule: %s", *rebootOps.CreateRule)
		if *rebootOps.CreateRule != "@request.auth.id = user_id" {
			t.Error("CreateRule should restrict to user_id match")
		}
	}

	log.Printf("Verified reboot_operations collection rules are user-only")
}
