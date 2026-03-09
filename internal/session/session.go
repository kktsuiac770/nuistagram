package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	maxAge     = 24 * time.Hour
	inactivity = 1 * time.Hour
	cleanup    = 10 * time.Minute
)

type data struct {
	userID    int64
	createdAt time.Time
	lastSeen  time.Time
	csrfToken string
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]data
}

func NewManager() *Manager {
	m := &Manager{
		sessions: make(map[string]data),
	}
	go m.cleanupLoop()
	return m
}

func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(cleanup)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for token, d := range m.sessions {
			if now.Sub(d.lastSeen) > inactivity || now.Sub(d.createdAt) > maxAge {
				delete(m.sessions, token)
			}
		}
		m.mu.Unlock()
	}
}

func (m *Manager) Create(userID int64) string {
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)

	csrfBytes := make([]byte, 32)
	rand.Read(csrfBytes)
	csrfToken := hex.EncodeToString(csrfBytes)

	now := time.Now()
	m.mu.Lock()
	m.sessions[token] = data{
		userID:    userID,
		createdAt: now,
		lastSeen:  now,
		csrfToken: csrfToken,
	}
	m.mu.Unlock()

	return token
}

func (m *Manager) GetUserID(token string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	d, ok := m.sessions[token]
	if !ok {
		return 0
	}
	now := time.Now()
	if now.Sub(d.lastSeen) > inactivity || now.Sub(d.createdAt) > maxAge {
		delete(m.sessions, token)
		return 0
	}
	d.lastSeen = now
	m.sessions[token] = d
	return d.userID
}

func (m *Manager) IsValid(token string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	d, ok := m.sessions[token]
	if !ok {
		return false
	}
	now := time.Now()
	return now.Sub(d.lastSeen) <= inactivity && now.Sub(d.createdAt) <= maxAge
}

func (m *Manager) Delete(token string) {
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
}

func (m *Manager) GenerateCSRFToken(sessionToken string) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)

	m.mu.Lock()
	if d, ok := m.sessions[sessionToken]; ok {
		d.csrfToken = token
		m.sessions[sessionToken] = d
	}
	m.mu.Unlock()

	return token
}

func (m *Manager) ValidateCSRFToken(sessionToken, csrfToken string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if d, ok := m.sessions[sessionToken]; ok {
		return d.csrfToken == csrfToken
	}
	return false
}

func (m *Manager) GetCSRFToken(sessionToken string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if d, ok := m.sessions[sessionToken]; ok {
		return d.csrfToken
	}
	return ""
}
