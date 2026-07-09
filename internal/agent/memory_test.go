package agent

import (
	"testing"

	"krillin-ai/internal/providers/llm"
)

func TestTerminologyMemory_AddAndGet(t *testing.T) {
	mem := NewTerminologyMemory()

	term := llm.Term{Term: "AI", Translation: "人工智慧", Note: "技術術語"}
	mem.Add(term)

	found := mem.Get("AI")
	if found == nil {
		t.Fatal("expected to find term 'AI'")
	}
	if found.Translation != "人工智慧" {
		t.Errorf("expected translation '人工智慧', got '%s'", found.Translation)
	}
}

func TestTerminologyMemory_Get_NotFound(t *testing.T) {
	mem := NewTerminologyMemory()

	found := mem.Get("non-existent")
	if found != nil {
		t.Error("expected nil for non-existent term")
	}
}

func TestTerminologyMemory_List(t *testing.T) {
	mem := NewTerminologyMemory()

	mem.Add(llm.Term{Term: "AI", Translation: "人工智慧"})
	mem.Add(llm.Term{Term: "GPU", Translation: "顯示卡"})

	terms := mem.List()
	if len(terms) != 2 {
		t.Errorf("expected 2 terms, got %d", len(terms))
	}
}

func TestTerminologyMemory_Clear(t *testing.T) {
	mem := NewTerminologyMemory()

	mem.Add(llm.Term{Term: "AI", Translation: "人工智慧"})
	mem.Clear()

	if len(mem.List()) != 0 {
		t.Error("expected empty list after Clear")
	}
}

func TestTerminologyMemory_ApplyToText(t *testing.T) {
	mem := NewTerminologyMemory()

	mem.Add(llm.Term{Term: "AI", Translation: "人工智慧"})
	mem.Add(llm.Term{Term: "GPU", Translation: "顯示卡"})

	text := "AI and GPU are important"
	result := mem.ApplyToText(text)

	if result != "人工智慧 and 顯示卡 are important" {
		t.Errorf("expected '人工智慧 and 顯示卡 are important', got '%s'", result)
	}
}

func TestTerminologyMemory_ApplyToText_NoMatch(t *testing.T) {
	mem := NewTerminologyMemory()

	text := "Hello world"
	result := mem.ApplyToText(text)

	if result != text {
		t.Errorf("expected unchanged text, got '%s'", result)
	}
}

func TestConversationMemory_Add(t *testing.T) {
	mem := NewConversationMemory()

	mem.Add("user", "Hello")
	mem.Add("assistant", "Hi there")

	if len(mem.messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(mem.messages))
	}
}

func TestConversationMemory_GetHistory(t *testing.T) {
	mem := NewConversationMemory()

	mem.Add("user", "Hello")
	mem.Add("assistant", "Hi there")
	mem.Add("user", "How are you?")

	history := mem.GetHistory(10)
	if len(history) != 3 {
		t.Errorf("expected 3 messages, got %d", len(history))
	}

	history = mem.GetHistory(2)
	if len(history) != 2 {
		t.Errorf("expected 2 messages with limit, got %d", len(history))
	}
}

func TestConversationMemory_Clear(t *testing.T) {
	mem := NewConversationMemory()

	mem.Add("user", "Hello")
	mem.Clear()

	if len(mem.messages) != 0 {
		t.Errorf("expected 0 messages after Clear, got %d", len(mem.messages))
	}
}

func TestConversationMemory_Messages(t *testing.T) {
	mem := NewConversationMemory()

	mem.Add("user", "Hello")
	mem.Add("assistant", "Hi")

	msgs := mem.Messages()
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected first role 'user', got '%s'", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("expected second role 'assistant', got '%s'", msgs[1].Role)
	}
}

func TestAgentMemory_All(t *testing.T) {
	mem := NewAgentMemory()

	mem.Terminology.Add(llm.Term{Term: "AI", Translation: "人工智慧"})
	mem.Conversation.Add("user", "Hello")

	if mem.Terminology.Get("AI") == nil {
		t.Error("expected to find AI term")
	}
	if len(mem.Conversation.Messages()) != 1 {
		t.Error("expected 1 conversation message")
	}
}

func TestTerminologyMemory_Delete(t *testing.T) {
	mem := NewTerminologyMemory()

	mem.Add(llm.Term{Term: "AI", Translation: "人工智慧"})
	mem.Delete("AI")

	if mem.Get("AI") != nil {
		t.Error("expected term to be deleted")
	}
}

func TestTerminologyMemory_Update(t *testing.T) {
	mem := NewTerminologyMemory()

	mem.Add(llm.Term{Term: "AI", Translation: "人工智慧"})
	mem.Update("AI", llm.Term{Term: "AI", Translation: "AI系統", Note: "updated"})

	term := mem.Get("AI")
	if term.Translation != "AI系統" {
		t.Errorf("expected translation 'AI系統', got '%s'", term.Translation)
	}
	if term.Note != "updated" {
		t.Errorf("expected note 'updated', got '%s'", term.Note)
	}
}
