package main

import (
	"crypto/rand"
	"encoding/hex"
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
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
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
	sess := &Session{
		ID:       id,
		Messages: nil,
	}
	s.sessions[id] = sess
	return sess, true
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
