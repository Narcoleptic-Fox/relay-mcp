package memory

import (
	"testing"
	"time"

	"github.com/Narcoleptic-Fox/relay-mcp/internal/types"
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

        err := mem.AddTurn(thread.ThreadID, types.ConversationTurn{

            Role:    "user",

            Content: "Turn " + string(rune('0'+i)),

        })

        if err != nil {

            t.Fatalf("failed to add turn %d: %v", i, err)

        }

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

	    err := mem.AddTurn(thread.ThreadID, types.ConversationTurn{

	        Role:     "user",

	        Content:  "Initial question",

	        ToolName: "chat",

	    })

	    if err != nil {

	        t.Fatalf("failed to add first turn: %v", err)

	    }

	

	    // Continue with debug (same thread ID)

	    retrieved := mem.GetThread(thread.ThreadID)

	    if retrieved == nil {

	        t.Fatal("expected to retrieve thread")

	    }

	

	    err = mem.AddTurn(thread.ThreadID, types.ConversationTurn{

	        Role:     "user",

	        Content:  "Debug request",

	        ToolName: "debug",

	    })

	    if err != nil {

	        t.Fatalf("failed to add second turn: %v", err)

	    }

	

	    history := mem.GetHistory(thread.ThreadID)

	
	if len(history) != 2 {
		t.Errorf("expected 2 turns, got %d", len(history))
	}
	if history[0].ToolName != "chat" || history[1].ToolName != "debug" {
		t.Error("tool names not preserved")
	}
}

func TestConversationMemory_ExpirationRemovesFromMap(t *testing.T) {
	// Create memory with 0-hour TTL (immediate expiration)
	mem := New(50, 0)
	thread := mem.CreateThread("chat")
	threadID := thread.ThreadID

	// Verify thread was created
	stats := mem.Stats()
	if stats.ThreadCount != 1 {
		t.Fatalf("expected 1 thread, got %d", stats.ThreadCount)
	}

	// Wait a tiny bit to ensure time has passed
	time.Sleep(10 * time.Millisecond)

	// First GetThread should return nil (expired) AND delete from map
	result := mem.GetThread(threadID)
	if result != nil {
		t.Error("expected nil for expired thread")
	}

	// Verify thread was removed from map
	stats = mem.Stats()
	if stats.ThreadCount != 0 {
		t.Errorf("expected 0 threads after expiration cleanup, got %d", stats.ThreadCount)
	}
}

func TestConversationMemory_NonExpiredThreadPersists(t *testing.T) {
	// Create memory with 1-hour TTL
	mem := New(50, 1)
	thread := mem.CreateThread("chat")

	// Add a turn
	err := mem.AddTurn(thread.ThreadID, types.ConversationTurn{
		Role:    "user",
		Content: "Hello",
	})
	if err != nil {
		t.Fatalf("failed to add turn: %v", err)
	}

	// Thread should still be retrievable
	retrieved := mem.GetThread(thread.ThreadID)
	if retrieved == nil {
		t.Error("expected to retrieve non-expired thread")
	}

	// Stats should show 1 thread
	stats := mem.Stats()
	if stats.ThreadCount != 1 {
		t.Errorf("expected 1 thread, got %d", stats.ThreadCount)
	}
}
