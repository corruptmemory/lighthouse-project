package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
)

type Session struct {
	ID       string
	Messages []anthropic.MessageParam
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	dir      string // base chats directory
}

func NewSessionStore(dir string) *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
		dir:      dir,
	}
}

func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}

func (s *SessionStore) GetOrCreate(id string) (*Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sess, ok := s.sessions[id]; ok {
		return sess, false
	}

	if id == "" {
		id = generateID()
	}

	// Try to restore from disk
	var messages []anthropic.MessageParam
	if s.dir != "" {
		restored, err := loadChatHistory(s.dir, id)
		if err != nil {
			log.Printf("restore session %s: %v", id, err)
		} else {
			messages = restored
		}
	}

	sess := &Session{
		ID:       id,
		Messages: messages,
	}
	s.sessions[id] = sess
	return sess, true
}

// SaveExchange persists a single chat exchange (the new messages from one
// agent loop iteration) to a numbered markdown file on disk.
func (s *SessionStore) SaveExchange(sessionID, prompt string, newMessages []anthropic.MessageParam) error {
	if s.dir == "" {
		return nil
	}
	return writeChatExchange(s.dir, sessionID, prompt, newMessages)
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
