package tools

import (
	"context"
)

// Tool is the interface all tools must implement
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns the tool description
	Description() string

	// Schema returns the JSON schema for tool parameters
	Schema() map[string]any

	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args map[string]any) (*ToolResult, error)
}

// ToolResult is the result of tool execution
type ToolResult struct {
	Content  string         // Text content to return
	Metadata map[string]any // Optional metadata
	IsError  bool           // Whether this is an error result
}

// NewToolResult creates a successful result
func NewToolResult(content string) *ToolResult {
	return &ToolResult{Content: content}
}

// NewToolError creates an error result
func NewToolError(message string) *ToolResult {
	return &ToolResult{
		Content: message,
		IsError: true,
	}
}

// ArgumentParser helps parse tool arguments
type ArgumentParser struct {
	args map[string]any
}

// NewArgumentParser creates a new parser
func NewArgumentParser(args map[string]any) *ArgumentParser {
	return &ArgumentParser{args: args}
}

// GetString returns a string argument
func (p *ArgumentParser) GetString(key string) string {
	if v, ok := p.args[key].(string); ok {
		return v
	}
	return ""
}

// GetStringRequired returns a required string argument
func (p *ArgumentParser) GetStringRequired(key string) (string, error) {
	v := p.GetString(key)
	if v == "" {
		return "", ErrMissingRequired{Field: key}
	}
	return v, nil
}

// GetInt returns an int argument
func (p *ArgumentParser) GetInt(key string, defaultVal int) int {
	switch v := p.args[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	default:
		return defaultVal
	}
}

// GetFloat returns a float argument
func (p *ArgumentParser) GetFloat(key string, defaultVal float64) float64 {
	switch v := p.args[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return defaultVal
	}
}

// GetBool returns a bool argument
func (p *ArgumentParser) GetBool(key string, defaultVal bool) bool {
	if v, ok := p.args[key].(bool); ok {
		return v
	}
	return defaultVal
}

// GetStringArray returns a string array argument
func (p *ArgumentParser) GetStringArray(key string) []string {
	v, ok := p.args[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(v))
	for _, item := range v {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// GetObjectArray returns an array of objects
func (p *ArgumentParser) GetObjectArray(key string) []map[string]any {
	v, ok := p.args[key].([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(v))
	for _, item := range v {
		if m, ok := item.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result
}

// Error types
type ErrMissingRequired struct {
	Field string
}

func (e ErrMissingRequired) Error() string {
	return "missing required field: " + e.Field
}

type ErrInvalidValue struct {
	Field   string
	Message string
}

func (e ErrInvalidValue) Error() string {
	return "invalid value for " + e.Field + ": " + e.Message
}
