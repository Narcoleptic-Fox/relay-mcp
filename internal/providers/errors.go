package providers

import (
	"fmt"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/types"
)

// ErrModelNotFound indicates a model wasn't found
type ErrModelNotFound struct {
	Model    string
	Provider types.ProviderType
}

func (e ErrModelNotFound) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("model %q not found in provider %s", e.Model, e.Provider)
	}
	return fmt.Sprintf("model %q not found in any provider", e.Model)
}

// ErrProviderNotConfigured indicates a provider isn't configured
type ErrProviderNotConfigured struct {
	Provider types.ProviderType
}

func (e ErrProviderNotConfigured) Error() string {
	return fmt.Sprintf("provider %s not configured", e.Provider)
}

// ErrAPIError indicates an API error
type ErrAPIError struct {
	Provider   types.ProviderType
	StatusCode int
	Message    string
}

func (e ErrAPIError) Error() string {
	return fmt.Sprintf("%s API error (%d): %s", e.Provider, e.StatusCode, e.Message)
}
