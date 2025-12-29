package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/hooks"
	"github.com/nannyagent/nannyapi/internal/schedules"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func setupPatchScheduleTestApp(t *testing.T) (*tests.TestApp, func()) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}

	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}

	// Register hooks
	hooks.RegisterPatchHooks(app)
	hooks.RegisterProxmoxHooks(app)
	schedules.RegisterScheduler(app)

	return app, func() {
		app.Cleanup()
	}
}

func createTestScript(t *testing.T, app core.App, platformFamily string) {
	scriptsCollection, err := app.FindCollectionByNameOrId("scripts")
	if err != nil {
		t.Fatal(err)
	}

	// Check if exists
	existing, _ := app.FindRecordsByFilter(scriptsCollection.Id, "platform_family = {:pf}", "", 1, 0, map[string]interface{}{"pf": platformFamily})
	if len(existing) > 0 {
		return
	}

	script := core.NewRecord(scriptsCollection)
	script.Set("name", "test-script-"+platformFamily)
	script.Set("platform_family", platformFamily)
	script.Set("os_type", "linux")

	content := []byte("#!/bin/bash\necho test")
	file, err := filesystem.NewFileFromBytes(content, "test.sh")
	if err != nil {
		t.Fatal(err)
	}
	script.Set("file", file)

	// SHA256
	h := sha256.New()
	h.Write(content)
	script.Set("sha256", hex.EncodeToString(h.Sum(nil)))

	if err := app.Save(script); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}
}

func TestPatchScheduling(t *testing.T) {
	app, cleanup := setupPatchScheduleTestApp(t)
	defer cleanup()

	// 1. Setup data
	email := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
	user := createTestUser(app, t, email, "password123")
	agent := createTestAgent(app, t, user.Id, "test-agent")

	// Create script for agent platform
	createTestScript(t, app, agent.GetString("platform_family"))

	// 2. Create Schedule for Agent
	schedulesCollection, err := app.FindCollectionByNameOrId("patch_schedules")
	if err != nil {
		t.Fatal(err)
	}

	schedule := core.NewRecord(schedulesCollection)
	schedule.Set("user_id", user.Id)
	schedule.Set("agent_id", agent.Id)
	schedule.Set("cron_expression", "* * * * *") // Every minute
	schedule.Set("is_active", true)

	if err := app.Save(schedule); err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}

	// Verify next_run_at is set
	nextRun := schedule.GetDateTime("next_run_at")
	if nextRun.IsZero() {
		t.Fatal("next_run_at should be set")
	}

	// Force next_run_at to be in the past to trigger execution
	// Fetch fresh record to ensure Original is set correctly for update hook
	schedule, err = app.FindRecordById("patch_schedules", schedule.Id)
	if err != nil {
		t.Fatal(err)
	}
	schedule.Set("next_run_at", time.Now().Add(-1*time.Minute))
	if err := app.Save(schedule); err != nil {
		t.Fatal(err)
	}

	// 3. Run Scheduler Logic
	schedules.ExecuteSchedule(app, schedule.Id)

	// 4. Verify Patch Operation Created
	patchOps, err := app.FindCollectionByNameOrId("patch_operations")
	if err != nil {
		t.Fatal(err)
	}

	ops, err := app.FindRecordsByFilter(patchOps.Id, "agent_id = {:agentID}", "", 1, 0, map[string]interface{}{
		"agentID": agent.Id,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) == 0 {
		t.Fatal("Patch operation should have been created")
	}

	op := ops[0]
	if op.GetString("status") != "pending" {
		t.Errorf("Expected status pending, got %s", op.GetString("status"))
	}
	if op.GetString("mode") != "apply" {
		t.Errorf("Expected mode apply, got %s", op.GetString("mode"))
	}
	if op.GetString("script_url") == "" {
		t.Error("script_url should be populated by hook")
	}

	// Verify schedule updated
	refreshedSchedule, err := app.FindRecordById(schedulesCollection, schedule.Id)
	if err != nil {
		t.Fatal(err)
	}
	if refreshedSchedule.GetDateTime("last_run_at").IsZero() {
		t.Error("last_run_at should be updated")
	}
	if refreshedSchedule.GetDateTime("next_run_at").Time().Before(time.Now()) {
		t.Error("next_run_at should be in the future")
	}
}

func TestLxcPatchScheduling(t *testing.T) {
	app, cleanup := setupPatchScheduleTestApp(t)
	defer cleanup()

	email := fmt.Sprintf("test-lxc-%d@example.com", time.Now().UnixNano())
	user := createTestUser(app, t, email, "password123")
	agent := createTestAgent(app, t, user.Id, "test-agent-2")

	// Create Node
	nodesCollection, err := app.FindCollectionByNameOrId("proxmox_nodes")
	if err != nil {
		t.Fatal(err)
	}
	node := core.NewRecord(nodesCollection)
	node.Set("name", "test-node")
	node.Set("agent_id", agent.Id)
	node.Set("status", "online")
	node.Set("uptime", "1000")
	node.Set("ip", "192.168.1.1")
	node.Set("pve_version", "8.0")
	node.Set("recorded_at", time.Now())
	if err := app.Save(node); err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	// Create LXC
	lxcCollection, err := app.FindCollectionByNameOrId("proxmox_lxc")
	if err != nil {
		t.Fatal(err)
	}
	lxc := core.NewRecord(lxcCollection)
	lxc.Set("lxc_id", "100")
	lxc.Set("name", "test-lxc")
	lxc.Set("status", "running")
	lxc.Set("ostype", "alpine")
	lxc.Set("vmid", 100)
	lxc.Set("recorded_at", time.Now())
	lxc.Set("agent_id", agent.Id)
	lxc.Set("user_id", user.Id)
	lxc.Set("node_id", node.Id)
	if err := app.Save(lxc); err != nil {
		t.Fatalf("Failed to create LXC: %v", err)
	}

	// Create script for LXC platform (alpine)
	createTestScript(t, app, "alpine")

	// Create Schedule for LXC
	schedulesCollection, err := app.FindCollectionByNameOrId("patch_schedules")
	if err != nil {
		t.Fatal(err)
	}

	schedule := core.NewRecord(schedulesCollection)
	schedule.Set("user_id", user.Id)
	schedule.Set("lxc_id", lxc.Id)
	schedule.Set("agent_id", agent.Id) // Agent is needed for routing
	schedule.Set("cron_expression", "* * * * *")
	schedule.Set("is_active", true)

	if err := app.Save(schedule); err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}

	// Force execution
	schedule, err = app.FindRecordById("patch_schedules", schedule.Id)
	if err != nil {
		t.Fatal(err)
	}
	schedule.Set("next_run_at", time.Now().Add(-1*time.Minute))
	if err := app.Save(schedule); err != nil {
		t.Fatal(err)
	}

	schedules.ExecuteSchedule(app, schedule.Id)

	// Verify
	patchOps, err := app.FindCollectionByNameOrId("patch_operations")
	if err != nil {
		t.Fatal(err)
	}

	ops, err := app.FindRecordsByFilter(patchOps.Id, "lxc_id = {:lxcID}", "", 1, 0, map[string]interface{}{
		"lxcID": lxc.Id,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) == 0 {
		t.Fatal("Patch operation should have been created for LXC")
	}

	op := ops[0]
	if op.GetString("vmid") != "100" {
		t.Errorf("Expected vmid 100, got %s", op.GetString("vmid"))
	}
}
