package models

import (
	"time"
)

// AntigravityHookInput is the common stdin payload from Antigravity hooks
type AntigravityHookInput struct {
	ConversationID        string                 `json:"conversationId"`
	WorkspacePaths        []string               `json:"workspacePaths"`
	TranscriptPath        string                 `json:"transcriptPath"`
	ArtifactDirectoryPath string                 `json:"artifactDirectoryPath"`
	ModelName             string                 `json:"modelName"`
	StepIdx               int                    `json:"stepIdx,omitempty"`
	ToolCall              *ToolCall              `json:"toolCall,omitempty"`
	InvocationNum         int                    `json:"invocationNum,omitempty"`
	TerminationReason     string                 `json:"terminationReason,omitempty"`
	Error                 string                 `json:"error,omitempty"`
	FullyIdle             bool                   `json:"fullyIdle,omitempty"`
}

type ToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// HookDecisionOutput is the response returned to Antigravity stdout
type HookDecisionOutput struct {
	Decision            string                 `json:"decision"` // "allow", "deny", "ask", "force_ask"
	Reason              string                 `json:"reason,omitempty"`
	PermissionOverrides []string               `json:"permissionOverrides,omitempty"`
	Overwrite           map[string]interface{} `json:"overwrite,omitempty"`
	InjectSteps         []InjectStep           `json:"injectSteps,omitempty"`
}

type InjectStep struct {
	UserMessage      string    `json:"userMessage,omitempty"`
	EphemeralMessage string    `json:"ephemeralMessage,omitempty"`
	ToolCall         *ToolCall `json:"toolCall,omitempty"`
}

// Approval represents an action waiting for human decision
type Approval struct {
	ID             string                 `json:"id"`
	Workspace      string                 `json:"workspace"`
	ConversationID string                 `json:"conversationId"`
	StepIdx        int                    `json:"stepIdx"`
	ToolName       string                 `json:"toolName"`
	ToolArgs       map[string]interface{} `json:"toolArgs"`
	CommandLine    string                 `json:"commandLine,omitempty"`
	FilePath       string                 `json:"filePath,omitempty"`
	Diff           string                 `json:"diff,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Status         string                 `json:"status"` // "pending", "allowed", "denied", "timed_out"
	IsCritical     bool                   `json:"isCritical,omitempty"`
	AutoApproved   bool                   `json:"autoApproved,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	DecidedAt      *time.Time             `json:"decidedAt,omitempty"`
	DecisionReason string                 `json:"decisionReason,omitempty"`

	// Unexported channel for synchronous hook waiting
	DecisionChan chan HookDecisionOutput `json:"-"`
}

type ServerSettings struct {
	Mode string `json:"mode"` // "smart", "auto", "manual"
}

// WorkspaceSession tracks active sessions across windows
type WorkspaceSession struct {
	ID             string    `json:"id"` // ConversationID
	Workspace      string    `json:"workspace"`
	WorkspacePath  string    `json:"workspacePath"`
	TranscriptPath string    `json:"transcriptPath"`
	LastToolName   string    `json:"lastToolName"`
	Status         string    `json:"status"` // "idle", "running", "waiting_approval"
	LastActive     time.Time `json:"lastActive"`
}

// WSMessage is the packet sent over WebSocket to clients
type WSMessage struct {
	Type    string      `json:"type"` // "approval_created", "approval_resolved", "session_updated", "transcript_append", "init"
	Payload interface{} `json:"payload"`
}

type UserMessageRecord struct {
	Workspace string    `json:"workspace"`
	CascadeID string    `json:"cascadeId"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
