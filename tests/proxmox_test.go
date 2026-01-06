package tests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/hooks"
	"github.com/nannyagent/nannyapi/internal/types"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func setupProxmoxTestApp(t *testing.T) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}

	// Run migrations
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register Proxmox hooks
	hooks.RegisterProxmoxHooks(app)

	return app
}

func createAgent(t *testing.T, app *tests.TestApp) (string, string) {
	// Create a user first (agents belong to users)
	userCollection, _ := app.FindCollectionByNameOrId("users")
	userRecord := core.NewRecord(userCollection)
	userRecord.Set("email", "agentowner@example.com")
	userRecord.Set("password", "Password123!")
	userRecord.Set("passwordConfirm", "Password123!")
	if err := app.Save(userRecord); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create device code
	deviceCodeCollection, _ := app.FindCollectionByNameOrId("device_codes")
	deviceCodeRecord := core.NewRecord(deviceCodeCollection)
	deviceCodeRecord.Set("device_code", "test-device-code")
	deviceCodeRecord.Set("user_code", "12345678")
	// Set expiry to 1 hour from now
	deviceCodeRecord.Set("expires_at", time.Now().Add(time.Hour))
	if err := app.Save(deviceCodeRecord); err != nil {
		t.Fatalf("Failed to create device code: %v", err)
	}

	// Create agent
	agentCollection, _ := app.FindCollectionByNameOrId("agents")
	agentRecord := core.NewRecord(agentCollection)
	agentRecord.Set("name", "test-agent")
	agentRecord.Set("user_id", userRecord.Id)
	agentRecord.Set("device_code_id", deviceCodeRecord.Id)
	agentRecord.Set("hostname", "test-hostname")
	agentRecord.Set("platform_family", "linux")
	agentRecord.Set("version", "1.0.0")
	agentRecord.SetPassword("AgentPass123!")

	if err := app.Save(agentRecord); err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Generate auth token for agent
	token, err := agentRecord.NewAuthToken()
	if err != nil {
		t.Fatalf("Failed to generate agent token: %v", err)
	}

	return token, agentRecord.Id
}

func createUserToken(t *testing.T, app *tests.TestApp) string {
	user, err := app.FindFirstRecordByFilter("users", "email='agentowner@example.com'")
	if err != nil {
		t.Fatalf("User not found: %v", err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatalf("Failed to generate user token: %v", err)
	}
	return token
}

func TestProxmoxIngest(t *testing.T) {
	app := setupProxmoxTestApp(t)
	defer app.Cleanup()

	token, _ := createAgent(t, app)

	// 1. Ingest Cluster
	clusterReq := types.ProxmoxCluster{
		ClusterName: "test-cluster",
		ClusterID:   "cluster-123",
		Nodes:       3,
		Quorate:     1,
		Version:     1,
	}
	clusterBody, _ := json.Marshal(clusterReq)

	(&tests.ApiScenario{
		Name:   "Ingest Cluster",
		Method: "POST",
		URL:    "/api/proxmox/cluster",
		Body:   strings.NewReader(string(clusterBody)),
		Headers: map[string]string{
			"Authorization": token,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{"Cluster ingested successfully"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify Cluster DB
	clusterRec, err := app.FindFirstRecordByFilter("proxmox_cluster", "px_cluster_id = 'cluster-123'")
	if err != nil {
		t.Fatalf("Cluster not found in DB: %v", err)
	}
	if clusterRec.GetString("cluster_name") != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got '%s'", clusterRec.GetString("cluster_name"))
	}

	// 2. Ingest Node
	nodeReq := types.ProxmoxNode{
		Name:       "pve-01",
		NodeID:     1,
		ClusterID:  "cluster-123",
		IP:         "192.168.1.10",
		Level:      "c",
		Local:      1,
		Online:     1,
		PVEVersion: "8.0.4",
	}
	nodeBody, _ := json.Marshal(nodeReq)

	(&tests.ApiScenario{
		Name:   "Ingest Node",
		Method: "POST",
		URL:    "/api/proxmox/node",
		Body:   strings.NewReader(string(nodeBody)),
		Headers: map[string]string{
			"Authorization": token,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{"Node ingested successfully"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify Node DB
	nodeRec, err := app.FindFirstRecordByFilter("proxmox_nodes", "name = 'pve-01'")
	if err != nil {
		t.Fatalf("Node not found in DB: %v", err)
	}
	if nodeRec.GetString("cluster_id") != clusterRec.Id {
		t.Errorf("Expected node to be linked to cluster %s, got %s", clusterRec.Id, nodeRec.GetString("cluster_id"))
	}

	// 3. Ingest LXC
	lxcReq := types.ProxmoxLXC{
		Node:   "pve-01",
		VMID:   100,
		Name:   "test-lxc",
		Status: "running",
		OSType: "debian",
		Uptime: 12345,
		LXCID:  "lxc-100",
	}
	lxcBody, _ := json.Marshal(lxcReq)

	(&tests.ApiScenario{
		Name:   "Ingest LXC",
		Method: "POST",
		URL:    "/api/proxmox/lxc",
		Body:   strings.NewReader(string(lxcBody)),
		Headers: map[string]string{
			"Authorization": token,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{"LXC ingested successfully"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify LXC DB
	lxcRec, err := app.FindFirstRecordByFilter("proxmox_lxc", "vmid = 100")
	if err != nil {
		t.Fatalf("LXC not found in DB: %v", err)
	}
	if lxcRec.GetString("node_id") != nodeRec.Id {
		t.Errorf("Expected LXC to be linked to node %s, got %s", nodeRec.Id, lxcRec.GetString("node_id"))
	}

	// 4. Ingest Qemu
	qemuReq := types.ProxmoxQemu{
		Node:    "pve-01",
		VMID:    101,
		Name:    "test-vm",
		Status:  "stopped",
		QemuID:  "qemu-101",
		OSType:  "l26",
		Uptime:  0,
		VMGenID: "some-gen-id",
		KVM:     1,
		Boot:    "order=scsi0",
		HostCPU: "host",
	}
	qemuBody, _ := json.Marshal(qemuReq)

	(&tests.ApiScenario{
		Name:   "Ingest Qemu",
		Method: "POST",
		URL:    "/api/proxmox/qemu",
		Body:   strings.NewReader(string(qemuBody)),
		Headers: map[string]string{
			"Authorization": token,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{"QEMU VM ingested successfully"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify Qemu DB
	qemuRec, err := app.FindFirstRecordByFilter("proxmox_qemu", "vmid = 101")
	if err != nil {
		t.Fatalf("Qemu not found in DB: %v", err)
	}
	if qemuRec.GetString("node_id") != nodeRec.Id {
		t.Errorf("Expected Qemu to be linked to node %s, got %s", nodeRec.Id, qemuRec.GetString("node_id"))
	}
}

func TestProxmoxPaginationAndFiltering(t *testing.T) {
	app := setupProxmoxTestApp(t)
	defer app.Cleanup()

	agentToken, _ := createAgent(t, app)
	userToken := createUserToken(t, app)

	// Create a node
	nodeReq := types.ProxmoxNode{
		Name:       "pve-01",
		NodeID:     1,
		IP:         "192.168.1.10",
		PVEVersion: "8.0.4",
	}
	nodeBody, _ := json.Marshal(nodeReq)
	(&tests.ApiScenario{
		Method: "POST", URL: "/api/proxmox/node", Body: strings.NewReader(string(nodeBody)),
		Headers: map[string]string{"Authorization": agentToken, "Content-Type": "application/json"}, ExpectedStatus: 200, ExpectedContent: []string{`"message":"Node ingested successfully"`}, TestAppFactory: func(t testing.TB) *tests.TestApp { return app }, DisableTestAppCleanup: true,
	}).Test(t)

	// Create 25 LXCs
	for i := 1; i <= 25; i++ {
		lxcReq := types.ProxmoxLXC{
			Node:   "pve-01",
			VMID:   100 + i,
			Name:   "test-lxc-" + strings.Repeat("x", i), // unique names
			Status: "running",
			OSType: "debian",
			LXCID:  "lxc-" + strings.Repeat("x", i),
		}
		lxcBody, _ := json.Marshal(lxcReq)
		(&tests.ApiScenario{
			Method: "POST", URL: "/api/proxmox/lxc", Body: strings.NewReader(string(lxcBody)),
			Headers: map[string]string{"Authorization": agentToken, "Content-Type": "application/json"}, ExpectedStatus: 200, ExpectedContent: []string{`"message":"LXC ingested successfully"`}, TestAppFactory: func(t testing.TB) *tests.TestApp { return app }, DisableTestAppCleanup: true,
		}).Test(t)
	}

	// Test Pagination: Page 1 (default 20)
	(&tests.ApiScenario{
		Name:   "List LXC Page 1",
		Method: "GET",
		URL:    "/api/proxmox/lxc?page=1&perPage=20",
		Headers: map[string]string{
			"Authorization": userToken,
		},
		ExpectedStatus: 200,
		ExpectedEvents: map[string]int{"OnRecordsListRequest": 0}, // Custom handler doesn't trigger standard hooks usually unless manually triggered, but we check content
		ExpectedContent: []string{
			`"page":1`,
			`"perPage":20`,
			`"totalItems":25`,
			`"totalPages":2`,
			`"items":[`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp { return app }, DisableTestAppCleanup: true,
	}).Test(t)

	// Test Pagination: Page 2
	(&tests.ApiScenario{
		Name:   "List LXC Page 2",
		Method: "GET",
		URL:    "/api/proxmox/lxc?page=2&perPage=20",
		Headers: map[string]string{
			"Authorization": userToken,
		},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"page":2`,
			`"items":[`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp { return app }, DisableTestAppCleanup: true,
	}).Test(t)

	// Test Filtering by Node
	nodeRec, _ := app.FindFirstRecordByFilter("proxmox_nodes", "name = 'pve-01'")
	(&tests.ApiScenario{
		Name:   "List LXC Filtered by Node",
		Method: "GET",
		URL:    "/api/proxmox/lxc?node_id=" + nodeRec.Id,
		Headers: map[string]string{
			"Authorization": userToken,
		},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"totalItems":25`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp { return app }, DisableTestAppCleanup: true,
	}).Test(t)
}

func TestProxmoxDelete(t *testing.T) {
	app := setupProxmoxTestApp(t)
	defer app.Cleanup()

	agentToken, _ := createAgent(t, app)
	userToken := createUserToken(t, app)

	// Create a cluster
	clusterReq := types.ProxmoxCluster{ClusterName: "del-cluster", ClusterID: "del-1", Nodes: 1, Quorate: 1, Version: 1}
	clusterBody, _ := json.Marshal(clusterReq)
	(&tests.ApiScenario{
		Method: "POST", URL: "/api/proxmox/cluster", Body: strings.NewReader(string(clusterBody)),
		Headers: map[string]string{"Authorization": agentToken, "Content-Type": "application/json"}, ExpectedStatus: 200, ExpectedContent: []string{`"message":"Cluster ingested successfully"`}, TestAppFactory: func(t testing.TB) *tests.TestApp { return app }, DisableTestAppCleanup: true,
	}).Test(t)

	clusterRec, _ := app.FindFirstRecordByFilter("proxmox_cluster", "px_cluster_id = 'del-1'")

	// Delete Cluster
	(&tests.ApiScenario{
		Name:   "Delete Cluster",
		Method: "DELETE",
		URL:    "/api/proxmox/clusters/" + clusterRec.Id,
		Headers: map[string]string{
			"Authorization": userToken,
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{"Cluster deleted successfully"},
		TestAppFactory:  func(t testing.TB) *tests.TestApp { return app }, DisableTestAppCleanup: true,
	}).Test(t)

	// Verify deletion
	_, err := app.FindRecordById("proxmox_cluster", clusterRec.Id)
	if err == nil {
		t.Error("Cluster should have been deleted")
	}
}

func TestProxmoxClusterScope(t *testing.T) {
	app := setupProxmoxTestApp(t)
	defer app.Cleanup()

	// Helper to create user/agent pair
	createPair := func(email string) string {
		users, _ := app.FindCollectionByNameOrId("users")
		u := core.NewRecord(users)
		u.Set("email", email)
		u.Set("password", "Pass123!456")
		u.Set("passwordConfirm", "Pass123!456")
		if err := app.Save(u); err != nil {
			t.Fatalf("Failed to user %s: %v", email, err)
		}

		// Create dummy device code
		dcColl, _ := app.FindCollectionByNameOrId("device_codes")
		dcr := core.NewRecord(dcColl)
		dcr.Set("device_code", "code-"+email)
		dcr.Set("user_code", strings.Split(email, "@")[0]+"123") // simple unique
		dcr.Set("expires_at", time.Now().Add(time.Hour))
		if err := app.Save(dcr); err != nil {
			t.Fatalf("Failed to save device code: %v", err)
		}

		agents, _ := app.FindCollectionByNameOrId("agents")
		a := core.NewRecord(agents)
		a.Set("name", "agent-"+email)
		a.Set("user_id", u.Id)
		a.Set("device_code_id", dcr.Id)
		// Minimal fields
		a.Set("hostname", "host-"+email)
		a.Set("platform_family", "linux")
		a.Set("version", "1.0")
		a.SetPassword("AgentPass123!")

		if err := app.Save(a); err != nil {
			t.Fatalf("Failed to agent %s: %v", email, err)
		}
		tok, _ := a.NewAuthToken()
		return tok
	}

	token1 := createPair("user1@test.com")
	token2 := createPair("user2@test.com")

	// 1. User 1 ingests "Prod"
	req := types.ProxmoxCluster{
		ClusterName: "Prod",
		ClusterID:   "cluster", // Generic ID
		Nodes:       1,
		Quorate:     1,
		Version:     1,
	}
	body, _ := json.Marshal(req)

	(&tests.ApiScenario{
		Name: "User 1 Ingest", Method: "POST", URL: "/api/proxmox/cluster",
		Body: strings.NewReader(string(body)), Headers: map[string]string{"Authorization": token1, "Content-Type": "application/json"},
		ExpectedStatus: 200, ExpectedContent: []string{"Cluster ingested successfully"}, DisableTestAppCleanup: true, TestAppFactory: func(test testing.TB) *tests.TestApp { return app },
	}).Test(t)

	// 2. User 2 ingests "Prod"
	(&tests.ApiScenario{
		Name: "User 2 Ingest", Method: "POST", URL: "/api/proxmox/cluster",
		Body: strings.NewReader(string(body)), Headers: map[string]string{"Authorization": token2, "Content-Type": "application/json"},
		ExpectedStatus: 200, ExpectedContent: []string{"Cluster ingested successfully"}, DisableTestAppCleanup: true, TestAppFactory: func(test testing.TB) *tests.TestApp { return app },
	}).Test(t)

	// 3. Verify we have 2 distinct clusters
	total, err := app.CountRecords("proxmox_cluster", dbx.HashExp{"cluster_name": "Prod"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Errorf("Expected 2 clusters named 'Prod' (different users), got %d", total)
	}

	// 4. User 1 ingests "Prod" again (should update existing)
	req.Nodes = 2
	body2, _ := json.Marshal(req)
	(&tests.ApiScenario{
		Name: "User 1 Update", Method: "POST", URL: "/api/proxmox/cluster",
		Body: strings.NewReader(string(body2)), Headers: map[string]string{"Authorization": token1, "Content-Type": "application/json"},
		ExpectedStatus: 200, ExpectedContent: []string{"Cluster ingested successfully"}, DisableTestAppCleanup: true, TestAppFactory: func(test testing.TB) *tests.TestApp { return app },
	}).Test(t)

	// Verify User 1's cluster updated
	user1, _ := app.FindFirstRecordByFilter("users", "email='user1@test.com'")
	rec, err := app.FindFirstRecordByFilter("proxmox_cluster", "cluster_name='Prod' && user_id={:uid}", dbx.Params{"uid": user1.Id})
	if err != nil {
		t.Fatal("User 1 cluster not found")
	}
	if rec.GetInt("nodes") != 2 {
		t.Errorf("Expected nodes update to 2, got %d", rec.GetInt("nodes"))
	}

	// Verify total is still 2
	totalAgain, _ := app.CountRecords("proxmox_cluster", dbx.HashExp{"cluster_name": "Prod"})
	if totalAgain != 2 {
		t.Errorf("Expected 2 clusters total after update, got %d", totalAgain)
	}
}
