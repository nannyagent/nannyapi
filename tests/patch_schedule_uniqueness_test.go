package tests

import (
	"log"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/hooks"
	"github.com/nannyagent/nannyapi/internal/schedules"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPatchScheduleUniqueness(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}

	// Register hooks
	hooks.RegisterPatchHooks(app)
	schedules.RegisterScheduler(app)

	// Create a user and agent
	// Note: We need a valid agent setup
	userColl, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(userColl)
	user.Set("email", "schedule_unique@example.com")
	user.Set("password", "Pass123!456")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}

	// Create device code for agent
	dcColl, _ := app.FindCollectionByNameOrId("device_codes")
	dc := core.NewRecord(dcColl)
	dc.Set("device_code", "unique-dc-123")
	dc.Set("user_code", "12345678")
	dc.Set("expires_at", time.Now().Add(time.Hour))
	if err := app.Save(dc); err != nil {
		t.Fatal(err)
	}

	agentColl, _ := app.FindCollectionByNameOrId("agents")
	agent := core.NewRecord(agentColl)
	agent.Set("name", "test-agent")
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", dc.Id)
	agent.Set("hostname", "host1")
	agent.Set("platform_family", "debian")
	agent.Set("version", "1.0")
	agent.SetPassword("AgentPass123!")
	if err := app.Save(agent); err != nil {
		t.Fatal(err)
	}

	schedColl, _ := app.FindCollectionByNameOrId("patch_schedules")

	// 1. Create first schedule for agent (host)
	s1 := core.NewRecord(schedColl)
	s1.Set("user_id", user.Id)
	s1.Set("agent_id", agent.Id)
	s1.Set("lxc_id", "")
	s1.Set("cron_expression", "0 0 * * *")
	s1.Set("is_active", true)

	if err := app.Save(s1); err != nil {
		t.Fatalf("Failed to save first schedule: %v", err)
	}

	// 2. Try to create duplicate schedule for same agent (host)
	s2 := core.NewRecord(schedColl)
	s2.Set("user_id", user.Id)
	s2.Set("agent_id", agent.Id)
	s2.Set("lxc_id", "")
	s2.Set("cron_expression", "0 12 * * *")
	s2.Set("is_active", true)

	// This should fail
	err = app.Save(s2)
	if err == nil {
		t.Fatal("Expected error when saving duplicate schedule for agent, got nil")
	} else {
		log.Printf("Got expected error: %v", err)
	}

	// 3. Create schedule for LXC container
	// First create Node record
	nodeColl, _ := app.FindCollectionByNameOrId("proxmox_nodes")
	node := core.NewRecord(nodeColl)
	node.Set("name", "pve1")
	node.Set("agent_id", agent.Id)
	node.Set("px_node_id", 1)
	node.Set("ip", "10.0.0.1")
	node.Set("pve_version", "7.0")
	node.Set("recorded_at", time.Now())
	if err := app.Save(node); err != nil {
		t.Fatalf("Failed to save Node record: %v", err)
	}

	// First create LXC record
	lxcColl, _ := app.FindCollectionByNameOrId("proxmox_lxc")
	lxc := core.NewRecord(lxcColl)
	lxc.Set("vmid", 100)
	lxc.Set("name", "test-lxc")
	lxc.Set("agent_id", agent.Id)
	lxc.Set("lxc_id", "lxc/100") // This is the string ID field
	lxc.Set("node_id", node.Id)
	lxc.Set("ostype", "debian")
	lxc.Set("status", "running")
	lxc.Set("recorded_at", time.Now())

	if err := app.Save(lxc); err != nil {
		t.Fatalf("Failed to save LXC record: %v", err)
	}
	lxcDBID := lxc.Id

	s3 := core.NewRecord(schedColl)
	s3.Set("user_id", user.Id)
	s3.Set("agent_id", agent.Id)
	s3.Set("lxc_id", lxcDBID)
	s3.Set("cron_expression", "0 0 * * *")
	s3.Set("is_active", true)

	if err := app.Save(s3); err != nil {
		t.Fatalf("Failed to save LXC schedule: %v", err)
	}

	// 4. Try duplicate for same LXC
	s4 := core.NewRecord(schedColl)
	s4.Set("user_id", user.Id)
	s4.Set("agent_id", agent.Id)
	s4.Set("lxc_id", lxcDBID)
	s4.Set("cron_expression", "0 12 * * *")
	s4.Set("is_active", true)

	// This should fail
	err = app.Save(s4)
	if err == nil {
		t.Fatal("Expected error when saving duplicate schedule for LXC, got nil")
	} else {
		log.Printf("Got expected error for LXC: %v", err)
	}

	// 5. Update existing schedule (should succeed if unique)
	s1.Set("cron_expression", "0 12 * * *")
	if err := app.Save(s1); err != nil {
		t.Fatalf("Failed to update existing schedule: %v", err)
	}
}
