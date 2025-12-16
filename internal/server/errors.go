package server

import "fmt"

// Error codes
const (
	ErrCodeParse          = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603

	// Custom error codes
	ErrCodeToolNotFound    = -32001
	ErrCodeToolDisabled    = -32002
	ErrCodeProviderError   = -32003
	ErrCodeInvalidArgument = -32004
)

// MCPError is a custom error type
type MCPError struct {
	Code    int
	Message string
	Data    any
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Error constructors
func ErrToolNotFound(name string) *MCPError {
	return &MCPError{
		Code:    ErrCodeToolNotFound,
		Message: fmt.Sprintf("tool not found: %s", name),
	}
}

func ErrToolDisabled(name string) *MCPError {
	return &MCPError{
		Code:    ErrCodeToolDisabled,
		Message: fmt.Sprintf("tool is disabled: %s", name),
	}
}

func ErrProviderError(provider, message string) *MCPError {
	return &MCPError{
		Code:    ErrCodeProviderError,
		Message: fmt.Sprintf("provider %s error: %s", provider, message),
	}
}

func ErrInvalidArgument(name, reason string) *MCPError {
	return &MCPError{
		Code:    ErrCodeInvalidArgument,
		Message: fmt.Sprintf("invalid argument %s: %s", name, reason),
	}
}
