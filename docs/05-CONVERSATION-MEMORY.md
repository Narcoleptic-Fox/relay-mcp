# 05 - Conversation Memory

## Overview

The conversation memory system enables multi-turn conversations and cross-tool continuation. This is a key feature that allows the AI to maintain context across multiple tool invocations.

## Key Features

1. **Thread-Based Conversations**: Each conversation has a unique thread ID
2. **Cross-Tool Continuation**: A thread started with `chat` can continue with `debug`, `codereview`, etc.
3. **TTL-Based Expiration**: Threads expire after a configurable time (default 3 hours)
4. **Turn Limits**: Prevents runaway conversations (default 50 turns)
5. **File Deduplication**: Avoids sending the same file content multiple times

## Conversation Memory (internal/memory/conversation.go)

```go
package memory

import (
    "context"
    "log/slog"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/yourorg/pal-mcp/internal/types"
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
```

## Thread Context (internal/memory/thread.go)

```go
package memory

import (
    "strings"
    "time"

    "github.com/yourorg/pal-mcp/internal/types"
)

// ThreadBuilder helps build thread context for AI prompts
type ThreadBuilder struct {
    thread     *types.ThreadContext
    maxTokens  int
    tokenCount int
}

// NewThreadBuilder creates a new thread builder
func NewThreadBuilder(thread *types.ThreadContext, maxTokens int) *ThreadBuilder {
    return &ThreadBuilder{
        thread:    thread,
        maxTokens: maxTokens,
    }
}

// BuildConversationHistory builds the conversation history for the AI
func (b *ThreadBuilder) BuildConversationHistory() []types.ConversationTurn {
    if len(b.thread.Turns) == 0 {
        return nil
    }

    // Strategy: Include newest turns first, up to token budget
    var selected []types.ConversationTurn

    for i := len(b.thread.Turns) - 1; i >= 0; i-- {
        turn := b.thread.Turns[i]
        turnTokens := b.estimateTokens(turn.Content)

        if b.tokenCount+turnTokens > b.maxTokens {
            break
        }

        b.tokenCount += turnTokens
        // Prepend to maintain chronological order
        selected = append([]types.ConversationTurn{turn}, selected...)
    }

    return selected
}

// BuildContextSummary creates a summary of the conversation
func (b *ThreadBuilder) BuildContextSummary() string {
    var sb strings.Builder

    sb.WriteString("## Conversation Context\n\n")
    sb.WriteString("**Thread ID:** " + b.thread.ThreadID + "\n")
    sb.WriteString("**Started:** " + b.thread.CreatedAt.Format(time.RFC3339) + "\n")
    sb.WriteString("**Tool:** " + b.thread.ToolName + "\n")
    sb.WriteString("**Turns:** " + string(rune(len(b.thread.Turns))) + "\n\n")

    // Collect unique tools used
    tools := make(map[string]bool)
    for _, turn := range b.thread.Turns {
        if turn.ToolName != "" {
            tools[turn.ToolName] = true
        }
    }

    if len(tools) > 1 {
        sb.WriteString("**Tools Used:** ")
        first := true
        for tool := range tools {
            if !first {
                sb.WriteString(", ")
            }
            sb.WriteString(tool)
            first = false
        }
        sb.WriteString("\n")
    }

    return sb.String()
}

// GetRecentFiles returns files from recent turns
func (b *ThreadBuilder) GetRecentFiles(maxFiles int) []string {
    seen := make(map[string]bool)
    var files []string

    for i := len(b.thread.Turns) - 1; i >= 0 && len(files) < maxFiles; i-- {
        for _, f := range b.thread.Turns[i].Files {
            if !seen[f] {
                seen[f] = true
                files = append(files, f)
            }
        }
    }

    return files
}

func (b *ThreadBuilder) estimateTokens(content string) int {
    // Rough estimate: 4 characters per token
    return len(content) / 4
}
```

## Cross-Tool Continuation

The key feature is that a conversation can flow between tools:

```
User → chat (thread-123)
     → Response
User → debug (continuation_id: thread-123)
     → Debug uses chat's context
User → codereview (continuation_id: thread-123)
     → Review uses chat + debug context
```

### How It Works

```go
// internal/tools/simple/chat.go

func (t *ChatTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
    continuationID := parser.GetString("continuation_id")

    // Get existing thread or create new one
    thread, isExisting := t.GetOrCreateThread(continuationID)

    if isExisting {
        // Get conversation history from previous tools
        history := t.memory.GetHistory(thread.ThreadID)

        // Include in AI request
        resp, err := t.GenerateContent(ctx, provider, &providers.GenerateRequest{
            Prompt:              fullPrompt,
            ConversationHistory: history,  // ← Previous context
            // ...
        })
    }

    // Always return the thread ID for continuation
    result := fmt.Sprintf("%s\n\n---\ncontinuation_id: %s", resp.Content, thread.ThreadID)
}
```

## File Deduplication

When files are referenced multiple times, we deduplicate:

```go
// internal/memory/dedup.go
package memory

import (
    "crypto/sha256"
    "encoding/hex"
)

// FileDeduplicator tracks files across conversation turns
type FileDeduplicator struct {
    seen map[string]string // hash -> path
}

// NewFileDeduplicator creates a new deduplicator
func NewFileDeduplicator() *FileDeduplicator {
    return &FileDeduplicator{
        seen: make(map[string]string),
    }
}

// ShouldInclude returns true if this file should be included
func (d *FileDeduplicator) ShouldInclude(path, content string) bool {
    hash := d.hashContent(content)

    if existingPath, ok := d.seen[hash]; ok {
        // Already seen this exact content
        return existingPath != path // Different path = include
    }

    d.seen[hash] = path
    return true
}

// FilterFiles filters out duplicate files
func (d *FileDeduplicator) FilterFiles(files []FileWithContent) []FileWithContent {
    var result []FileWithContent

    for _, f := range files {
        if d.ShouldInclude(f.Path, f.Content) {
            result = append(result, f)
        }
    }

    return result
}

func (d *FileDeduplicator) hashContent(content string) string {
    h := sha256.Sum256([]byte(content))
    return hex.EncodeToString(h[:])
}

// FileWithContent holds a file and its content
type FileWithContent struct {
    Path    string
    Content string
}
```

## Error Types (internal/memory/errors.go)

```go
package memory

// ErrThreadNotFound indicates the thread doesn't exist
type ErrThreadNotFound struct {
    ThreadID string
}

func (e ErrThreadNotFound) Error() string {
    return "thread not found: " + e.ThreadID
}

// ErrThreadExpired indicates the thread has expired
type ErrThreadExpired struct {
    ThreadID string
}

func (e ErrThreadExpired) Error() string {
    return "thread expired: " + e.ThreadID
}
```

## Usage Example

```go
// Creating a new conversation
mem := memory.New(50, 3) // 50 turns max, 3 hour TTL
thread := mem.CreateThread("chat")

// Adding turns
mem.AddTurn(thread.ThreadID, types.ConversationTurn{
    Role:    "user",
    Content: "Help me debug this function",
    Files:   []string{"/path/to/file.go"},
})

mem.AddTurn(thread.ThreadID, types.ConversationTurn{
    Role:    "assistant",
    Content: "I see a potential issue on line 42...",
})

// Continuing in another tool
existingThread := mem.GetThread(thread.ThreadID)
if existingThread != nil {
    history := mem.GetHistory(thread.ThreadID)
    // Use history in new tool...
}
```

## Token Management

When building conversation history for the AI, we need to respect token limits:

```go
// internal/memory/tokens.go
package memory

import "github.com/yourorg/pal-mcp/internal/types"

const (
    DefaultMaxHistoryTokens = 50000  // Reserve 50k for history
    DefaultMaxFileTokens    = 100000 // Reserve 100k for files
)

// TokenBudget manages token allocation
type TokenBudget struct {
    TotalBudget    int
    HistoryBudget  int
    FileBudget     int
    ResponseBudget int
}

// NewTokenBudget creates a budget for a model
func NewTokenBudget(contextWindow int) *TokenBudget {
    // Reserve space for response
    responseBudget := min(contextWindow/4, 16000)
    remaining := contextWindow - responseBudget

    // Split remaining between history and files
    historyBudget := min(remaining/3, DefaultMaxHistoryTokens)
    fileBudget := remaining - historyBudget

    return &TokenBudget{
        TotalBudget:    contextWindow,
        HistoryBudget:  historyBudget,
        FileBudget:     fileBudget,
        ResponseBudget: responseBudget,
    }
}

// AllocateHistory returns turns that fit in budget
func (b *TokenBudget) AllocateHistory(turns []types.ConversationTurn) []types.ConversationTurn {
    var result []types.ConversationTurn
    tokens := 0

    // Start from newest, work backwards
    for i := len(turns) - 1; i >= 0; i-- {
        turnTokens := estimateTokens(turns[i].Content)
        if tokens+turnTokens > b.HistoryBudget {
            break
        }
        tokens += turnTokens
        result = append([]types.ConversationTurn{turns[i]}, result...)
    }

    return result
}

func estimateTokens(content string) int {
    return len(content) / 4
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

## Thread Lifecycle

```
┌─────────────────┐
│  CreateThread   │
│   (tool call)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Active Thread │◄──────┐
│  (accepting     │       │
│   turns)        │       │ AddTurn
└────────┬────────┘       │
         │                │
         ├────────────────┘
         │
         │ TTL expires OR
         │ context.Done
         ▼
┌─────────────────┐
│    Expired      │
│  (cleanup)      │
└─────────────────┘
```

## Testing

```go
// internal/memory/conversation_test.go
package memory

import (
    "context"
    "testing"
    "time"

    "github.com/yourorg/pal-mcp/internal/types"
)

func TestConversationMemory_CreateThread(t *testing.T) {
    mem := New(50, 3)

    thread := mem.CreateThread("chat")

    if thread.ThreadID == "" {
        t.Error("expected thread ID")
    }
    if thread.ToolName != "chat" {
        t.Errorf("expected tool 'chat', got %s", thread.ToolName)
    }
}

func TestConversationMemory_AddTurn(t *testing.T) {
    mem := New(50, 3)
    thread := mem.CreateThread("chat")

    err := mem.AddTurn(thread.ThreadID, types.ConversationTurn{
        Role:    "user",
        Content: "Hello",
    })

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    history := mem.GetHistory(thread.ThreadID)
    if len(history) != 1 {
        t.Errorf("expected 1 turn, got %d", len(history))
    }
}

func TestConversationMemory_MaxTurns(t *testing.T) {
    mem := New(3, 3) // Only 3 turns max
    thread := mem.CreateThread("chat")

    for i := 0; i < 5; i++ {
        mem.AddTurn(thread.ThreadID, types.ConversationTurn{
            Role:    "user",
            Content: "Turn " + string(rune('0'+i)),
        })
    }

    history := mem.GetHistory(thread.ThreadID)
    if len(history) != 3 {
        t.Errorf("expected 3 turns (max), got %d", len(history))
    }

    // Should have the last 3 turns
    if history[0].Content != "Turn 2" {
        t.Errorf("expected 'Turn 2', got %s", history[0].Content)
    }
}

func TestConversationMemory_CrossToolContinuation(t *testing.T) {
    mem := New(50, 3)

    // Start with chat
    thread := mem.CreateThread("chat")
    mem.AddTurn(thread.ThreadID, types.ConversationTurn{
        Role:     "user",
        Content:  "Initial question",
        ToolName: "chat",
    })

    // Continue with debug (same thread ID)
    retrieved := mem.GetThread(thread.ThreadID)
    if retrieved == nil {
        t.Fatal("expected to retrieve thread")
    }

    mem.AddTurn(thread.ThreadID, types.ConversationTurn{
        Role:     "user",
        Content:  "Debug request",
        ToolName: "debug",
    })

    history := mem.GetHistory(thread.ThreadID)
    if len(history) != 2 {
        t.Errorf("expected 2 turns, got %d", len(history))
    }
    if history[0].ToolName != "chat" || history[1].ToolName != "debug" {
        t.Error("tool names not preserved")
    }
}
```

## Next Steps

Continue to [06-CLINK.md](./06-CLINK.md) for the CLI linking system.
