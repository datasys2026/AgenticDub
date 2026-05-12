package agent

import (
	"strings"
	"sync"
)

type TerminologyMemory struct {
	mu     sync.RWMutex
	terms  map[string]Term
}

func NewTerminologyMemory() *TerminologyMemory {
	return &TerminologyMemory{
		terms: make(map[string]Term),
	}
}

func (m *TerminologyMemory) Add(term Term) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terms[term.Term] = term
}

func (m *TerminologyMemory) Get(term string) *Term {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if t, ok := m.terms[term]; ok {
		return &t
	}
	return nil
}

func (m *TerminologyMemory) List() []Term {
	m.mu.RLock()
	defer m.mu.RUnlock()
	terms := make([]Term, 0, len(m.terms))
	for _, t := range m.terms {
		terms = append(terms, t)
	}
	return terms
}

func (m *TerminologyMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terms = make(map[string]Term)
}

func (m *TerminologyMemory) Delete(term string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.terms, term)
}

func (m *TerminologyMemory) Update(term string, newTerm Term) {
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
	messages []Message
}

func NewConversationMemory() *ConversationMemory {
	return &ConversationMemory{
		messages: make([]Message, 0),
	}
}

func (m *ConversationMemory) Add(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, Message{Role: role, Content: content})
}

func (m *ConversationMemory) GetHistory(limit int) []Message {
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
	m.messages = make([]Message, 0)
}

func (m *ConversationMemory) Messages() []Message {
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
