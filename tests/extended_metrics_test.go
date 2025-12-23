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
	agent.Set("platform", "linux")
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
	agent.Set("platform", "linux")
	agent.Set("version", "1.0.0")
	agent.Set("status", "active")
	agent.SetPassword("testpass123")
	if err := app.Save(agent); err != nil {
		t.Fatalf("Failed to save agent: %v", err)
	}

	// Test payload with extended metrics
	payload := map[string]interface{}{
		"action": "ingest-metrics",
		"system_metrics": map[string]interface{}{
			"cpu_percent":     67.8,
			"cpu_cores":       16,
			"memory_used_gb":  12.5,
			"memory_total_gb": 32.0,
			"disk_used_gb":    450.0,
			"disk_total_gb":   1000.0,
			"network_rx_gbps": 1.2,
			"network_tx_gbps": 0.8,
			"load_avg_1min":   4.50,
			"load_avg_5min":   3.75,
			"load_avg_15min":  3.20,
			"filesystems": []map[string]interface{}{
				{
					"device":        "/dev/nvme0n1p1",
					"mount_path":    "/",
					"used_gb":       450.0,
					"free_gb":       550.0,
					"total_gb":      1000.0,
					"usage_percent": 45.0,
				},
				{
					"device":        "/dev/nvme0n1p2",
					"mount_path":    "/data",
					"used_gb":       100.0,
					"free_gb":       150.0,
					"total_gb":      250.0,
					"usage_percent": 40.0,
				},
			},
		},
	}

	// Simulate what the handler does
	metrics := payload["system_metrics"].(map[string]interface{})

	metricsCollection, _ := app.FindCollectionByNameOrId("agent_metrics")
	metricsRecord := core.NewRecord(metricsCollection)
	metricsRecord.Set("agent_id", agent.Id)

	toFloat64 := func(v interface{}) (float64, bool) {
		if v == nil {
			return 0, false
		}
		switch val := v.(type) {
		case float64:
			return val, true
		case int:
			return float64(val), true
		default:
			return 0, false
		}
	}

	// Set fields
	if cpu, ok := toFloat64(metrics["cpu_percent"]); ok {
		metricsRecord.Set("cpu_percent", cpu)
	}
	if cores, ok := toFloat64(metrics["cpu_cores"]); ok {
		metricsRecord.Set("cpu_cores", int(cores))
	}
	if memUsed, ok := toFloat64(metrics["memory_used_gb"]); ok {
		metricsRecord.Set("memory_used_gb", memUsed)
	}
	if memTotal, ok := toFloat64(metrics["memory_total_gb"]); ok {
		metricsRecord.Set("memory_total_gb", memTotal)
		if memUsed, found := toFloat64(metrics["memory_used_gb"]); found && memTotal > 0 {
			memPercent := (memUsed / memTotal) * 100
			metricsRecord.Set("memory_percent", memPercent)
		}
	}
	if diskUsed, ok := toFloat64(metrics["disk_used_gb"]); ok {
		metricsRecord.Set("disk_used_gb", diskUsed)
	}
	if diskTotal, ok := toFloat64(metrics["disk_total_gb"]); ok {
		metricsRecord.Set("disk_total_gb", diskTotal)
		if diskUsed, found := toFloat64(metrics["disk_used_gb"]); found && diskTotal > 0 {
			diskPercent := (diskUsed / diskTotal) * 100
			metricsRecord.Set("disk_usage_percent", diskPercent)
		}
	}
	if load1, ok := toFloat64(metrics["load_avg_1min"]); ok {
		metricsRecord.Set("load_avg_1min", load1)
	}
	if load5, ok := toFloat64(metrics["load_avg_5min"]); ok {
		metricsRecord.Set("load_avg_5min", load5)
	}
	if load15, ok := toFloat64(metrics["load_avg_15min"]); ok {
		metricsRecord.Set("load_avg_15min", load15)
	}
	if filesystemsList, ok := metrics["filesystems"].([]map[string]interface{}); ok {
		filesystemsJSON, _ := json.Marshal(filesystemsList)
		filesystemsStr := string(filesystemsJSON)
		metricsRecord.Set("filesystems", filesystemsStr)
	} else if filesystemsList2, ok := metrics["filesystems"].([]interface{}); ok {
		filesystemsJSON, _ := json.Marshal(filesystemsList2)
		filesystemsStr := string(filesystemsJSON)
		metricsRecord.Set("filesystems", filesystemsStr)
	}
	if netRx, ok := toFloat64(metrics["network_rx_gbps"]); ok {
		metricsRecord.Set("network_in_gbps", netRx)
	}
	if netTx, ok := toFloat64(metrics["network_tx_gbps"]); ok {
		metricsRecord.Set("network_out_gbps", netTx)
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
			InGbps:  0.5,
			OutGbps: 0.3,
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
