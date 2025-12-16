package memory

import (
	"testing"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/types"
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
