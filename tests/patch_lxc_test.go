package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/hooks"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestPatchLXC(t *testing.T) {
	testApp, _ := tests.NewTestApp()
	defer testApp.Cleanup()

	// Run migrations
	if err := testApp.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register hooks
	hooks.RegisterPatchHooks(testApp)

	// 1. Setup Data
	usersCollection, err := testApp.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Failed to find users collection: %v", err)
	}
	user := core.NewRecord(usersCollection)
	user.Set("email", "lxc-patch-test@example.com")
	user.Set("password", "Test123456!@#")
	user.SetVerified(true)
	if err := testApp.Save(user); err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	// Create Device Code
	deviceCodesCollection, err := testApp.FindCollectionByNameOrId("device_codes")
	if err != nil {
		t.Fatalf("Failed to find device_codes collection: %v", err)
	}
	deviceCode := core.NewRecord(deviceCodesCollection)
	deviceCode.Set("device_code", "dev-lxc-123")
	deviceCode.Set("user_code", "USRLXC")
	deviceCode.Set("expires_at", time.Now().Add(time.Hour))
	if err := testApp.Save(deviceCode); err != nil {
		t.Fatalf("Failed to save device code: %v", err)
	}

	// Create Agent
	agentsCollection, err := testApp.FindCollectionByNameOrId("agents")
	if err != nil {
		t.Fatalf("Failed to find agents collection: %v", err)
	}
	agent := core.NewRecord(agentsCollection)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", deviceCode.Id)
	agent.Set("hostname", "host-node")
	agent.Set("platform_family", "debian")
	agent.Set("version", "1.0.0")
	agent.SetPassword("AgentPassLXC123!")
	if err := testApp.Save(agent); err != nil {
		t.Fatalf("Failed to save agent: %v", err)
	}

	// Create Script (Alpine)
	scriptsCollection, err := testApp.FindCollectionByNameOrId("scripts")
	if err != nil {
		t.Fatalf("Failed to find scripts collection: %v", err)
	}
	script := core.NewRecord(scriptsCollection)
	script.Set("name", "alpine-update")
	script.Set("platform_family", "alpine")

	scriptContent := "#!/bin/sh\napk update && apk upgrade"
	h := sha256.New()
	h.Write([]byte(scriptContent))
	sha256Hash := hex.EncodeToString(h.Sum(nil))
	script.Set("sha256", sha256Hash)

	f, _ := filesystem.NewFileFromBytes([]byte(scriptContent), "apk-update.sh")
	script.Set("file", f)

	if err := testApp.Save(script); err != nil {
		t.Fatalf("Failed to save script: %v", err)
	}

	// Create Proxmox Node
	nodesCollection, err := testApp.FindCollectionByNameOrId("proxmox_nodes")
	if err != nil {
		t.Fatalf("Failed to find proxmox_nodes collection: %v", err)
	}
	node := core.NewRecord(nodesCollection)
	node.Set("name", "pve-node-1")
	node.Set("node_id", "pve/node1")
	node.Set("agent_id", agent.Id)
	node.Set("user_id", user.Id)
	node.Set("status", "online")
	node.Set("ip", "192.168.1.10")
	node.Set("pve_version", "8.0.4")
	node.Set("recorded_at", time.Now())
	if err := testApp.Save(node); err != nil {
		t.Fatalf("Failed to save node: %v", err)
	}

	// Create LXC Container
	lxcCollection, err := testApp.FindCollectionByNameOrId("proxmox_lxc")
	if err != nil {
		t.Fatalf("Failed to find proxmox_lxc collection: %v", err)
	}
	lxc := core.NewRecord(lxcCollection)
	lxc.Set("name", "alpine-container")
	lxc.Set("vmid", 100)
	lxc.Set("lxc_id", "lxc/100")
	lxc.Set("node_id", node.Id)
	lxc.Set("ostype", "alpine")
	lxc.Set("status", "running")
	lxc.Set("agent_id", agent.Id)
	lxc.Set("user_id", user.Id)
	lxc.Set("recorded_at", time.Now())
	if err := testApp.Save(lxc); err != nil {
		t.Fatalf("Failed to save lxc: %v", err)
	}

	// 2. Test Patch Creation for LXC
	patchPayload := `{"agent_id": "` + agent.Id + `", "lxc_id": "` + lxc.Id + `", "mode": "dry-run"}`

	// Generate User Token
	userToken, err := user.NewAuthToken()
	if err != nil {
		t.Fatalf("Failed to generate user token: %v", err)
	}

	(&tests.ApiScenario{
		Name:   "Create LXC Patch Operation (User)",
		Method: "POST",
		URL:    "/api/patches",
		Body:   strings.NewReader(patchPayload),
		Headers: map[string]string{
			"Authorization": "Bearer " + userToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus: 201,
		ExpectedContent: []string{
			`"status":"pending"`,
			`"mode":"dry-run"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify Patch Operation Record
	patchOps, err := testApp.FindRecordsByFilter("patch_operations", "lxc_id = '"+lxc.Id+"'", "", 1, 0, nil)
	if err != nil || len(patchOps) == 0 {
		t.Fatalf("Failed to find patch operation for LXC")
	}
	op := patchOps[0]
	if op.GetString("script_id") != script.Id {
		t.Errorf("Expected script %s, got %s", script.Id, op.GetString("script_id"))
	}
	if op.GetString("vmid") != "100" {
		t.Errorf("Expected vmid 100, got %s", op.GetString("vmid"))
	}

	// 3. Test Invalid OSType
	lxcInvalid := core.NewRecord(lxcCollection)
	lxcInvalid.Set("name", "unknown-os")
	lxcInvalid.Set("vmid", 101)
	lxcInvalid.Set("lxc_id", "lxc/101")
	lxcInvalid.Set("node_id", node.Id)
	lxcInvalid.Set("ostype", "templeos")
	lxcInvalid.Set("status", "running")
	lxcInvalid.Set("agent_id", agent.Id)
	lxcInvalid.Set("user_id", user.Id)
	lxcInvalid.Set("recorded_at", time.Now())
	if err := testApp.Save(lxcInvalid); err != nil {
		t.Fatalf("Failed to save invalid lxc: %v", err)
	}

	invalidPayload := `{"agent_id": "` + agent.Id + `", "lxc_id": "` + lxcInvalid.Id + `", "mode": "dry-run"}`
	(&tests.ApiScenario{
		Name:   "Create LXC Patch Operation (Invalid OS)",
		Method: "POST",
		URL:    "/api/patches",
		Body:   strings.NewReader(invalidPayload),
		Headers: map[string]string{
			"Authorization": "Bearer " + userToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  400,
		ExpectedContent: []string{`unsupported lxc ostype`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)
}
