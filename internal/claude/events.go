package claude

import "encoding/json"

// Envelope is the raw JSONL line as emitted by `claude -p --output-format stream-json`.
// Fields are a superset across all event types; unset fields are zero/nil.
type Envelope struct {
	Type            string          `json:"type"`
	Subtype         string          `json:"subtype,omitempty"`
	SessionID       string          `json:"session_id,omitempty"`
	UUID            string          `json:"uuid,omitempty"`
	ParentToolUseID string          `json:"parent_tool_use_id,omitempty"`
	Event           json.RawMessage `json:"event,omitempty"`
	Message         json.RawMessage `json:"message,omitempty"`
	RateLimitInfo   *RateLimitInfo  `json:"rate_limit_info,omitempty"`

	HookID    string `json:"hook_id,omitempty"`
	HookName  string `json:"hook_name,omitempty"`
	HookEvent string `json:"hook_event,omitempty"`
	Outcome   string `json:"outcome,omitempty"`
	ExitCode  int    `json:"exit_code,omitempty"`
	Stdout    string `json:"stdout,omitempty"`
	Stderr    string `json:"stderr,omitempty"`

	CWD               string      `json:"cwd,omitempty"`
	Tools             []string    `json:"tools,omitempty"`
	MCPServers        []MCPServer `json:"mcp_servers,omitempty"`
	Model             string      `json:"model,omitempty"`
	PermissionMode    string      `json:"permissionMode,omitempty"`
	SlashCommands     []string    `json:"slash_commands,omitempty"`
	APIKeySource      string      `json:"apiKeySource,omitempty"`
	ClaudeCodeVersion string      `json:"claude_code_version,omitempty"`

	IsError        bool                      `json:"is_error,omitempty"`
	DurationMS     int64                     `json:"duration_ms,omitempty"`
	NumTurns       int                       `json:"num_turns,omitempty"`
	Result         string                    `json:"result,omitempty"`
	StopReason     string                    `json:"stop_reason,omitempty"`
	TotalCostUSD   float64                   `json:"total_cost_usd,omitempty"`
	Usage          *Usage                    `json:"usage,omitempty"`
	ModelUsage     map[string]ModelUsageInfo `json:"modelUsage,omitempty"`
	TerminalReason string                    `json:"terminal_reason,omitempty"`

	Status string `json:"status,omitempty"`
	TTFTMs int64  `json:"ttft_ms,omitempty"`
}

type MCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Usage struct {
	InputTokens              int    `json:"input_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	ServiceTier              string `json:"service_tier,omitempty"`
}

func (u Usage) Total() int {
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens + u.OutputTokens
}

type ModelUsageInfo struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	CostUSD                  float64 `json:"costUSD"`
	ContextWindow            int     `json:"contextWindow"`
	MaxOutputTokens          int     `json:"maxOutputTokens"`
}

type RateLimitInfo struct {
	Status          string `json:"status"`
	ResetsAt        int64  `json:"resetsAt"`
	RateLimitType   string `json:"rateLimitType"`
	OverageStatus   string `json:"overageStatus"`
	OverageResetsAt int64  `json:"overageResetsAt"`
	IsUsingOverage  bool   `json:"isUsingOverage"`
}

// StreamEvent is the inner payload of {"type":"stream_event","event":...}.
type StreamEvent struct {
	Type         string        `json:"type"`
	Index        int           `json:"index,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"`
	Delta        *ContentDelta `json:"delta,omitempty"`
	Message      *AssistantMsg `json:"message,omitempty"`
	Usage        *Usage        `json:"usage,omitempty"`
}

type ContentBlock struct {
	Type  string          `json:"type"` // "text" | "thinking" | "tool_use"
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ContentDelta covers all delta variants in stream_event.event.delta.
// Content-block deltas use Type ∈ {text_delta, thinking_delta, input_json_delta}.
// message_delta uses StopReason / StopSequence / StopDetails (no Type).
type ContentDelta struct {
	Type         string          `json:"type,omitempty"`
	Text         string          `json:"text,omitempty"`
	Thinking     string          `json:"thinking,omitempty"`
	PartialJSON  string          `json:"partial_json,omitempty"`
	StopReason   string          `json:"stop_reason,omitempty"`
	StopSequence string          `json:"stop_sequence,omitempty"`
	StopDetails  json.RawMessage `json:"stop_details,omitempty"`
}

type AssistantMsg struct {
	ID         string         `json:"id"`
	Model      string         `json:"model"`
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason,omitempty"`
	Usage      *Usage         `json:"usage,omitempty"`
}

// ----------------------------------------------------------------------------
// Decoded events emitted on the driver's channel. Use a type switch in callers.

type Event interface{ isEvent() }

type SystemInit struct {
	CWD               string
	Model             string
	PermissionMode    string
	Tools             []string
	MCPServers        []MCPServer
	SlashCommands     []string
	ClaudeCodeVersion string
	SessionID         string
}

func (SystemInit) isEvent() {}

type SystemStatus struct{ Status string }

func (SystemStatus) isEvent() {}

type HookStarted struct{ HookName, HookEvent, HookID string }

func (HookStarted) isEvent() {}

type HookResponse struct {
	HookName, HookEvent, HookID, Outcome string
	ExitCode                             int
	Stdout, Stderr                       string
}

func (HookResponse) isEvent() {}

type MessageStart struct {
	MessageID string
	Model     string
	Usage     Usage
	TTFTMs    int64
}

func (MessageStart) isEvent() {}

type ContentBlockStart struct {
	Index int
	Block ContentBlock
}

func (ContentBlockStart) isEvent() {}

type ContentBlockDelta struct {
	Index int
	Delta ContentDelta
}

func (ContentBlockDelta) isEvent() {}

type ContentBlockStop struct{ Index int }

func (ContentBlockStop) isEvent() {}

type MessageDelta struct {
	StopReason string
	Usage      Usage
}

func (MessageDelta) isEvent() {}

type MessageStop struct{}

func (MessageStop) isEvent() {}

type AssistantSnapshot struct{ Message AssistantMsg }

func (AssistantSnapshot) isEvent() {}

type UserEcho struct{ Text string }

func (UserEcho) isEvent() {}

type RateLimit struct{ Info RateLimitInfo }

func (RateLimit) isEvent() {}

type Result struct {
	Subtype        string
	IsError        bool
	DurationMS     int64
	NumTurns       int
	Result         string
	StopReason     string
	TotalCostUSD   float64
	ModelUsage     map[string]ModelUsageInfo
	TerminalReason string
}

func (Result) isEvent() {}

type ParseError struct {
	Err  error
	Line []byte
}

func (ParseError) isEvent() {}

type DriverExit struct {
	Err error
}

func (DriverExit) isEvent() {}
