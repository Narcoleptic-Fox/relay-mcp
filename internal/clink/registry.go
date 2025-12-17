package clink

import (
	    "sync"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	)
	// Registry manages CLI agents
type Registry struct {
	agents map[string]Agent
	mu     sync.RWMutex
}

// NewRegistry creates a new agent registry
func NewRegistry(cfg *config.Config) (*Registry, error) {
	r := &Registry{
		agents: make(map[string]Agent),
	}

	// Register configured CLI clients
	for name, clientCfg := range cfg.CLIClients {
		agent, err := r.createAgent(name, clientCfg, cfg)
		if err != nil {
			// Log warning but continue
			continue
		}

		if agent.IsAvailable() {
			r.agents[name] = agent
		}
	}

	return r, nil
}

func (r *Registry) createAgent(name string, clientCfg config.CLIClientConfig, cfg *config.Config) (Agent, error) {
	switch name {
	case "gemini":
		return NewGeminiAgent(clientCfg, cfg), nil
	case "claude":
		return NewClaudeAgent(clientCfg, cfg), nil
	case "codex":
		return NewCodexAgent(clientCfg, cfg), nil
	default:
		return NewGenericAgent(clientCfg, cfg), nil
	}
}

// Get returns an agent by name
func (r *Registry) Get(name string) (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[name]
	return agent, ok
}

// List returns all available agents
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}
