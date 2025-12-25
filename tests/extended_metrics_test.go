package tests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// TestExtendedMetricsFields tests that extended metrics fields are properly stored and retrieved
func TestExtendedMetricsFields(t *testing.T) {
	app, _ := tests.NewTestApp()
	defer app.Cleanup()

	// Create user
	usersCollection, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(usersCollection)
	user.Set("email", "extended-metrics-user@test.local")
	user.Set("password", "test-password-123")
	if err := app.Save(user); err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	// Create device code (required by agents)
	deviceCodesCollection, _ := app.FindCollectionByNameOrId("device_codes")
	deviceCode := core.NewRecord(deviceCodesCollection)
	deviceCode.Set("device_code", "123456")
	deviceCode.Set("user_code", "ABC-DEF")
	deviceCode.Set("user_id", user.Id)
	deviceCode.Set("status", "pending")
	deviceCode.Set("expires_at", time.Now().Add(15*time.Minute))
	if err := app.Save(deviceCode); err != nil {
		t.Fatalf("Failed to save device code: %v", err)
	}

	// Create agent
	agentsCollection, _ := app.FindCollectionByNameOrId("agents")
	agent := core.NewRecord(agentsCollection)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", deviceCode.Id)
	agent.Set("hostname", "test-host-extended")
	agent.Set("os_type", "linux")
	agent.Set("platform_family", "debian")
	agent.Set("version", "1.0.0")
	agent.Set("status", "active")
	agent.SetPassword("testpass123")
	if err := app.Save(agent); err != nil {
		t.Fatalf("Failed to save agent: %v", err)
	}

	// Create metrics record
	metricsCollection, _ := app.FindCollectionByNameOrId("agent_metrics")
	metrics := core.NewRecord(metricsCollection)
	metrics.Set("agent_id", agent.Id)
	metrics.Set("cpu_percent", 45.5)
	metrics.Set("cpu_cores", 8)
	metrics.Set("memory_used_gb", 8.2)
	metrics.Set("memory_total_gb", 16.0)
	metrics.Set("memory_percent", 51.25)
	metrics.Set("disk_used_gb", 250.5)
	metrics.Set("disk_total_gb", 512.0)
	metrics.Set("disk_usage_percent", 48.93)
	metrics.Set("network_in_gbps", 0.5)
	metrics.Set("network_out_gbps", 0.3)
	metrics.Set("load_avg_1min", 2.45)
	metrics.Set("load_avg_5min", 2.10)
	metrics.Set("load_avg_15min", 1.95)

	// Filesystems as JSON
	filesystems := []map[string]interface{}{
		{
			"device":        "/dev/sda1",
			"mount_path":    "/",
			"used_gb":       250.5,
			"free_gb":       261.5,
			"total_gb":      512.0,
			"usage_percent": 48.93,
		},
		{
			"device":        "/dev/sda2",
			"mount_path":    "/home",
			"used_gb":       100.0,
			"free_gb":       200.0,
			"total_gb":      300.0,
			"usage_percent": 33.33,
		},
	}
	metricsJSON, _ := json.Marshal(filesystems)
	metrics.Set("filesystems", string(metricsJSON))
	metrics.Set("recorded_at", time.Now())

	if err := app.Save(metrics); err != nil {
		t.Fatalf("Failed to save metrics: %v", err)
	}

	// Verify all fields are stored correctly
	retrieved, err := app.FindRecordById(metricsCollection, metrics.Id)
	if err != nil {
		t.Fatalf("Failed to retrieve metrics: %v", err)
	}

	if cpuCores := retrieved.GetInt("cpu_cores"); cpuCores != 8 {
		t.Errorf("Expected cpu_cores=8, got %d", cpuCores)
	}

	if memPercent := retrieved.GetFloat("memory_percent"); memPercent < 51 || memPercent > 52 {
		t.Errorf("Expected memory_percent~51.25, got %f", memPercent)
	}

	if diskPercent := retrieved.GetFloat("disk_usage_percent"); diskPercent < 48 || diskPercent > 49 {
		t.Errorf("Expected disk_usage_percent~48.93, got %f", diskPercent)
	}

	if load1 := retrieved.GetFloat("load_avg_1min"); load1 != 2.45 {
		t.Errorf("Expected load_avg_1min=2.45, got %f", load1)
	}

	if load5 := retrieved.GetFloat("load_avg_5min"); load5 != 2.10 {
		t.Errorf("Expected load_avg_5min=2.10, got %f", load5)
	}

	if load15 := retrieved.GetFloat("load_avg_15min"); load15 != 1.95 {
		t.Errorf("Expected load_avg_15min=1.95, got %f", load15)
	}

	filesystemsData := retrieved.GetString("filesystems")
	if filesystemsData == "" {
		t.Error("Expected filesystems to be set")
	}

	if !strings.Contains(filesystemsData, "/dev/sda1") {
		t.Errorf("Expected filesystems to contain /dev/sda1, got %s", filesystemsData)
	}
}

// TestIngestMetricsWithExtendedData tests ingesting metrics via API with extended fields
func TestIngestMetricsWithExtendedData(t *testing.T) {
	app, _ := tests.NewTestApp()
	defer app.Cleanup()

	// Create user
	usersCollection, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(usersCollection)
	user.Set("email", "ingest-metrics-user@test.local")
	user.Set("password", "test-password-123")
	if err := app.Save(user); err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	// Create device code (required by agents) - WITH expires_at
	deviceCodesCollection, _ := app.FindCollectionByNameOrId("device_codes")
	deviceCode := core.NewRecord(deviceCodesCollection)
	deviceCode.Set("device_code", "654321")
	deviceCode.Set("user_id", user.Id)
	deviceCode.Set("user_code", "XYZ-UVW")
	deviceCode.Set("expires_at", time.Now().Add(15*time.Minute))
	if err := app.Save(deviceCode); err != nil {
		t.Fatalf("Failed to save device code: %v", err)
	}

	// Create agent
	agentsCollection, _ := app.FindCollectionByNameOrId("agents")
	agent := core.NewRecord(agentsCollection)
	agent.Set("user_id", user.Id)
	agent.Set("device_code_id", deviceCode.Id)
	agent.Set("hostname", "ingest-test-host")
	agent.Set("os_type", "linux")
	agent.Set("platform_family", "debian")
	agent.Set("version", "1.0.0")
	agent.Set("status", "active")
	agent.SetPassword("testpass123")
	if err := app.Save(agent); err != nil {
		t.Fatalf("Failed to save agent: %v", err)
	}

	// Test payload with extended metrics
	metrics := types.SystemMetrics{
		CPUPercent:       67.8,
		CPUCores:         16,
		MemoryUsedGB:     12.5,
		MemoryTotalGB:    32.0,
		MemoryPercent:    (12.5 / 32.0) * 100,
		DiskUsedGB:       450.0,
		DiskTotalGB:      1000.0,
		DiskUsagePercent: (450.0 / 1000.0) * 100,
		NetworkStats: types.NetworkStats{
			InGB:  1.2,
			OutGB: 0.8,
		},
		LoadAverage: types.LoadAverage{
			OneMin:     4.50,
			FiveMin:    3.75,
			FifteenMin: 3.20,
		},
		Filesystems: []types.FilesystemStats{
			{
				Device:       "/dev/nvme0n1p1",
				MountPath:    "/",
				UsedGB:       450.0,
				FreeGB:       550.0,
				TotalGB:      1000.0,
				UsagePercent: 45.0,
			},
			{
				Device:       "/dev/nvme0n1p2",
				MountPath:    "/data",
				UsedGB:       100.0,
				FreeGB:       150.0,
				TotalGB:      250.0,
				UsagePercent: 40.0,
			},
		},
	}

	metricsCollection, _ := app.FindCollectionByNameOrId("agent_metrics")
	metricsRecord := core.NewRecord(metricsCollection)
	metricsRecord.Set("agent_id", agent.Id)

	// Set fields directly from struct (simulating new handler)
	metricsRecord.Set("cpu_percent", metrics.CPUPercent)
	metricsRecord.Set("cpu_cores", metrics.CPUCores)
	metricsRecord.Set("memory_used_gb", metrics.MemoryUsedGB)
	metricsRecord.Set("memory_total_gb", metrics.MemoryTotalGB)
	metricsRecord.Set("memory_percent", metrics.MemoryPercent)
	metricsRecord.Set("disk_used_gb", metrics.DiskUsedGB)
	metricsRecord.Set("disk_total_gb", metrics.DiskTotalGB)
	metricsRecord.Set("disk_usage_percent", metrics.DiskUsagePercent)
	metricsRecord.Set("network_in_gb", metrics.NetworkStats.InGB)
	metricsRecord.Set("network_out_gb", metrics.NetworkStats.OutGB)
	metricsRecord.Set("load_avg_1min", metrics.LoadAverage.OneMin)
	metricsRecord.Set("load_avg_5min", metrics.LoadAverage.FiveMin)
	metricsRecord.Set("load_avg_15min", metrics.LoadAverage.FifteenMin)

	if len(metrics.Filesystems) > 0 {
		filesystemsJSON, _ := json.Marshal(metrics.Filesystems)
		metricsRecord.Set("filesystems", string(filesystemsJSON))
	}

	metricsRecord.Set("recorded_at", time.Now())

	if err := app.Save(metricsRecord); err != nil {
		t.Fatalf("Failed to save ingested metrics: %v", err)
	}

	// Verify all fields are stored correctly
	retrieved, _ := app.FindRecordById(metricsCollection, metricsRecord.Id)

	if cpuPercent := retrieved.GetFloat("cpu_percent"); cpuPercent != 67.8 {
		t.Errorf("Expected cpu_percent=67.8, got %f", cpuPercent)
	}

	if cpuCores := retrieved.GetInt("cpu_cores"); cpuCores != 16 {
		t.Errorf("Expected cpu_cores=16, got %d", cpuCores)
	}

	if memPercent := retrieved.GetFloat("memory_percent"); memPercent < 39 || memPercent > 40 {
		t.Errorf("Expected memory_percent~39.06, got %f", memPercent)
	}

	if diskPercent := retrieved.GetFloat("disk_usage_percent"); diskPercent != 45.0 {
		t.Errorf("Expected disk_usage_percent=45.0, got %f", diskPercent)
	}

	if load1 := retrieved.GetFloat("load_avg_1min"); load1 != 4.5 {
		t.Errorf("Expected load_avg_1min=4.5, got %f", load1)
	}

	if load5 := retrieved.GetFloat("load_avg_5min"); load5 != 3.75 {
		t.Errorf("Expected load_avg_5min=3.75, got %f", load5)
	}

	if load15 := retrieved.GetFloat("load_avg_15min"); load15 != 3.2 {
		t.Errorf("Expected load_avg_15min=3.2, got %f", load15)
	}

	filesystemsData := retrieved.GetString("filesystems")
	if !strings.Contains(filesystemsData, "/dev/nvme0n1p1") {
		t.Errorf("Expected filesystems to contain /dev/nvme0n1p1, got %s", filesystemsData)
	}
}

// TestFilesystemStatsStructure tests the FilesystemStats type
func TestFilesystemStatsStructure(t *testing.T) {
	fs := types.FilesystemStats{
		Device:       "/dev/sda1",
		MountPath:    "/",
		UsedGB:       250.5,
		FreeGB:       261.5,
		TotalGB:      512.0,
		UsagePercent: 48.93,
	}

	if fs.Device != "/dev/sda1" {
		t.Errorf("Expected device=/dev/sda1, got %s", fs.Device)
	}

	if fs.MountPath != "/" {
		t.Errorf("Expected mount_path=/, got %s", fs.MountPath)
	}

	if fs.UsagePercent != 48.93 {
		t.Errorf("Expected usage_percent=48.93, got %f", fs.UsagePercent)
	}

	if fs.UsedGB != 250.5 {
		t.Errorf("Expected used_gb=250.5, got %f", fs.UsedGB)
	}

	if fs.FreeGB != 261.5 {
		t.Errorf("Expected free_gb=261.5, got %f", fs.FreeGB)
	}

	if fs.TotalGB != 512.0 {
		t.Errorf("Expected total_gb=48.93, got %f", fs.TotalGB)
	}
}

// TestLoadAverageStructure tests the LoadAverage type
func TestLoadAverageStructure(t *testing.T) {
	la := types.LoadAverage{
		OneMin:     2.45,
		FiveMin:    2.10,
		FifteenMin: 1.95,
	}

	if la.OneMin != 2.45 {
		t.Errorf("Expected one_min=2.45, got %f", la.OneMin)
	}

	if la.FiveMin != 2.10 {
		t.Errorf("Expected five_min=2.10, got %f", la.FiveMin)
	}

	if la.FifteenMin != 1.95 {
		t.Errorf("Expected fifteen_min=1.95, got %f", la.FifteenMin)
	}
}

// TestSystemMetricsWithComputedValues tests SystemMetrics with computed percentages
func TestSystemMetricsWithComputedValues(t *testing.T) {
	metrics := types.SystemMetrics{
		CPUPercent:       45.5,
		CPUCores:         8,
		MemoryUsedGB:     8.2,
		MemoryTotalGB:    16.0,
		MemoryPercent:    51.25,
		DiskUsedGB:       250.5,
		DiskTotalGB:      512.0,
		DiskUsagePercent: 48.93,
		Filesystems: []types.FilesystemStats{
			{
				Device:       "/dev/sda1",
				MountPath:    "/",
				UsedGB:       250.5,
				FreeGB:       261.5,
				TotalGB:      512.0,
				UsagePercent: 48.93,
			},
		},
		LoadAverage: types.LoadAverage{
			OneMin:     2.45,
			FiveMin:    2.10,
			FifteenMin: 1.95,
		},
		NetworkStats: types.NetworkStats{
			InGB:  0.5,
			OutGB: 0.3,
		},
	}

	// Verify JSON marshaling
	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal metrics: %v", err)
	}

	var unmarshalled types.SystemMetrics
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Fatalf("Failed to unmarshal metrics: %v", err)
	}

	if unmarshalled.CPUCores != 8 {
		t.Errorf("Expected cpu_cores=8, got %d", unmarshalled.CPUCores)
	}

	if unmarshalled.MemoryPercent != 51.25 {
		t.Errorf("Expected memory_percent=51.25, got %f", unmarshalled.MemoryPercent)
	}

	if len(unmarshalled.Filesystems) != 1 {
		t.Errorf("Expected 1 filesystem, got %d", len(unmarshalled.Filesystems))
	}

	if unmarshalled.LoadAverage.OneMin != 2.45 {
		t.Errorf("Expected load_avg_1min=2.45, got %f", unmarshalled.LoadAverage.OneMin)
	}
}
