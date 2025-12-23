package types

import "time"

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	AgentStatusActive   AgentStatus = "active"
	AgentStatusInactive AgentStatus = "inactive"
	AgentStatusRevoked  AgentStatus = "revoked"
)

// AgentHealthStatus represents health check status
type AgentHealthStatus string

const (
	HealthStatusHealthy  AgentHealthStatus = "healthy"
	HealthStatusStale    AgentHealthStatus = "stale"
	HealthStatusInactive AgentHealthStatus = "inactive"
)

// DeviceAuthRequest - anonymous device auth start
type DeviceAuthRequest struct {
	Action string `json:"action"` // "device-auth-start"
}

// DeviceAuthResponse - response with device & user codes
type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"` // seconds
}

// AuthorizeRequest - user authorizes device code
type AuthorizeRequest struct {
	Action   string `json:"action"`    // "authorize"
	UserCode string `json:"user_code"` // 8-char code
}

// AuthorizeResponse - confirmation of authorization
type AuthorizeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RegisterRequest - agent registers with device code
type RegisterRequest struct {
	Action         string   `json:"action"`          // "register"
	DeviceCode     string   `json:"device_code"`     // UUID from device-auth-start
	Hostname       string   `json:"hostname"`        // Agent hostname
	OSType         string   `json:"os_type"`         // OS type (linux, darwin, windows)
	OSInfo         string   `json:"os_info"`         // OS info (e.g. "Ubuntu 22.04 LTS")
	OSVersion      string   `json:"os_version"`      // OS version (e.g. "22.04")
	Version        string   `json:"version"`         // Agent version
	PrimaryIP      string   `json:"primary_ip"`      // Primary IP address (WAN/eth0)
	KernelVersion  string   `json:"kernel_version"`  // Kernel version
	Arch           string   `json:"arch"`            // CPU architecture (amd64, arm64)
	AllIPs         []string `json:"all_ips"`         // All IP addresses from all NICs
	PlatformFamily string   `json:"platform_family"` // Platform family (debian, redhat, etc.)
}

// TokenResponse - access & refresh tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
	AgentID      string `json:"agent_id"`
}

// RefreshRequest - refresh access token
type RefreshRequest struct {
	Action       string `json:"action"`        // "refresh"
	RefreshToken string `json:"refresh_token"` // Current refresh token
}

// NetworkStats contains network metrics in GB
type NetworkStats struct {
	InGB  float64 `json:"in_gb"`
	OutGB float64 `json:"out_gb"`
}

// FilesystemStats contains filesystem information
type FilesystemStats struct {
	Device       string  `json:"device"`     // e.g., "/dev/sda1"
	MountPath    string  `json:"mount_path"` // e.g., "/"
	UsedGB       float64 `json:"used_gb"`
	FreeGB       float64 `json:"free_gb"`
	TotalGB      float64 `json:"total_gb"`
	UsagePercent float64 `json:"usage_percent"` // Used / Total * 100
}

// LoadAverage contains load average metrics
type LoadAverage struct {
	OneMin     float64 `json:"one_min"`     // 1 minute load average
	FiveMin    float64 `json:"five_min"`    // 5 minute load average
	FifteenMin float64 `json:"fifteen_min"` // 15 minute load average
}

// SystemMetrics contains all system metrics from agent
type SystemMetrics struct {
	CPUPercent       float64           `json:"cpu_percent"`
	CPUCores         int               `json:"cpu_cores"`
	MemoryUsedGB     float64           `json:"memory_used_gb"`
	MemoryTotalGB    float64           `json:"memory_total_gb"`
	MemoryPercent    float64           `json:"memory_percent"` // Computed: used/total*100
	DiskUsedGB       float64           `json:"disk_used_gb"`
	DiskTotalGB      float64           `json:"disk_total_gb"`
	DiskUsagePercent float64           `json:"disk_usage_percent"` // Computed: used/total*100
	Filesystems      []FilesystemStats `json:"filesystems"`        // List of filesystems
	LoadAverage      LoadAverage       `json:"load_average"`
	NetworkStats     NetworkStats      `json:"network_stats"`
}

// IngestMetricsRequest - agent sends metrics every 30s
type IngestMetricsRequest struct {
	Action        string        `json:"action"`         // "ingest-metrics"
	SystemMetrics SystemMetrics `json:"system_metrics"` // System metrics (new format)

	// Agent metadata updates
	OSInfo        string   `json:"os_info,omitempty"`
	OSVersion     string   `json:"os_version,omitempty"`
	Version       string   `json:"version,omitempty"`
	PrimaryIP     string   `json:"primary_ip,omitempty"`
	KernelVersion string   `json:"kernel_version,omitempty"`
	KernelFamily  string   `json:"platform_family,omitempty"`
	Arch          string   `json:"arch,omitempty"`
	AllIPs        []string `json:"all_ips,omitempty"`
}

// IngestMetricsResponse - confirmation
type IngestMetricsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ListAgentsRequest - list user's agents
type ListAgentsRequest struct {
	Action string `json:"action"` // "list"
}

// AgentListItem - single agent in list
type AgentListItem struct {
	ID            string            `json:"id"`
	Hostname      string            `json:"hostname"`
	OSType        string            `json:"os_type"`
	OSInfo        string            `json:"os_info"`
	OSVersion     string            `json:"os_version"`
	Version       string            `json:"version"`
	Status        AgentStatus       `json:"status"`
	Health        AgentHealthStatus `json:"health"`
	LastSeen      *time.Time        `json:"last_seen"`
	Created       time.Time         `json:"created"`
	KernelVersion string            `json:"kernel_version"`
	Arch          string            `json:"arch"`
}

// ListAgentsResponse - list of agents
type ListAgentsResponse struct {
	Agents []AgentListItem `json:"agents"`
}

// RevokeAgentRequest - revoke agent access
type RevokeAgentRequest struct {
	Action  string `json:"action"`   // "revoke"
	AgentID string `json:"agent_id"` // Agent to revoke
}

// RevokeAgentResponse - confirmation
type RevokeAgentResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// HealthRequest - get agent health & latest metrics
type HealthRequest struct {
	Action  string `json:"action"`   // "health"
	AgentID string `json:"agent_id"` // Agent to check
}

// HealthResponse - agent health status with metrics
type HealthResponse struct {
	AgentID       string            `json:"agent_id"`
	Status        AgentStatus       `json:"status"`
	Health        AgentHealthStatus `json:"health"`
	LastSeen      *time.Time        `json:"last_seen"`
	LatestMetrics *SystemMetrics    `json:"latest_metrics"` // nil if no metrics
}

// ErrorResponse - standard error response
type ErrorResponse struct {
	Error string `json:"error"`
}
