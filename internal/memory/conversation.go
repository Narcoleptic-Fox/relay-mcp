package memory

import (
	"context"
	"log/slog"
	"sync"
	"time"

    "github.com/google/uuid"
    "github.com/Narcoleptic-Fox/relay-mcp/internal/types"
)

// ConversationMemory manages conversation threads
type ConversationMemory struct {
	threads     map[string]*types.ThreadContext
	mu          sync.RWMutex
	maxTurns    int
	ttlHours    int
	cleanupDone chan struct{}
}

// New creates a new conversation memory
func New(maxTurns, ttlHours int) *ConversationMemory {
	return &ConversationMemory{
		threads:     make(map[string]*types.ThreadContext),
		maxTurns:    maxTurns,
		ttlHours:    ttlHours,
		cleanupDone: make(chan struct{}),
	}
}

// CreateThread creates a new conversation thread
func (m *ConversationMemory) CreateThread(toolName string) *types.ThreadContext {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	thread := &types.ThreadContext{
		ThreadID:      uuid.New().String(),
		CreatedAt:     now,
		LastUpdatedAt: now,
		ToolName:      toolName,
		Turns:         make([]types.ConversationTurn, 0),
	}

	m.threads[thread.ThreadID] = thread
	slog.Debug("created thread", "id", thread.ThreadID, "tool", toolName)

	return thread
}

// GetThread retrieves a thread by ID
func (m *ConversationMemory) GetThread(threadID string) *types.ThreadContext {
	m.mu.RLock()
	defer m.mu.RUnlock()

	thread, ok := m.threads[threadID]
	if !ok {
		return nil
	}

	// Check if expired
	if time.Since(thread.LastUpdatedAt) > time.Duration(m.ttlHours)*time.Hour {
		slog.Debug("thread expired", "id", threadID)
		return nil
	}

	return thread
}

// AddTurn adds a conversation turn to a thread
func (m *ConversationMemory) AddTurn(threadID string, turn types.ConversationTurn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	thread, ok := m.threads[threadID]
	if !ok {
		return ErrThreadNotFound{ThreadID: threadID}
	}

	// Set timestamp if not set
	if turn.Timestamp.IsZero() {
		turn.Timestamp = time.Now()
	}

	// Enforce max turns
	if len(thread.Turns) >= m.maxTurns {
		// Remove oldest turns (keep last maxTurns-1)
		thread.Turns = thread.Turns[len(thread.Turns)-m.maxTurns+1:]
		slog.Debug("trimmed thread", "id", threadID, "turns", len(thread.Turns))
	}

	thread.Turns = append(thread.Turns, turn)
	thread.LastUpdatedAt = time.Now()

	slog.Debug("added turn", "id", threadID, "role", turn.Role, "turns", len(thread.Turns))

	return nil
}

// GetHistory returns the conversation history for a thread
func (m *ConversationMemory) GetHistory(threadID string) []types.ConversationTurn {
	m.mu.RLock()
	defer m.mu.RUnlock()

	thread, ok := m.threads[threadID]
	if !ok {
		return nil
	}

	// Return a copy
	history := make([]types.ConversationTurn, len(thread.Turns))
	copy(history, thread.Turns)
	return history
}

// GetFileList returns all unique files referenced in the conversation
func (m *ConversationMemory) GetFileList(threadID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	thread, ok := m.threads[threadID]
	if !ok {
		return nil
	}

	// Collect unique files, newest first
	seen := make(map[string]bool)
	var files []string

	// Iterate in reverse (newest first)
	for i := len(thread.Turns) - 1; i >= 0; i-- {
		for _, f := range thread.Turns[i].Files {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}

	return files
}

// StartCleanup starts the background cleanup goroutine
func (m *ConversationMemory) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(m.cleanupDone)
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// cleanup removes expired threads
func (m *ConversationMemory) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	ttl := time.Duration(m.ttlHours) * time.Hour
	expired := 0

	for id, thread := range m.threads {
		if now.Sub(thread.LastUpdatedAt) > ttl {
			delete(m.threads, id)
			expired++
		}
	}

	if expired > 0 {
		slog.Info("cleaned up expired threads", "count", expired, "remaining", len(m.threads))
	}
}

// Stats returns memory statistics
func (m *ConversationMemory) Stats() MemoryStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalTurns := 0
	for _, thread := range m.threads {
		totalTurns += len(thread.Turns)
	}

	return MemoryStats{
		ThreadCount: len(m.threads),
		TotalTurns:  totalTurns,
	}
}

// MemoryStats holds memory statistics
type MemoryStats struct {
	ThreadCount int
	TotalTurns  int
}
