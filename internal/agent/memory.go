package agent

import (
	"strings"
	"sync"

	"krillin-ai/internal/providers/llm"
)

type TerminologyMemory struct {
	mu     sync.RWMutex
	terms  map[string]llm.Term
}

func NewTerminologyMemory() *TerminologyMemory {
	return &TerminologyMemory{
		terms: make(map[string]llm.Term),
	}
}

func (m *TerminologyMemory) Add(term llm.Term) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terms[term.Term] = term
}

func (m *TerminologyMemory) Get(term string) *llm.Term {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if t, ok := m.terms[term]; ok {
		return &t
	}
	return nil
}

func (m *TerminologyMemory) List() []llm.Term {
	m.mu.RLock()
	defer m.mu.RUnlock()
	terms := make([]llm.Term, 0, len(m.terms))
	for _, t := range m.terms {
		terms = append(terms, t)
	}
	return terms
}

func (m *TerminologyMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terms = make(map[string]llm.Term)
}

func (m *TerminologyMemory) Delete(term string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.terms, term)
}

func (m *TerminologyMemory) Update(term string, newTerm llm.Term) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terms[term] = newTerm
}

func (m *TerminologyMemory) ApplyToText(text string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := text
	for key, term := range m.terms {
		result = strings.ReplaceAll(result, key, term.Translation)
	}
	return result
}

type ConversationMemory struct {
	mu       sync.RWMutex
	messages []llm.Message
}

func NewConversationMemory() *ConversationMemory {
	return &ConversationMemory{
		messages: make([]llm.Message, 0),
	}
}

func (m *ConversationMemory) Add(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, llm.Message{Role: role, Content: content})
}

func (m *ConversationMemory) GetHistory(limit int) []llm.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > len(m.messages) {
		return m.messages
	}
	return m.messages[len(m.messages)-limit:]
}

func (m *ConversationMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]llm.Message, 0)
}

func (m *ConversationMemory) Messages() []llm.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.messages
}

type AgentMemory struct {
	Terminology   *TerminologyMemory
	Conversation  *ConversationMemory
}

func NewAgentMemory() *AgentMemory {
	return &AgentMemory{
		Terminology:  NewTerminologyMemory(),
		Conversation: NewConversationMemory(),
	}
}
