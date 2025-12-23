package tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/hooks"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestAgentInvestigationAuth(t *testing.T) {
	// Set required env vars to avoid panics in handlers
	t.Setenv("CLICKHOUSE_URL", "http://localhost:8123")
	t.Setenv("CLICKHOUSE_PASSWORD", "password")
	t.Setenv("TENSORZERO_API_URL", "http://localhost:3000")
	t.Setenv("TENSORZERO_API_KEY", "dummy")
	t.Setenv("JWT_SECRET", "test-secret")

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Run migrations
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register hooks
	hooks.RegisterAgentHooks(app)
	hooks.RegisterInvestigationHooks(app)

	// 1. Create User
	usersCollection, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(usersCollection)
	email := fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())
	user.Set("email", email)
	user.Set("password", "TestPass123!")
	if err := app.Save(user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 2. Create Agent
	agentsCollection, _ := app.FindCollectionByNameOrId("agents")

	// Create device code first (required by schema)
	deviceCodesCollection, _ := app.FindCollectionByNameOrId("device_codes")
	deviceCode := core.NewRecord(deviceCodesCollection)
	deviceCode.Set("device_code", "dev-123")
	deviceCode.Set("user_code", "USR123")
	deviceCode.Set("expires_at", time.Now().Add(time.Hour))
	app.Save(deviceCode)

	agent := core.NewRecord(agentsCollection)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", deviceCode.Id)
	agent.Set("hostname", "test-agent")
	agent.Set("os_type", "linux")
	agent.Set("platform_family", "debian")
	agent.Set("version", "1.0.0")
	agent.SetPassword("AgentPass123!") // Important: Set password for Auth collection
	if err := app.Save(agent); err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 3. Generate Auth Token for Agent
	token, err := agent.NewAuthToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// 4. Create an investigation record
	investigationsCollection, _ := app.FindCollectionByNameOrId("investigations")
	inv := core.NewRecord(investigationsCollection)
	inv.Set("user_id", user.Id)
	inv.Set("agent_id", agent.Id)
	inv.Set("user_prompt", "test issue")
	inv.Set("status", "pending")
	inv.Set("initiated_at", time.Now())
	if err := app.Save(inv); err != nil {
		t.Fatalf("Failed to create investigation: %v", err)
	}

	// 5. Test /api/investigations endpoint with Agent Token
	reqBody := map[string]interface{}{
		"investigation_id": inv.Id,
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Use ApiScenario to test the endpoint
	(&tests.ApiScenario{
		Name:   "Agent proxy request to /api/investigations",
		Method: "POST",
		URL:    "/api/investigations",
		Body:   strings.NewReader(string(jsonBody)),
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  400, // We expect 400 because TensorZero is not reachable
		ExpectedContent: []string{"TensorZero error"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
	}).Test(t)
}
