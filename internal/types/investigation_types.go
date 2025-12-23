package types

import "time"

// InvestigationStatus represents investigation workflow state
type InvestigationStatus string

const (
	InvestigationStatusPending    InvestigationStatus = "pending"
	InvestigationStatusInProgress InvestigationStatus = "in_progress"
	InvestigationStatusCompleted  InvestigationStatus = "completed"
	InvestigationStatusFailed     InvestigationStatus = "failed"
)

// Investigation represents a system investigation record
// Stores user prompt + episode_id, fetches inferences from ClickHouse via episode_id
type Investigation struct {
	ID             string                 `json:"id" db:"id"`                           // PocketBase generated UUID
	UserID         string                 `json:"user_id" db:"user_id"`                 // Investigation owner
	AgentID        string                 `json:"agent_id" db:"agent_id"`               // Agent being investigated
	EpisodeID      string                 `json:"episode_id" db:"episode_id"`           // TensorZero episode reference
	UserPrompt     string                 `json:"user_prompt" db:"user_prompt"`         // Initial user issue description
	Priority       string                 `json:"priority" db:"priority"`               // Priority level: low, medium, high
	Status         InvestigationStatus    `json:"status" db:"status"`                   // Investigation lifecycle status
	ResolutionPlan string                 `json:"resolution_plan" db:"resolution_plan"` // AI-generated resolution from TensorZero
	InitiatedAt    time.Time              `json:"initiated_at" db:"initiated_at"`       // When investigation started
	CompletedAt    *time.Time             `json:"completed_at" db:"completed_at"`       // When investigation completed (nil if ongoing)
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"` // Additional investigation context
}

// InvestigationRequest is sent by frontend to initiate investigation
type InvestigationRequest struct {
	AgentID  string `json:"agent_id" validate:"required,uuid4"`
	Issue    string `json:"issue" validate:"required,min=10,max=2000"`
	Priority string `json:"priority" validate:"omitempty,oneof=low medium high"` // Defaults to medium
}

// InvestigationResponse is returned when investigation is created or retrieved
type InvestigationResponse struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	AgentID        string                 `json:"agent_id"`
	EpisodeID      string                 `json:"episode_id"`
	UserPrompt     string                 `json:"user_prompt"`
	Priority       string                 `json:"priority"`
	Status         InvestigationStatus    `json:"status"`
	ResolutionPlan string                 `json:"resolution_plan"` // AI-generated resolution from TensorZero
	InitiatedAt    time.Time              `json:"initiated_at"`
	CompletedAt    *time.Time             `json:"completed_at"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Metadata       map[string]interface{} `json:"metadata"`
	InferenceCount int                    `json:"inference_count"` // Count from ClickHouse
}

// InvestigationListResponse for listing user's investigations
type InvestigationListResponse struct {
	ID             string              `json:"id"`
	AgentID        string              `json:"agent_id"`
	UserPrompt     string              `json:"user_prompt"`
	Priority       string              `json:"priority"`
	Status         InvestigationStatus `json:"status"`
	InitiatedAt    time.Time           `json:"initiated_at"`
	CompletedAt    *time.Time          `json:"completed_at"`
	CreatedAt      time.Time           `json:"created_at"`
	InferenceCount int                 `json:"inference_count"` // Count from ClickHouse
}

// ClickHouseInference represents inference data from TensorZero's ClickHouse
type ClickHouseInference struct {
	ID               string      `json:"id"`
	FunctionName     string      `json:"function_name"`
	VariantName      string      `json:"variant_name"`
	Timestamp        time.Time   `json:"timestamp"`
	ProcessingTimeMs int64       `json:"processing_time_ms"`
	Input            interface{} `json:"input"`
	Output           interface{} `json:"output"`
	Usage            interface{} `json:"usage"` // Token usage
}

// InferenceDetailsResponse enriched inference with model info
type InferenceDetailsResponse struct {
	ID               string           `json:"id"`
	FunctionName     string           `json:"function_name"`
	VariantName      string           `json:"variant_name"`
	EpisodeID        string           `json:"episode_id"`
	Input            string           `json:"input"`  // User input to function
	Output           string           `json:"output"` // Function output
	ProcessingTimeMs int64            `json:"processing_time_ms"`
	Timestamp        time.Time        `json:"timestamp"`
	ModelInferences  []ModelInference `json:"model_inferences"` // Related LLM calls
	Feedback         []Feedback       `json:"feedback"`         // User feedback from ClickHouse
}

// ModelInference represents an LLM call within an inference
type ModelInference struct {
	ID               string `json:"id"`
	ModelName        string `json:"model_name"`     // e.g., "gpt-4o"
	ModelProvider    string `json:"model_provider"` // e.g., "openai"
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	ResponseTimeMs   int64  `json:"response_time_ms"`
	TimeToFirstToken int64  `json:"ttft_ms"`
}

// Feedback represents user feedback for an inference from ClickHouse
type Feedback struct {
	ID         string      `json:"id"`
	MetricName string      `json:"metric_name"`
	Value      interface{} `json:"value"`
	Timestamp  time.Time   `json:"timestamp"`
}

// TensorZeroModel represents the TensorZero model to use
type TensorZeroModel string

const (
	TensorZeroModelDiagnoseAndHealApplication TensorZeroModel = "tensorzero::function_name::diagnose_and_heal_application"
	TensorZeroModelDiagnoseAndHeal            TensorZeroModel = "tensorzero::function_name::diagnose_and_heal"
)

// TensorZeroCoreRequest is sent to TensorZero for AI analysis
type TensorZeroCoreRequest struct {
	Model    TensorZeroModel `json:"model"` // tensorzero::function_name::diagnose_and_heal or diagnose_and_heal_application
	Messages []ChatMessage   `json:"messages"`
}

// ChatMessage for TensorZero conversation
type ChatMessage struct {
	Role    string `json:"role"` // user, assistant, system
	Content string `json:"content"`
}

// TensorZeroResponse is the complete response from TensorZero Core API
// Matches: /openai/v1/chat/completions
type TensorZeroResponse struct {
	ID                string             `json:"id"`                 // Response ID
	EpisodeID         string             `json:"episode_id"`         // Unique episode identifier
	Choices           []TensorZeroChoice `json:"choices"`            // Completion choices
	Created           int64              `json:"created"`            // Unix timestamp
	Model             string             `json:"model"`              // Model used (e.g., tensorzero::function_name::diagnose_and_heal::variant_name::v1)
	SystemFingerprint string             `json:"system_fingerprint"` // System fingerprint
	ServiceTier       interface{}        `json:"service_tier"`       // Service tier (can be null)
	Object            string             `json:"object"`             // "chat.completion"
	Usage             TokenUsage         `json:"usage"`              // Token usage stats
}

// TensorZeroChoice represents a completion choice from TensorZero
type TensorZeroChoice struct {
	Index        int               `json:"index"`         // Choice index
	FinishReason string            `json:"finish_reason"` // "stop", "length", etc
	Message      TensorZeroMessage `json:"message"`       // Assistant message
}

// TensorZeroMessage represents an assistant message from TensorZero
type TensorZeroMessage struct {
	Role      string        `json:"role"`       // "assistant"
	Content   string        `json:"content"`    // JSON string containing response_type, reasoning, commands, ebpf_programs, etc
	ToolCalls []interface{} `json:"tool_calls"` // Tool calls (if any)
}

// DiagnosticResponse is parsed from TensorZeroMessage.Content
// This is the JSON content within the assistant's message
type DiagnosticResponse struct {
	ResponseType string        `json:"response_type"` // "diagnostic"
	Reasoning    string        `json:"reasoning"`     // Diagnostic reasoning
	Commands     []string      `json:"commands"`      // Shell commands to run
	EBPFPrograms []EBPFProgram `json:"ebpf_programs"` // eBPF tracing programs
}

// EBPFProgram represents an eBPF program to run
type EBPFProgram struct {
	Name        string                 `json:"name"`        // Program name
	Code        string                 `json:"code"`        // Program code/script
	Type        string                 `json:"type"`        // "bpftrace"
	Target      string                 `json:"target"`      // bpftrace script/target
	Duration    int                    `json:"duration"`    // Duration in seconds
	Filters     map[string]interface{} `json:"filters"`     // Optional filters
	Description string                 `json:"description"` // Program description
}

// ResolutionResponse is the final response with resolution plan
type ResolutionResponse struct {
	ResponseType   string `json:"response_type"`   // "resolution"
	RootCause      string `json:"root_cause"`      // Root cause analysis
	ResolutionPlan string `json:"resolution_plan"` // Step-by-step resolution plan
	Confidence     string `json:"confidence"`      // "High", "Medium", "Low"
	EBPFEvidence   string `json:"ebpf_evidence"`   // Evidence from eBPF monitoring
}

// Choice is TensorZero's completion choice (deprecated: use TensorZeroChoice)
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// TokenUsage for TensorZero response
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EpisodeData represents TensorZero episode information
type EpisodeData struct {
	ID                    string    `json:"id"`
	InferenceCount        int       `json:"inference_count"`
	FirstInferenceTime    time.Time `json:"first_inference_time"`
	LastInferenceTime     time.Time `json:"last_inference_time"`
	FunctionsUsed         []string  `json:"functions_used"`
	TotalProcessingTimeMs int64     `json:"total_processing_time_ms"`
}

// InferenceData represents TensorZero inference details
type InferenceData struct {
	ID               string                 `json:"id"`
	FunctionName     string                 `json:"function_name"`
	VariantName      string                 `json:"variant_name"`
	EpisodeID        string                 `json:"episode_id"`
	Input            map[string]interface{} `json:"input"`
	Output           []OutputMessage        `json:"output"`
	ProcessingTimeMs int64                  `json:"processing_time_ms"`
	Timestamp        time.Time              `json:"timestamp"`
	ModelInferences  []ModelInference       `json:"model_inferences"`
}

// OutputMessage represents output from an inference
type OutputMessage struct {
	Type string `json:"type"` // text, json, etc
	Text string `json:"text"`
}
