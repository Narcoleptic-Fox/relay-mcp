package simple

import (
	"context"
	    "fmt"
	    "strings"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/providers"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/tools"
	)
	// ListModelsTool lists available models
type ListModelsTool struct {
	cfg      *config.Config
	registry *providers.Registry
}

// NewListModelsTool creates a new listmodels tool
func NewListModelsTool(cfg *config.Config, registry *providers.Registry) *ListModelsTool {
	return &ListModelsTool{
		cfg:      cfg,
		registry: registry,
	}
}

func (t *ListModelsTool) Name() string {
	return "listmodels"
}

func (t *ListModelsTool) Description() string {
	return "Shows which AI model providers are configured, available model names, their aliases and capabilities."
}

func (t *ListModelsTool) Schema() map[string]any {
	return tools.NewSchemaBuilder().Build()
}

func (t *ListModelsTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	models := t.registry.GetAllModels()

	var sb strings.Builder
	sb.WriteString("# Available Models\n\n")

	currentProvider := ""
	for _, m := range models {
		if string(m.Provider) != currentProvider {
			currentProvider = string(m.Provider)
			sb.WriteString(fmt.Sprintf("\n## %s\n\n", strings.ToUpper(currentProvider)))
		}

		aliases := ""
		if len(m.Aliases) > 0 {
			aliases = fmt.Sprintf(" (aliases: %s)", strings.Join(m.Aliases, ", "))
		}

		features := []string{}
		if m.SupportsExtendedThinking {
			features = append(features, "thinking")
		}
		if m.SupportsVision {
			features = append(features, "vision")
		}
		if m.AllowCodeGeneration {
			features = append(features, "code-gen")
		}

		sb.WriteString(fmt.Sprintf("- **%s**%s\n", m.ModelName, aliases))
		sb.WriteString(fmt.Sprintf("  - Score: %d | Context: %dk | Features: %s\n",
			m.IntelligenceScore,
			m.ContextWindow/1000,
			strings.Join(features, ", "),
		))
	}

	return tools.NewToolResult(sb.String()), nil
}
