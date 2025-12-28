package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/hooks"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestProxmoxClusterMultiAgent(t *testing.T) {
	testApp, _ := tests.NewTestApp()
	defer testApp.Cleanup()

	// Run migrations
	if err := testApp.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register hooks
	hooks.RegisterProxmoxHooks(testApp)

	// 1. Setup Data
	usersCollection, err := testApp.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Failed to find users collection: %v", err)
	}
	user := core.NewRecord(usersCollection)
	user.Set("email", "proxmox-test@example.com")
	user.Set("password", "Test123456!@#")
	user.SetVerified(true)
	if err := testApp.Save(user); err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	// Create Device Code (reusable for simplicity in test setup)
	deviceCodesCollection, err := testApp.FindCollectionByNameOrId("device_codes")
	if err != nil {
		t.Fatalf("Failed to find device_codes collection: %v", err)
	}
	deviceCode := core.NewRecord(deviceCodesCollection)
	deviceCode.Set("device_code", "dev-px-123")
	deviceCode.Set("user_code", "USRPX1")
	deviceCode.Set("expires_at", time.Now().Add(time.Hour))
	if err := testApp.Save(deviceCode); err != nil {
		t.Fatalf("Failed to save device code: %v", err)
	}

	// Create Agent A
	agentsCollection, err := testApp.FindCollectionByNameOrId("agents")
	if err != nil {
		t.Fatalf("Failed to find agents collection: %v", err)
	}

	agentA := core.NewRecord(agentsCollection)
	agentA.Set("user_id", user.Id)
	agentA.Set("device_code_id", deviceCode.Id)
	agentA.Set("hostname", "node-a")
	agentA.Set("platform_family", "debian")
	agentA.Set("version", "1.0.0")
	agentA.SetPassword("AgentPassA123!")
	if err := testApp.Save(agentA); err != nil {
		t.Fatalf("Failed to save agent A: %v", err)
	}
	tokenA, _ := agentA.NewAuthToken()

	// Create Agent B
	agentB := core.NewRecord(agentsCollection)
	agentB.Set("user_id", user.Id)
	agentB.Set("device_code_id", deviceCode.Id)
	agentB.Set("hostname", "node-b")
	agentB.Set("platform_family", "debian")
	agentB.Set("version", "1.0.0")
	agentB.SetPassword("AgentPassB123!")
	if err := testApp.Save(agentB); err != nil {
		t.Fatalf("Failed to save agent B: %v", err)
	}
	tokenB, _ := agentB.NewAuthToken()

	// 2. Agent A ingests Cluster
	clusterPayload := `{"name": "test-cluster", "nodes": 2, "quorate": 1, "version": 8, "px_cluster_id": "cluster-uuid-123"}`

	(&tests.ApiScenario{
		Name:   "Agent A ingests Cluster",
		Method: "POST",
		URL:    "/api/proxmox/cluster",
		Body:   strings.NewReader(clusterPayload),
		Headers: map[string]string{
			"Authorization": "Bearer " + tokenA,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{`"message":"Cluster ingested successfully"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify Agent A is in cluster
	clusterRec, err := testApp.FindFirstRecordByFilter("proxmox_cluster", "px_cluster_id = 'cluster-uuid-123'")
	if err != nil {
		t.Fatalf("Failed to find cluster: %v", err)
	}
	agents := clusterRec.GetStringSlice("agents")
	if len(agents) != 1 || agents[0] != agentA.Id {
		t.Errorf("Expected agents [A], got %v", agents)
	}

	// 3. Agent B ingests SAME Cluster
	(&tests.ApiScenario{
		Name:   "Agent B ingests Cluster",
		Method: "POST",
		URL:    "/api/proxmox/cluster",
		Body:   strings.NewReader(clusterPayload),
		Headers: map[string]string{
			"Authorization": "Bearer " + tokenB,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{`"message":"Cluster ingested successfully"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify Agent B is ALSO in cluster
	clusterRec, err = testApp.FindRecordById("proxmox_cluster", clusterRec.Id)
	if err != nil {
		t.Fatalf("Failed to reload cluster: %v", err)
	}
	agents = clusterRec.GetStringSlice("agents")
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d: %v", len(agents), agents)
	}

	// 4. Agent A ingests Node A
	nodeAPayload := `{"name": "node-a", "ip": "192.168.1.10", "pveversion": "8.0", "online": 1}`
	(&tests.ApiScenario{
		Name:   "Agent A ingests Node A",
		Method: "POST",
		URL:    "/api/proxmox/node",
		Body:   strings.NewReader(nodeAPayload),
		Headers: map[string]string{
			"Authorization": "Bearer " + tokenA,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{`"message":"Node ingested successfully"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify Node A has cluster_id
	nodeA, err := testApp.FindFirstRecordByFilter("proxmox_nodes", "name = 'node-a'")
	if err != nil {
		t.Fatalf("Failed to find node A: %v", err)
	}
	if nodeA.GetString("cluster_id") != clusterRec.Id {
		t.Errorf("Node A cluster_id mismatch. Expected %s, got %s", clusterRec.Id, nodeA.GetString("cluster_id"))
	}

	// 5. Agent B ingests Node B
	nodeBPayload := `{"name": "node-b", "ip": "192.168.1.11", "pveversion": "8.0", "online": 1}`
	(&tests.ApiScenario{
		Name:   "Agent B ingests Node B",
		Method: "POST",
		URL:    "/api/proxmox/node",
		Body:   strings.NewReader(nodeBPayload),
		Headers: map[string]string{
			"Authorization": "Bearer " + tokenB,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{`"message":"Node ingested successfully"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify Node B has cluster_id (found via Agent B)
	nodeB, err := testApp.FindFirstRecordByFilter("proxmox_nodes", "name = 'node-b'")
	if err != nil {
		t.Fatalf("Failed to find node B: %v", err)
	}
	if nodeB.GetString("cluster_id") != clusterRec.Id {
		t.Errorf("Node B cluster_id mismatch. Expected %s, got %s", clusterRec.Id, nodeB.GetString("cluster_id"))
	}

	// Create Agent C
	agentC := core.NewRecord(agentsCollection)
	agentC.Set("user_id", user.Id)
	agentC.Set("device_code_id", deviceCode.Id)
	agentC.Set("hostname", "node-c")
	agentC.Set("platform_family", "debian")
	agentC.Set("version", "1.0.0")
	agentC.SetPassword("AgentPassC123!")
	if err := testApp.Save(agentC); err != nil {
		t.Fatalf("Failed to save agent C: %v", err)
	}
	tokenC, _ := agentC.NewAuthToken()

	// 6. Agent C ingests Node C (before Cluster C exists)
	nodeCPayload := `{"name": "node-c", "ip": "192.168.1.12", "pveversion": "8.0", "online": 1}`
	(&tests.ApiScenario{
		Name:   "Agent C ingests Node C (Pre-Cluster)",
		Method: "POST",
		URL:    "/api/proxmox/node",
		Body:   strings.NewReader(nodeCPayload),
		Headers: map[string]string{
			"Authorization": "Bearer " + tokenC,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{`"message":"Node ingested successfully"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify Node C has NO cluster_id
	nodeC, err := testApp.FindFirstRecordByFilter("proxmox_nodes", "name = 'node-c'")
	if err != nil {
		t.Fatalf("Failed to find node C: %v", err)
	}
	if nodeC.GetString("cluster_id") != "" {
		t.Errorf("Node C should not have cluster_id yet, got %s", nodeC.GetString("cluster_id"))
	}

	// 7. Agent C ingests Cluster C
	clusterCPayload := `{"name": "cluster-c", "nodes": 1, "quorate": 1, "version": 8, "px_cluster_id": "cluster-uuid-456"}`
	(&tests.ApiScenario{
		Name:   "Agent C ingests Cluster C",
		Method: "POST",
		URL:    "/api/proxmox/cluster",
		Body:   strings.NewReader(clusterCPayload),
		Headers: map[string]string{
			"Authorization": "Bearer " + tokenC,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  200,
		ExpectedContent: []string{`"message":"Cluster ingested successfully"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return testApp
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	// Verify Node C HAS cluster_id (Backfilled automatically)
	nodeC, err = testApp.FindFirstRecordByFilter("proxmox_nodes", "name = 'node-c'")
	if err != nil {
		t.Fatalf("Failed to find node C: %v", err)
	}
	clusterC, _ := testApp.FindFirstRecordByFilter("proxmox_cluster", "px_cluster_id = 'cluster-uuid-456'")
	if nodeC.GetString("cluster_id") != clusterC.Id {
		t.Errorf("Node C cluster_id mismatch. Expected %s, got %s", clusterC.Id, nodeC.GetString("cluster_id"))
	}
}
