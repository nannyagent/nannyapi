package types

import "time"

// PatchMode represents patch operation mode
type PatchMode string

const (
	PatchModeDryRun PatchMode = "dry-run"
	PatchModeApply  PatchMode = "apply"
)

// PatchStatus represents patch operation lifecycle
type PatchStatus string

const (
	PatchStatusPending    PatchStatus = "pending"
	PatchStatusRunning    PatchStatus = "running"
	PatchStatusCompleted  PatchStatus = "completed"
	PatchStatusFailed     PatchStatus = "failed"
	PatchStatusRolledBack PatchStatus = "rolled_back"
)

// PatchOperation represents a patch operation record
// Scripts/outputs are NOT stored in database, only references
type PatchOperation struct {
	ID          string      `json:"id" db:"id"`                   // PocketBase generated UUID
	UserID      string      `json:"user_id" db:"user_id"`         // Operation initiator
	AgentID     string      `json:"agent_id" db:"agent_id"`       // Target agent
	Mode        PatchMode   `json:"mode" db:"mode"`               // dry-run or apply
	Status      PatchStatus `json:"status" db:"status"`           // Operation status
	ScriptURL   string      `json:"script_url" db:"script_url"`   // Reference to script in storage
	OutputPath  string      `json:"output_path" db:"output_path"` // Path in storage for output
	ErrorMsg    string      `json:"error_msg" db:"error_msg"`     // Error if failed (empty if success)
	StartedAt   *time.Time  `json:"started_at" db:"started_at"`
	CompletedAt *time.Time  `json:"completed_at" db:"completed_at"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

// PatchRequest initiates a patch operation
type PatchRequest struct {
	AgentID    string `json:"agent_id" validate:"required,uuid4"`
	Mode       string `json:"mode" validate:"required,oneof=dry-run apply"`
	ScriptURL  string `json:"script_url" validate:"required,url"`
	ScriptArgs string `json:"script_args" validate:"omitempty"` // Script arguments passed to execution
}

// PatchResponse returned when operation is created
type PatchResponse struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	AgentID   string      `json:"agent_id"`
	Mode      PatchMode   `json:"mode"`
	Status    PatchStatus `json:"status"`
	ScriptURL string      `json:"script_url"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// PackageUpdate represents a package that was updated/needs updating
type PackageUpdate struct {
	ID            string    `json:"id" db:"id"`                           // PocketBase generated UUID
	PatchOpID     string    `json:"patch_op_id" db:"patch_op_id"`         // Reference to patch operation
	PackageName   string    `json:"package_name" db:"package_name"`       // Package name
	CurrentVer    string    `json:"current_ver" db:"current_ver"`         // Current installed version
	TargetVer     string    `json:"target_ver" db:"target_ver"`           // Version to update to (or empty if removal)
	UpdateType    string    `json:"update_type" db:"update_type"`         // install, update, remove
	Status        string    `json:"status" db:"status"`                   // pending, applied, failed
	DryRunResults string    `json:"dry_run_results" db:"dry_run_results"` // Dry-run simulation result
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// AgentPatchPayload is sent to agent via realtime for execution
type AgentPatchPayload struct {
	OperationID string `json:"operation_id"`
	Mode        string `json:"mode"` // dry-run or apply
	ScriptURL   string `json:"script_url"`
	ScriptArgs  string `json:"script_args"`
	Timestamp   string `json:"timestamp"`
}

// AgentPatchResult is received from agent after execution
type AgentPatchResult struct {
	OperationID string             `json:"operation_id"`
	Success     bool               `json:"success"`
	OutputPath  string             `json:"output_path"` // Where agent stored output
	ErrorMsg    string             `json:"error_msg"`
	PackageList []PatchPackageInfo `json:"package_list"` // Packages that were changed
	Duration    int64              `json:"duration_ms"`
	Timestamp   string             `json:"timestamp"`
}

// PatchPackageInfo represents package info from agent response
type PatchPackageInfo struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	UpdateType string `json:"update_type"` // install, update, remove
	Details    string `json:"details"`
}
