package clink

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	    "time"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	)
	// BaseAgent provides common functionality for CLI agents
type BaseAgent struct {
	name    string
	command string
	args    []string
	roles   map[string]config.CLIRole
	env     map[string]string
	timeout time.Duration
	cfg     *config.Config
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(clientCfg config.CLIClientConfig, cfg *config.Config) *BaseAgent {
	timeout := 5 * time.Minute
	if clientCfg.Timeout != "" {
		if d, err := time.ParseDuration(clientCfg.Timeout); err == nil {
			timeout = d
		}
	}

	// Load roles and their prompts
	roles := make(map[string]config.CLIRole)
	for name, roleCfg := range clientCfg.Roles {
		// Load prompt content
		systemPrompt := loadPromptFile(roleCfg.PromptPath)

		roles[name] = config.CLIRole{
			PromptPath:   roleCfg.PromptPath,
			SystemPrompt: systemPrompt,
			Args:         roleCfg.Args,
		}
	}

	return &BaseAgent{
		name:    clientCfg.Name,
		command: clientCfg.Command,
		args:    clientCfg.AdditionalArgs,
		roles:   roles,
		timeout: timeout,
		cfg:     cfg,
	}
}

func (a *BaseAgent) Name() string {
	return a.name
}

// IsAvailable checks if the CLI executable exists
func (a *BaseAgent) IsAvailable() bool {
	_, err := exec.LookPath(a.command)
	return err == nil
}

// Run executes the CLI agent
func (a *BaseAgent) Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error) {
	start := time.Now()

	// Build the full prompt
	fullPrompt := a.buildPrompt(req)

	// Set timeout
	timeout := a.timeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command
	args := append([]string{}, a.args...)
	if roleArgs := a.getRoleArgs(req.Role); len(roleArgs) > 0 {
		args = append(args, roleArgs...)
	}

	cmd := exec.CommandContext(ctx, a.command, args...)

	// Set working directory
	if req.WorkDir != "" {
		cmd.Dir = req.WorkDir
	}

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range a.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range req.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up stdin/stdout/stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Info("starting CLI agent",
		"name", a.name,
		"command", a.command,
		"args", args,
		"workdir", req.WorkDir,
	)

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting process: %w", err)
	}

	    // Write prompt to stdin
	    go func() {
	        defer stdin.Close()
	        if _, err := io.WriteString(stdin, fullPrompt); err != nil {
	            slog.Warn("failed to write to stdin", "error", err)
	        }
	    }()
		// Wait for completion
	err = cmd.Wait()
	duration := time.Since(start)

	output := &AgentOutput{
		Content:  stdout.String(),
		ExitCode: cmd.ProcessState.ExitCode(),
		Duration: duration,
	}

	if err != nil {
		output.ErrorMessage = fmt.Sprintf("process error: %v\nstderr: %s", err, stderr.String())
		slog.Warn("CLI agent error",
			"name", a.name,
			"error", err,
			"stderr", stderr.String(),
			"duration", duration,
		)
	} else {
		slog.Info("CLI agent completed",
			"name", a.name,
			"duration", duration,
			"output_length", len(output.Content),
		)
	}

	return output, nil
}

// buildPrompt constructs the full prompt with files and context
func (a *BaseAgent) buildPrompt(req *AgentRequest) string {
	var sb strings.Builder

	// Add system prompt from role
	if role, ok := a.roles[req.Role]; ok && role.SystemPrompt != "" {
		sb.WriteString(role.SystemPrompt)
		sb.WriteString("\n\n")
	} else if req.SystemPrompt != "" {
		sb.WriteString(req.SystemPrompt)
		sb.WriteString("\n\n")
	}

	// Add file contents
	if len(req.Files) > 0 {
		sb.WriteString("## Files\n\n")
		for _, path := range req.Files {
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", path, string(content)))
		}
	}

	// Add the main prompt
	sb.WriteString("## Request\n\n")
	sb.WriteString(req.Prompt)

	return sb.String()
}

func (a *BaseAgent) getRoleArgs(role string) []string {
	if r, ok := a.roles[role]; ok {
		return r.Args
	}
	return nil
}

func loadPromptFile(path string) string {
	if path == "" {
		return ""
	}

	// Try relative to prompts directory
	fullPath := filepath.Join("prompts", path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		// Try absolute path
		content, err = os.ReadFile(path)
		if err != nil {
			return ""
		}
	}
	return string(content)
}
