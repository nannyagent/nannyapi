package types

import "time"

// RebootStatus represents the reboot operation lifecycle
type RebootStatus string

const (
	RebootStatusPending   RebootStatus = "pending"
	RebootStatusSent      RebootStatus = "sent"
	RebootStatusRebooting RebootStatus = "rebooting"
	RebootStatusCompleted RebootStatus = "completed"
	RebootStatusFailed    RebootStatus = "failed"
	RebootStatusTimeout   RebootStatus = "timeout"
)

// RebootRequest is sent by user to initiate a reboot
type RebootRequest struct {
	AgentID        string `json:"agent_id" validate:"required"`
	LxcID          string `json:"lxc_id,omitempty"`
	Reason         string `json:"reason,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"` // Default: 300 (5 min)
}

// RebootResponse is returned when reboot operation is created
type RebootResponse struct {
	ID             string       `json:"id"`
	UserID         string       `json:"user_id"`
	AgentID        string       `json:"agent_id"`
	LxcID          string       `json:"lxc_id,omitempty"`
	Vmid           string       `json:"vmid,omitempty"`
	Status         RebootStatus `json:"status"`
	Reason         string       `json:"reason,omitempty"`
	TimeoutSeconds int          `json:"timeout_seconds"`
	RequestedAt    time.Time    `json:"requested_at"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// RebootListItem represents a reboot operation in a list
type RebootListItem struct {
	ID             string       `json:"id"`
	AgentID        string       `json:"agent_id"`
	LxcID          string       `json:"lxc_id,omitempty"`
	Vmid           string       `json:"vmid,omitempty"`
	Status         RebootStatus `json:"status"`
	Reason         string       `json:"reason,omitempty"`
	RequestedAt    time.Time    `json:"requested_at"`
	AcknowledgedAt *time.Time   `json:"acknowledged_at,omitempty"`
	CompletedAt    *time.Time   `json:"completed_at,omitempty"`
	ErrorMessage   string       `json:"error_message,omitempty"`
	TimeoutSeconds int          `json:"timeout_seconds"`
}

// RebootListResponse is returned when listing reboots
type RebootListResponse struct {
	Reboots []RebootListItem `json:"reboots"`
}

// RebootAcknowledgeResponse is returned when agent acknowledges reboot
type RebootAcknowledgeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
