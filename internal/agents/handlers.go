package agents

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/nannyagent/nannyapi/internal/types"
	"github.com/pocketbase/pocketbase/core"
)

// generateRandomPassword creates a strong random password
func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[num.Int64()]
	}
	return string(b), nil
}

// HandleDeviceAuthStart - anonymous agent requests device code
func HandleDeviceAuthStart(app core.App, c *core.RequestEvent, frontendURL string) error {
	deviceCode := generateID()
	userCode := generateUserCode()

	collection, err := app.FindCollectionByNameOrId("device_codes")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "setup error"})
	}

	record := core.NewRecord(collection)
	record.Set("device_code", deviceCode)
	record.Set("user_code", userCode)
	record.Set("authorized", false)
	record.Set("consumed", false)
	record.Set("expires_at", time.Now().Add(10*time.Minute))

	if err := app.Save(record); err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to create device code"})
	}

	verificationURI := strings.TrimSuffix(frontendURL, "/") + "/agent/authorize?user_code=" + userCode

	return c.JSON(http.StatusOK, types.DeviceAuthResponse{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURI: verificationURI,
		ExpiresIn:       600,
	})
}

// HandleAuthorize - user authorizes device code
func HandleAuthorize(app core.App, c *core.RequestEvent) error {
	var req types.AuthorizeRequest
	if err := c.BindBody(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request"})
	}

	if req.UserCode == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "user_code required"})
	}

	// Get authenticated user
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
	}
	user := authRecord.(*core.Record)

	collection, _ := app.FindCollectionByNameOrId("device_codes")
	records, err := app.FindRecordsByFilter(collection, "user_code = {:code}", "", 1, 0, map[string]any{"code": req.UserCode})
	if err != nil || len(records) == 0 {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid user code"})
	}

	record := records[0]
	if time.Now().After(record.GetDateTime("expires_at").Time()) {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "code expired"})
	}

	record.Set("authorized", true)
	record.Set("user_id", user.Id)
	if err := app.Save(record); err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to authorize"})
	}

	return c.JSON(http.StatusOK, types.AuthorizeResponse{
		Success: true,
		Message: "device authorized",
	})
}

// HandleRegister - agent registers using authorized device code
func HandleRegister(app core.App, c *core.RequestEvent) error {
	var req types.RegisterRequest
	if err := c.BindBody(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request"})
	}

	if req.DeviceCode == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "device_code required"})
	}

	deviceCodesCollection, _ := app.FindCollectionByNameOrId("device_codes")
	deviceRecords, err := app.FindRecordsByFilter(deviceCodesCollection, "device_code = {:code}", "", 1, 0, map[string]any{"code": req.DeviceCode})
	if err != nil || len(deviceRecords) == 0 {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid device code"})
	}

	deviceRecord := deviceRecords[0]

	if time.Now().After(deviceRecord.GetDateTime("expires_at").Time()) {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "device code expired"})
	}

	if !deviceRecord.GetBool("authorized") {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "device not authorized yet"})
	}

	if deviceRecord.GetBool("consumed") {
		existingAgentID := deviceRecord.GetString("agent_id")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device code already used", "agent_id": existingAgentID})
	}

	userID := deviceRecord.GetString("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "user_id missing"})
	}

	// Create agent
	agentsCollection, _ := app.FindCollectionByNameOrId("agents")
	agentRecord := core.NewRecord(agentsCollection)
	agentRecord.Set("user_id", userID)
	agentRecord.Set("device_code_id", deviceRecord.Id)
	agentRecord.Set("hostname", req.Hostname)
	agentRecord.Set("platform", req.Platform)
	agentRecord.Set("version", req.Version)
	agentRecord.Set("status", string(types.AgentStatusActive))
	agentRecord.Set("last_seen", time.Now())
	agentRecord.Set("kernel_version", req.KernelVersion)

	// Set random password to satisfy Auth collection requirements
	password, err := generateRandomPassword(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to generate password"})
	}
	agentRecord.SetPassword(password)

	// Store IP addresses if provided
	if req.PrimaryIP != "" {
		agentRecord.Set("primary_ip", req.PrimaryIP)
	}
	if len(req.AllIPs) > 0 {
		ipsJSON, _ := json.Marshal(req.AllIPs)
		agentRecord.Set("all_ips", string(ipsJSON))
	}

	// Generate tokens
	refreshToken := generateID()
	refreshTokenHash := HashToken(refreshToken)

	agentRecord.Set("refresh_token_hash", refreshTokenHash)
	agentRecord.Set("refresh_token_expires", time.Now().Add(30*24*time.Hour))

	if err := app.Save(agentRecord); err != nil {
		app.Logger().Error("Failed to save agent", "error", err)
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to create agent: " + err.Error()})
	}

	// Mark device code as consumed
	deviceRecord.Set("consumed", true)
	deviceRecord.Set("agent_id", agentRecord.Id)
	if err := app.Save(deviceRecord); err != nil {
		log.Printf("Warning: failed to mark device code as consumed: %v", err)
	}

	// Generate PocketBase Auth Token
	accessToken, tokenErr := agentRecord.NewAuthToken()
	if tokenErr != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to generate token"})
	}

	return c.JSON(http.StatusOK, types.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // Default PB token expiry is usually 14 days or configurable, but we can just say 3600 or whatever
		AgentID:      agentRecord.Id,
	})
}

// HandleRefreshToken - refresh access token
func HandleRefreshToken(app core.App, c *core.RequestEvent) error {
	var req types.RefreshRequest
	if err := c.BindBody(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request"})
	}

	if req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "refresh_token required"})
	}

	refreshTokenHash := HashToken(req.RefreshToken)

	collection, _ := app.FindCollectionByNameOrId("agents")
	records, err := app.FindRecordsByFilter(collection, "refresh_token_hash = {:hash}", "", 1, 0, map[string]any{"hash": refreshTokenHash})
	if err != nil || len(records) == 0 {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "invalid refresh token"})
	}

	agentRecord := records[0]

	if time.Now().After(agentRecord.GetDateTime("refresh_token_expires").Time()) {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "refresh token expired"})
	}

	// Generate new PocketBase Auth Token
	accessToken, tokenErr := agentRecord.NewAuthToken()
	if tokenErr != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to generate token"})
	}

	return c.JSON(http.StatusOK, types.TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   3600,
		AgentID:     agentRecord.Id,
	})
}

// HandleIngestMetrics - agent sends metrics every 30s
func HandleIngestMetrics(app core.App, c *core.RequestEvent) error {
	var req types.IngestMetricsRequest
	if err := c.BindBody(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request"})
	}

	// Get authenticated agent from context (set by middleware)
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
	}

	agentRecord, ok := authRecord.(*core.Record)
	if !ok || agentRecord.Collection().Name != "agents" {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "invalid agent authentication"})
	}

	if agentRecord.GetString("status") == string(types.AgentStatusRevoked) {
		return c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "agent revoked"})
	}

	// Update last_seen
	agentRecord.Set("last_seen", time.Now())
	if err := app.Save(agentRecord); err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to update agent"})
	}

	// Store metrics
	metricsCollection, _ := app.FindCollectionByNameOrId("agent_metrics")
	metricsRecord := core.NewRecord(metricsCollection)
	metricsRecord.Set("agent_id", agentRecord.Id)

	// Helper to convert interface{} to float64 - handles JSON number parsing
	toFloat64 := func(v interface{}) (float64, bool) {
		if v == nil {
			return 0, false
		}
		switch val := v.(type) {
		case float64:
			return val, true
		case float32:
			return float64(val), true
		case int:
			return float64(val), true
		case int64:
			return float64(val), true
		default:
			return 0, false
		}
	}

	// Set individual metric fields - support both system_metrics and metrics keys
	metrics := req.Metrics
	if systemMetrics, ok := req.SystemMetrics.(map[string]interface{}); ok && systemMetrics != nil {
		metrics = systemMetrics
	}

	if metrics != nil {
		// CPU - try multiple key names
		if cpu, ok := toFloat64(metrics["cpu_usage"]); ok {
			metricsRecord.Set("cpu_percent", cpu)
		} else if cpu, ok := toFloat64(metrics["cpu_usage_percent"]); ok {
			metricsRecord.Set("cpu_percent", cpu)
		}

		// CPU Cores
		if cores, ok := toFloat64(metrics["cpu_cores"]); ok {
			metricsRecord.Set("cpu_cores", int(cores))
		}

		if memUsed, ok := toFloat64(metrics["memory_used_gb"]); ok {
			metricsRecord.Set("memory_used_gb", memUsed)
		}
		if memTotal, ok := toFloat64(metrics["memory_total_gb"]); ok {
			metricsRecord.Set("memory_total_gb", memTotal)
			// Compute memory percent if we have memory_used_gb
			if memUsed, found := toFloat64(metrics["memory_used_gb"]); found && memTotal > 0 {
				memPercent := (memUsed / memTotal) * 100
				metricsRecord.Set("memory_percent", memPercent)
			}
		}

		// Disk
		if diskUsed, ok := toFloat64(metrics["disk_used_gb"]); ok {
			metricsRecord.Set("disk_used_gb", diskUsed)
		}
		if diskTotal, ok := toFloat64(metrics["disk_total_gb"]); ok {
			metricsRecord.Set("disk_total_gb", diskTotal)
			// Compute disk usage percent if we have disk_used_gb
			if diskUsed, found := toFloat64(metrics["disk_used_gb"]); found && diskTotal > 0 {
				diskPercent := (diskUsed / diskTotal) * 100
				metricsRecord.Set("disk_usage_percent", diskPercent)
			}
		}

		// Load Average
		if load1min, ok := toFloat64(metrics["load_avg_1min"]); ok {
			metricsRecord.Set("load_avg_1min", load1min)
		}
		if load5min, ok := toFloat64(metrics["load_avg_5min"]); ok {
			metricsRecord.Set("load_avg_5min", load5min)
		}
		if load15min, ok := toFloat64(metrics["load_avg_15min"]); ok {
			metricsRecord.Set("load_avg_15min", load15min)
		}

		// Filesystems - store as JSON array (TextField requires JSON string)
		if filesystemsList, ok := metrics["filesystems"].([]interface{}); ok {
			filesystemsJSON, _ := json.Marshal(filesystemsList)
			metricsRecord.Set("filesystems", string(filesystemsJSON))
		} else if filesystemsList2, ok := metrics["filesystems"].([]map[string]interface{}); ok {
			filesystemsJSON, _ := json.Marshal(filesystemsList2)
			metricsRecord.Set("filesystems", string(filesystemsJSON))
		}

		// Network - support both bytes and gbps
		if netRx, ok := toFloat64(metrics["network_rx_bytes"]); ok {
			metricsRecord.Set("network_in_gbps", netRx/1e9)
		} else if netRx, ok := toFloat64(metrics["network_rx_gbps"]); ok {
			metricsRecord.Set("network_in_gbps", netRx)
		}
		if netTx, ok := toFloat64(metrics["network_tx_bytes"]); ok {
			metricsRecord.Set("network_out_gbps", netTx/1e9)
		} else if netTx, ok := toFloat64(metrics["network_tx_gbps"]); ok {
			metricsRecord.Set("network_out_gbps", netTx)
		}
	}

	metricsRecord.Set("recorded_at", time.Now())

	if err := app.Save(metricsRecord); err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to save metrics"})
	}

	return c.JSON(http.StatusOK, types.IngestMetricsResponse{
		Success: true,
		Message: "metrics recorded",
	})
}

// HandleListAgents - list user's agents
func HandleListAgents(app core.App, c *core.RequestEvent) error {
	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
	}
	user := authRecord.(*core.Record)

	collection, _ := app.FindCollectionByNameOrId("agents")
	records, err := app.FindRecordsByFilter(collection, "user_id = {:userId}", "", 100, 0, map[string]any{"userId": user.Id})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to fetch agents"})
	}

	result := make([]types.AgentListItem, 0, len(records))
	for _, agent := range records {
		lastSeen := agent.GetDateTime("last_seen")
		var lastSeenPtr *time.Time
		if !lastSeen.Time().IsZero() {
			t := lastSeen.Time()
			lastSeenPtr = &t
		}

		health := CalculateHealth(lastSeenPtr, types.AgentStatus(agent.GetString("status")))

		result = append(result, types.AgentListItem{
			ID:            agent.Id,
			Hostname:      agent.GetString("hostname"),
			Platform:      agent.GetString("platform"),
			Version:       agent.GetString("version"),
			Status:        types.AgentStatus(agent.GetString("status")),
			Health:        health,
			LastSeen:      lastSeenPtr,
			Created:       agent.GetDateTime("created").Time(),
			KernelVersion: agent.GetString("kernel_version"),
		})
	}

	return c.JSON(http.StatusOK, types.ListAgentsResponse{Agents: result})
}

// HandleRevokeAgent - revoke agent access
func HandleRevokeAgent(app core.App, c *core.RequestEvent) error {
	var req types.RevokeAgentRequest
	if err := c.BindBody(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request"})
	}

	if req.AgentID == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "agent_id required"})
	}

	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
	}
	user := authRecord.(*core.Record)

	collection, _ := app.FindCollectionByNameOrId("agents")
	agentRecord, err := app.FindRecordById(collection, req.AgentID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "agent not found"})
	}

	if agentRecord.GetString("user_id") != user.Id {
		return c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "not your agent"})
	}

	agentRecord.Set("status", string(types.AgentStatusRevoked))
	agentRecord.Set("refresh_token_hash", "")
	agentRecord.Set("refresh_token_expires", nil)

	if err := app.Save(agentRecord); err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to revoke agent"})
	}

	return c.JSON(http.StatusOK, types.RevokeAgentResponse{
		Success: true,
		Message: "agent revoked",
	})
}

// HandleAgentHealth - get agent health & latest metrics
func HandleAgentHealth(app core.App, c *core.RequestEvent) error {
	var req types.HealthRequest
	if err := c.BindBody(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid request"})
	}

	if req.AgentID == "" {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "agent_id required"})
	}

	authRecord := c.Get("authRecord")
	if authRecord == nil {
		return c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "authentication required"})
	}
	user := authRecord.(*core.Record)

	collection, _ := app.FindCollectionByNameOrId("agents")
	agentRecord, err := app.FindRecordById(collection, req.AgentID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "agent not found"})
	}

	if agentRecord.GetString("user_id") != user.Id {
		return c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "not your agent"})
	}

	lastSeen := agentRecord.GetDateTime("last_seen")
	var lastSeenPtr *time.Time
	if !lastSeen.Time().IsZero() {
		t := lastSeen.Time()
		lastSeenPtr = &t
	}

	status := types.AgentStatus(agentRecord.GetString("status"))
	health := CalculateHealth(lastSeenPtr, status)

	// Get latest metrics
	var latestMetrics *types.SystemMetrics
	metricsCollection, _ := app.FindCollectionByNameOrId("agent_metrics")
	metricsRecords, err := app.FindRecordsByFilter(metricsCollection, "agent_id = {:agentId}", "-recorded_at", 1, 0, map[string]any{"agentId": req.AgentID})
	if err == nil && len(metricsRecords) > 0 {
		metricsRecord := metricsRecords[0]
		var metrics types.SystemMetrics
		if err := json.Unmarshal([]byte(metricsRecord.GetString("metrics")), &metrics); err == nil {
			latestMetrics = &metrics
		}
	}

	return c.JSON(http.StatusOK, types.HealthResponse{
		AgentID:       agentRecord.Id,
		Status:        status,
		Health:        health,
		LastSeen:      lastSeenPtr,
		LatestMetrics: latestMetrics,
	})
}

// CalculateHealth determines agent health status
func CalculateHealth(lastSeen *time.Time, status types.AgentStatus) types.AgentHealthStatus {
	if status == types.AgentStatusRevoked || status == types.AgentStatusInactive {
		return types.HealthStatusInactive
	}

	if lastSeen == nil {
		return types.HealthStatusInactive
	}

	timeSince := time.Since(*lastSeen)
	if timeSince < 5*time.Minute {
		return types.HealthStatusHealthy
	} else if timeSince < 15*time.Minute {
		return types.HealthStatusStale
	}
	return types.HealthStatusInactive
}

// CleanupExpiredDeviceCodes removes old device codes
func CleanupExpiredDeviceCodes(app core.App) {
	collection, err := app.FindCollectionByNameOrId("device_codes")
	if err != nil {
		log.Printf("Cleanup error: %v", err)
		return
	}

	oneDayAgo := time.Now().Add(-24 * time.Hour)
	filter := "expires_at < {:now} || (consumed = true && created < {:oneDayAgo})"
	records, err := app.FindRecordsByFilter(collection, filter, "", 100, 0, map[string]any{
		"now":       time.Now(),
		"oneDayAgo": oneDayAgo,
	})
	if err != nil {
		log.Printf("Cleanup query error: %v", err)
		return
	}

	for _, record := range records {
		if err := app.Delete(record); err != nil {
			log.Printf("Failed to delete device code %s: %v", record.Id, err)
		}
	}

	if len(records) > 0 {
		log.Printf("Cleaned up %d device codes", len(records))
	}
}

// Utility functions

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateUserCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	result := make([]byte, 8)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

// HashToken creates SHA-256 hash of token
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
