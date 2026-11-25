package domain

import "time"

const (
	// MaxToolRounds bounds the number of DeepSeek tool-calling iterations per job.
	MaxToolRounds = 5
	// MaxHistoryMessages caps how many recent conversation messages are sent to the model.
	MaxHistoryMessages = 20
	// MaxInstructionsRunes caps organization ai_instructions length in runes.
	MaxInstructionsRunes = 4000
)

// ChatMessage represents a single OpenAI-compatible chat message.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolDefinition describes a callable function tool for Chat Completions.
type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a tool's name, description, and JSON Schema parameters.
type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// ToolCall represents a tool invocation requested by the model.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction carries the function name and raw JSON arguments.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// CompletionResult is the parsed outcome of a single chat completion call.
type CompletionResult struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
}

// DayHours describes one weekday's opening hours for prompt injection.
type DayHours struct {
	Day       string
	IsOpen    bool
	OpenTime  string
	CloseTime string
}

// PromptConfig carries organization schedule and instructions for prompt assembly.
type PromptConfig struct {
	Timezone       string
	BusinessHours  []DayHours
	AiInstructions string
	Now            time.Time
}

// HistoryMessage is a minimal conversation message for prompt history.
type HistoryMessage struct {
	Direction string
	Body      string
}
