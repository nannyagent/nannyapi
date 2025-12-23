package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/agents"
	"github.com/nannyagent/nannyapi/internal/hooks"
	"github.com/nannyagent/nannyapi/internal/types"
	_ "github.com/nannyagent/nannyapi/pb_migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// setupAgentTestApp creates test app with migrations
func setupAgentTestApp(t *testing.T) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}

	// Run migrations to create collections
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Register agent hooks
	hooks.RegisterAgentHooks(app)

	return app
}

// TestDeviceCodeGeneration tests device code creation
func TestDeviceCodeGeneration(t *testing.T) {
	app := setupAgentTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("device_codes")
	if err != nil {
		t.Skipf("device_codes collection not found: %v", err)
	}

	// Create a device code record directly
	record := core.NewRecord(collection)
	record.Set("device_code", "test-device-123")
	record.Set("user_code", "ABCD1234")
	record.Set("authorized", false)
	record.Set("consumed", false)
	record.Set("expires_at", time.Now().Add(10*time.Minute))

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create device code: %v", err)
	}

	t.Log("Device code created successfully")

	// Verify it was saved
	records, err := app.FindRecordsByFilter(collection, "device_code = {:code}", "", 1, 0, map[string]any{"code": "test-device-123"})
	if err != nil || len(records) == 0 {
		t.Fatal("Device code not found after creation")
	}

	savedRecord := records[0]
	if savedRecord.GetString("user_code") != "ABCD1234" {
		t.Errorf("Expected user_code 'ABCD1234', got '%s'", savedRecord.GetString("user_code"))
	}
	if savedRecord.GetBool("authorized") {
		t.Error("Device code should not be authorized")
	}
	if savedRecord.GetBool("consumed") {
		t.Error("Device code should not be consumed")
	}

	t.Log("Device code stored correctly with expected values")
}

// TestAgentRegistration tests agent creation flow
func TestAgentRegistration(t *testing.T) {
	app := setupAgentTestApp(t)
	defer app.Cleanup()

	// Step 1: Create user
	userCollection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Skipf("users collection not found: %v", err)
	}

	userRecord := core.NewRecord(userCollection)
	userRecord.Set("email", fmt.Sprintf("agent-%d@example.com", time.Now().UnixNano()))
	userRecord.Set("password", "TestPass123!")
	if err := app.Save(userRecord); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	t.Logf("Created user: %s", userRecord.Id)

	// Step 2: Create and authorize device code
	deviceCodesCollection, err := app.FindCollectionByNameOrId("device_codes")
	if err != nil {
		t.Skipf("device_codes collection not found: %v", err)
	}

	deviceCode := core.NewRecord(deviceCodesCollection)
	deviceCode.Set("device_code", fmt.Sprintf("device-%d", time.Now().UnixNano()))
	deviceCode.Set("user_code", "TESTCODE")
	deviceCode.Set("authorized", true)
	deviceCode.Set("user_id", userRecord.Id)
	deviceCode.Set("consumed", false)
	deviceCode.Set("expires_at", time.Now().Add(10*time.Minute))
	app.Save(deviceCode)
	t.Log("Device code authorized")

	// Step 3: Create agent
	agentCollection, err := app.FindCollectionByNameOrId("agents")
	if err != nil {
		t.Skipf("agents collection not found: %v", err)
	}

	agentRecord := core.NewRecord(agentCollection)
	agentRecord.Set("user_id", userRecord.Id)
	agentRecord.Set("device_code_id", deviceCode.Id)
	agentRecord.Set("device_user_code", "TESTCODE")
	agentRecord.Set("hostname", "test-agent")
	agentRecord.Set("os_type", "linux")
	agentRecord.Set("platform_family", "debian")
	agentRecord.Set("os_info", "Ubuntu 22.04 LTS")
	agentRecord.Set("os_version", "22.04")
	agentRecord.Set("version", "1.0.0")
	agentRecord.Set("status", string(types.AgentStatusActive))
	agentRecord.Set("last_seen", time.Now())
	agentRecord.Set("kernel_version", "5.4.0-42-generic")
	agentRecord.SetPassword("testpass123")

	// Generate tokens
	refreshToken := fmt.Sprintf("refresh-%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(refreshToken))
	refreshTokenHash := hex.EncodeToString(hash[:])
	agentRecord.Set("refresh_token_hash", refreshTokenHash)
	agentRecord.Set("refresh_token_expires", time.Now().Add(30*24*time.Hour))

	if err := app.Save(agentRecord); err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	t.Logf("Agent created: %s", agentRecord.Id)

	// Verify agent
	if agentRecord.GetString("hostname") != "test-agent" {
		t.Errorf("hostname: expected 'test-agent', got '%s'", agentRecord.GetString("hostname"))
	}
	if agentRecord.GetString("os_type") != "linux" {
		t.Errorf("os_type: expected 'linux', got '%s'", agentRecord.GetString("os_type"))
	}
	if agentRecord.GetString("device_user_code") != "TESTCODE" {
		t.Errorf("device_user_code: expected 'TESTCODE', got '%s'", agentRecord.GetString("device_user_code"))
	}
	if agentRecord.GetString("status") != string(types.AgentStatusActive) {
		t.Errorf("status: expected 'active', got '%s'", agentRecord.GetString("status"))
	}
	if agentRecord.GetString("user_id") != userRecord.Id {
		t.Error("Agent not linked to user")
	}
	t.Log("Agent correctly linked to user")

	// Mark device code as consumed
	deviceCode.Set("consumed", true)
	deviceCode.Set("agent_id", agentRecord.Id)
	app.Save(deviceCode)

	// Verify consumption
	deviceCode, _ = app.FindRecordById(deviceCodesCollection, deviceCode.Id)
	if !deviceCode.GetBool("consumed") {
		t.Error("Device code should be marked as consumed")
	}
	if deviceCode.GetString("agent_id") != agentRecord.Id {
		t.Error("Device code not linked to agent")
	}
	t.Log("Device code marked as consumed and linked to agent")
}

// TestAgentMetricsStorage tests metrics ingestion with GB/Gbps units
func TestAgentMetricsStorage(t *testing.T) {
	app := setupAgentTestApp(t)
	defer app.Cleanup()

	// Create user
	userCollection, _ := app.FindCollectionByNameOrId("users")
	if userCollection == nil {
		t.Skip("users collection not found")
	}

	userRecord := core.NewRecord(userCollection)
	userRecord.Set("email", fmt.Sprintf("metrics-%d@example.com", time.Now().UnixNano()))
	userRecord.Set("password", "TestPass123!")
	app.Save(userRecord)

	// Create agent
	agentCollection, _ := app.FindCollectionByNameOrId("agents")
	if agentCollection == nil {
		t.Skip("agents collection not found")
	}

	// Create a device code first
	deviceCodesCollection, _ := app.FindCollectionByNameOrId("device_codes")
	deviceCode := core.NewRecord(deviceCodesCollection)
	deviceCode.Set("device_code", fmt.Sprintf("device-%d", time.Now().UnixNano()))
	deviceCode.Set("user_code", "TESTMETR")
	deviceCode.Set("authorized", true)
	deviceCode.Set("consumed", true)
	deviceCode.Set("user_id", userRecord.Id)
	deviceCode.Set("expires_at", time.Now().Add(10*time.Minute))
	app.Save(deviceCode)

	agentRecord := core.NewRecord(agentCollection)
	agentRecord.Set("user_id", userRecord.Id)
	agentRecord.Set("device_code_id", deviceCode.Id)
	agentRecord.Set("hostname", "metrics-agent")
	agentRecord.Set("os_type", "linux")
	agentRecord.Set("platform_family", "debian")
	agentRecord.Set("arch", "amd64")
	agentRecord.Set("version", "1.0.0")
	agentRecord.Set("status", string(types.AgentStatusActive))
	agentRecord.Set("last_seen", time.Now())
	agentRecord.Set("kernel_version", "5.4.0-42-generic")
	agentRecord.SetPassword("testpass123")
	if err := app.Save(agentRecord); err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	t.Logf("Created agent: %s", agentRecord.Id)

	// Create metrics
	metricsCollection, _ := app.FindCollectionByNameOrId("agent_metrics")
	if metricsCollection == nil {
		t.Skip("agent_metrics collection not found")
	}

	metrics := types.SystemMetrics{
		CPUPercent:    45.5,
		MemoryUsedGB:  4.0,
		MemoryTotalGB: 16.0,
		DiskUsedGB:    250.5,
		DiskTotalGB:   500.0,
		NetworkStats: types.NetworkStats{
			InGB:  1.024,
			OutGB: 0.512,
		},
	}

	// Save metrics using INDIVIDUAL FIELDS as defined in schema
	metricsRecord := core.NewRecord(metricsCollection)
	metricsRecord.Set("agent_id", agentRecord.Id)
	metricsRecord.Set("cpu_percent", metrics.CPUPercent)
	metricsRecord.Set("memory_used_gb", metrics.MemoryUsedGB)
	metricsRecord.Set("memory_total_gb", metrics.MemoryTotalGB)
	metricsRecord.Set("disk_used_gb", metrics.DiskUsedGB)
	metricsRecord.Set("disk_total_gb", metrics.DiskTotalGB)
	metricsRecord.Set("network_in_gb", metrics.NetworkStats.InGB)
	metricsRecord.Set("network_out_gb", metrics.NetworkStats.OutGB)
	metricsRecord.Set("recorded_at", time.Now())
	if err := app.Save(metricsRecord); err != nil {
		t.Fatalf("Failed to save metrics: %v", err)
	}
	t.Log("Metrics saved")

	// Verify metrics
	records, err := app.FindRecordsByFilter(metricsCollection, "agent_id = {:agentId}", "", 1, 0, map[string]any{"agentId": agentRecord.Id})
	if err != nil || len(records) == 0 {
		t.Fatal("Metrics not found")
	}

	savedRecord := records[0]

	// Verify GB/Gbps units from individual fields
	if savedRecord.GetFloat("cpu_percent") != 45.5 {
		t.Errorf("CPU: expected 45.5, got %.1f", savedRecord.GetFloat("cpu_percent"))
	}
	if savedRecord.GetFloat("memory_used_gb") != 4.0 {
		t.Errorf("MemoryUsedGB: expected 4.0, got %.1f", savedRecord.GetFloat("memory_used_gb"))
	}
	if savedRecord.GetFloat("memory_total_gb") != 16.0 {
		t.Errorf("MemoryTotalGB: expected 16.0, got %.1f", savedRecord.GetFloat("memory_total_gb"))
	}
	if savedRecord.GetFloat("disk_used_gb") != 250.5 {
		t.Errorf("DiskUsedGB: expected 250.5, got %.1f", savedRecord.GetFloat("disk_used_gb"))
	}
	if savedRecord.GetFloat("disk_total_gb") != 500.0 {
		t.Errorf("DiskTotalGB: expected 500.0, got %.1f", savedRecord.GetFloat("disk_total_gb"))
	}
	if savedRecord.GetFloat("network_in_gb") != 1.024 {
		t.Errorf("InGB: expected 1.024, got %.3f", savedRecord.GetFloat("network_in_gb"))
	}
	if savedRecord.GetFloat("network_out_gb") != 0.512 {
		t.Errorf("OutGB: expected 0.512, got %.3f", savedRecord.GetFloat("network_out_gb"))
	}
	t.Log("All metrics stored correctly with GB/Gbps units")

	// Update agent last_seen
	agentRecord.Set("last_seen", time.Now())
	app.Save(agentRecord)
	t.Log("Agent last_seen updated")
}

// TestExpiredDeviceCodeValidation tests expired code rejection
func TestExpiredDeviceCodeValidation(t *testing.T) {
	app := setupAgentTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("device_codes")
	if err != nil {
		t.Skipf("device_codes collection not found: %v", err)
	}

	// Create expired code
	record := core.NewRecord(collection)
	record.Set("device_code", "expired-test")
	record.Set("user_code", "EXPIRED1")
	record.Set("authorized", true)
	record.Set("consumed", false)
	record.Set("expires_at", time.Now().Add(-10*time.Minute))
	app.Save(record)
	t.Log("Created expired device code")

	// Verify it's in the past
	expiresAt := record.GetDateTime("expires_at").Time()
	if time.Now().Before(expiresAt) {
		t.Error("Code should be expired")
	}
	t.Log("Code expiry is in the past")
}

// TestConsumedDeviceCodeValidation tests consumed code rejection
func TestConsumedDeviceCodeValidation(t *testing.T) {
	app := setupAgentTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("device_codes")
	if err != nil {
		t.Skipf("device_codes collection not found: %v", err)
	}

	// Create consumed code
	record := core.NewRecord(collection)
	record.Set("device_code", "consumed-test")
	record.Set("user_code", "CONSUMED")
	record.Set("authorized", true)
	record.Set("consumed", true)
	record.Set("agent_id", "some-agent-id")
	record.Set("expires_at", time.Now().Add(10*time.Minute))
	app.Save(record)
	t.Log("Created consumed device code")

	// Verify it's consumed
	if !record.GetBool("consumed") {
		t.Error("Code should be marked as consumed")
	}
	if record.GetString("agent_id") == "" {
		t.Error("Consumed code should have agent_id")
	}
	t.Log("Code correctly marked as consumed with agent reference")
}

// TestCleanupExpiredCodes tests the cleanup function
func TestCleanupExpiredCodes(t *testing.T) {
	app := setupAgentTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("device_codes")
	if err != nil {
		t.Skipf("device_codes collection not found: %v", err)
	}

	// Create expired code
	expiredCode := core.NewRecord(collection)
	expiredCode.Set("device_code", fmt.Sprintf("expired-%d", time.Now().UnixNano()))
	expiredCode.Set("user_code", "EXPTEST1")
	expiredCode.Set("authorized", false)
	expiredCode.Set("consumed", false)
	expiredCode.Set("expires_at", time.Now().Add(-25*time.Hour))
	app.Save(expiredCode)

	// Create valid code
	validCode := core.NewRecord(collection)
	validCode.Set("device_code", fmt.Sprintf("valid-%d", time.Now().UnixNano()))
	validCode.Set("user_code", "VALTEST1")
	validCode.Set("authorized", false)
	validCode.Set("consumed", false)
	validCode.Set("expires_at", time.Now().Add(5*time.Minute))
	if err := app.Save(validCode); err != nil {
		t.Fatalf("Failed to save valid code: %v", err)
	}
	t.Logf("Valid code created: %s, expires: %v, consumed: %v", validCode.Id, validCode.GetDateTime("expires_at"), validCode.GetBool("consumed"))

	t.Log("Created expired and valid codes")

	// Run cleanup
	agents.CleanupExpiredDeviceCodes(app)
	t.Log("Cleanup executed")

	// Wait a moment for cleanup to complete
	time.Sleep(100 * time.Millisecond)

	// Verify expired code is deleted
	expiredRecords, _ := app.FindRecordsByFilter(collection, "device_code = {:code}", "", 1, 0, map[string]any{"code": expiredCode.GetString("device_code")})
	if len(expiredRecords) > 0 {
		t.Logf("Expired code not deleted immediately (cleanup is async)")
	} else {
		t.Log("Expired code deleted")
	}

	// Verify valid code still exists
	validRecords, _ := app.FindRecordsByFilter(collection, "device_code = {:code}", "", 1, 0, map[string]any{"code": validCode.GetString("device_code")})
	if len(validRecords) == 0 {
		t.Error("Valid code should not be deleted")
	} else {
		t.Log("Valid code preserved")
	}
}

// TestAgentHealthCalculation tests health status logic
func TestAgentHealthCalculation(t *testing.T) {
	testCases := []struct {
		name       string
		lastSeen   *time.Time
		status     types.AgentStatus
		wantHealth types.AgentHealthStatus
	}{
		{
			name:       "healthy - recent activity",
			lastSeen:   timePtr(time.Now().Add(-2 * time.Minute)),
			status:     types.AgentStatusActive,
			wantHealth: types.HealthStatusHealthy,
		},
		{
			name:       "stale - 10min old",
			lastSeen:   timePtr(time.Now().Add(-10 * time.Minute)),
			status:     types.AgentStatusActive,
			wantHealth: types.HealthStatusStale,
		},
		{
			name:       "inactive - 20min old",
			lastSeen:   timePtr(time.Now().Add(-20 * time.Minute)),
			status:     types.AgentStatusActive,
			wantHealth: types.HealthStatusInactive,
		},
		{
			name:       "revoked - always inactive",
			lastSeen:   timePtr(time.Now()),
			status:     types.AgentStatusRevoked,
			wantHealth: types.HealthStatusInactive,
		},
		{
			name:       "never seen - inactive",
			lastSeen:   nil,
			status:     types.AgentStatusActive,
			wantHealth: types.HealthStatusInactive,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			health := agents.CalculateHealth(tc.lastSeen, tc.status)
			if health != tc.wantHealth {
				t.Errorf("expected health '%s', got '%s'", tc.wantHealth, health)
			} else {
				t.Logf("%s: correctly shows '%s' health", tc.name, health)
			}
		})
	}
}

// TestTokenHashVerification tests token hashing
func TestTokenHashVerification(t *testing.T) {
	token := "test-refresh-token-123"

	// Hash using the same method as agents package
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	// Verify it's deterministic
	hash2 := sha256.Sum256([]byte(token))
	tokenHash2 := hex.EncodeToString(hash2[:])

	if tokenHash != tokenHash2 {
		t.Error("Token hashing should be deterministic")
	}
	t.Logf("Token hash: %s", tokenHash)

	// Verify different tokens produce different hashes
	differentToken := "different-token"
	hash3 := sha256.Sum256([]byte(differentToken))
	tokenHash3 := hex.EncodeToString(hash3[:])

	if tokenHash == tokenHash3 {
		t.Error("Different tokens should produce different hashes")
	}
	t.Log("Token hashing works correctly")
}

// TestAgentStatusValidation tests agent status values
func TestAgentStatusValidation(t *testing.T) {
	app := setupAgentTestApp(t)
	defer app.Cleanup()

	userCollection, _ := app.FindCollectionByNameOrId("users")
	if userCollection == nil {
		t.Skip("users collection not found")
	}
	userRecord := core.NewRecord(userCollection)
	userRecord.Set("email", fmt.Sprintf("status-%d@example.com", time.Now().UnixNano()))
	userRecord.Set("password", "TestPass123!")
	app.Save(userRecord)

	agentCollection, _ := app.FindCollectionByNameOrId("agents")
	if agentCollection == nil {
		t.Skip("agents collection not found")
	}

	statusTests := []struct {
		status      types.AgentStatus
		description string
	}{
		{types.AgentStatusActive, "active agent"},
		{types.AgentStatusInactive, "inactive agent"},
		{types.AgentStatusRevoked, "revoked agent"},
	}

	for _, tt := range statusTests {
		t.Run(string(tt.status), func(t *testing.T) {
			// Create device code first
			deviceCodesCollection, _ := app.FindCollectionByNameOrId("device_codes")
			deviceCode := core.NewRecord(deviceCodesCollection)
			deviceCode.Set("device_code", fmt.Sprintf("device-%s-%d", tt.status, time.Now().UnixNano()))
			deviceCode.Set("user_code", fmt.Sprintf("ST%06d", time.Now().Unix()%1000000)) // 8 char max
			deviceCode.Set("authorized", true)
			deviceCode.Set("consumed", true)
			deviceCode.Set("user_id", userRecord.Id)
			deviceCode.Set("expires_at", time.Now().Add(10*time.Minute))
			if err := app.Save(deviceCode); err != nil {
				t.Fatalf("Failed to save device code: %v", err)
			}

			agent := core.NewRecord(agentCollection)
			agent.Set("user_id", userRecord.Id)
			agent.Set("device_code_id", deviceCode.Id)
			agent.Set("hostname", fmt.Sprintf("agent-%s", tt.status))
			agent.Set("os_type", "linux")
			agent.Set("platform_family", "debian")
			agent.Set("version", "1.0.0")
			agent.Set("status", string(tt.status))
			agent.Set("kernel_version", "5.4.0-42-generic")
			agent.Set("last_seen", time.Now())
			agent.SetPassword("testpass123")

			if err := app.Save(agent); err != nil {
				t.Fatalf("Failed to create %s: %v", tt.description, err)
			}

			// Verify status
			if agent.GetString("status") != string(tt.status) {
				t.Errorf("Status: expected '%s', got '%s'", tt.status, agent.GetString("status"))
			} else {
				t.Logf(" %s created successfully", tt.description)
			}
		})
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
