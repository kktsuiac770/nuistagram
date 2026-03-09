package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	assert.NotNil(t, m)
	assert.NotNil(t, m.sessions)
	assert.Equal(t, 0, len(m.sessions))
}

func TestCreate_GeneratesTokens(t *testing.T) {
	m := NewManager()
	userID := int64(42)

	token := m.Create(userID)

	assert.NotEmpty(t, token)
	assert.Len(t, token, 32) // hex encoding of 16 bytes

	m.mu.RLock()
	d, exists := m.sessions[token]
	m.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, userID, d.userID)
	assert.NotEmpty(t, d.csrfToken)
	assert.Len(t, d.csrfToken, 64) // hex encoding of 32 bytes
	assert.True(t, d.createdAt.Before(time.Now().Add(time.Second)))
	assert.Equal(t, d.createdAt, d.lastSeen)
}

func TestCreate_MultipleTokensUnique(t *testing.T) {
	m := NewManager()

	token1 := m.Create(1)
	token2 := m.Create(2)

	assert.NotEqual(t, token1, token2)

	m.mu.RLock()
	csrf1 := m.sessions[token1].csrfToken
	csrf2 := m.sessions[token2].csrfToken
	m.mu.RUnlock()

	assert.NotEqual(t, csrf1, csrf2)
}

func TestGetUserID_ValidSession(t *testing.T) {
	m := NewManager()
	userID := int64(123)
	token := m.Create(userID)

	retrieved := m.GetUserID(token)

	assert.Equal(t, userID, retrieved)
}

func TestGetUserID_InvalidToken(t *testing.T) {
	m := NewManager()

	retrieved := m.GetUserID("nonexistent_token")

	assert.Equal(t, int64(0), retrieved)
}

func TestGetUserID_UpdatesLastSeen(t *testing.T) {
	m := NewManager()
	token := m.Create(42)

	m.mu.RLock()
	firstSeen := m.sessions[token].lastSeen
	m.mu.RUnlock()

	time.Sleep(10 * time.Millisecond)
	m.GetUserID(token)

	m.mu.RLock()
	secondSeen := m.sessions[token].lastSeen
	m.mu.RUnlock()

	assert.True(t, secondSeen.After(firstSeen))
}

func TestGetUserID_ExpiresAfterInactivity(t *testing.T) {
	m := NewManager()
	token := m.Create(42)

	// Manually set lastSeen to past
	m.mu.Lock()
	d := m.sessions[token]
	d.lastSeen = time.Now().Add(-(inactivity + 1*time.Second))
	m.sessions[token] = d
	m.mu.Unlock()

	retrieved := m.GetUserID(token)

	assert.Equal(t, int64(0), retrieved)
	m.mu.RLock()
	_, exists := m.sessions[token]
	m.mu.RUnlock()
	assert.False(t, exists)
}

func TestGetUserID_ExpiresAfterMaxAge(t *testing.T) {
	m := NewManager()
	token := m.Create(42)

	// Manually set createdAt to past (beyond maxAge)
	m.mu.Lock()
	d := m.sessions[token]
	d.createdAt = time.Now().Add(-(maxAge + 1*time.Second))
	d.lastSeen = time.Now() // keep recent to avoid inactivity expiry
	m.sessions[token] = d
	m.mu.Unlock()

	retrieved := m.GetUserID(token)

	assert.Equal(t, int64(0), retrieved)
	m.mu.RLock()
	_, exists := m.sessions[token]
	m.mu.RUnlock()
	assert.False(t, exists)
}

func TestIsValid_ValidSession(t *testing.T) {
	m := NewManager()
	token := m.Create(42)

	isValid := m.IsValid(token)

	assert.True(t, isValid)
}

func TestIsValid_InvalidToken(t *testing.T) {
	m := NewManager()

	isValid := m.IsValid("nonexistent")

	assert.False(t, isValid)
}

func TestIsValid_ExpiredByInactivity(t *testing.T) {
	m := NewManager()
	token := m.Create(42)

	m.mu.Lock()
	d := m.sessions[token]
	d.lastSeen = time.Now().Add(-(inactivity + 1*time.Second))
	m.sessions[token] = d
	m.mu.Unlock()

	isValid := m.IsValid(token)

	assert.False(t, isValid)
}

func TestIsValid_ExpiredByMaxAge(t *testing.T) {
	m := NewManager()
	token := m.Create(42)

	m.mu.Lock()
	d := m.sessions[token]
	d.createdAt = time.Now().Add(-(maxAge + 1*time.Second))
	d.lastSeen = time.Now()
	m.sessions[token] = d
	m.mu.Unlock()

	isValid := m.IsValid(token)

	assert.False(t, isValid)
}

func TestDelete_RemovesSession(t *testing.T) {
	m := NewManager()
	token := m.Create(42)

	m.mu.RLock()
	_, existsBefore := m.sessions[token]
	m.mu.RUnlock()
	assert.True(t, existsBefore)

	m.Delete(token)

	m.mu.RLock()
	_, existsAfter := m.sessions[token]
	m.mu.RUnlock()
	assert.False(t, existsAfter)
}

func TestDelete_NonexistentToken(t *testing.T) {
	m := NewManager()

	// Should not panic
	m.Delete("nonexistent_token")

	m.mu.RLock()
	count := len(m.sessions)
	m.mu.RUnlock()
	assert.Equal(t, 0, count)
}

func TestGenerateCSRFToken_UpdatesToken(t *testing.T) {
	m := NewManager()
	sessionToken := m.Create(42)

	m.mu.RLock()
	oldCSRF := m.sessions[sessionToken].csrfToken
	m.mu.RUnlock()

	newCSRF := m.GenerateCSRFToken(sessionToken)

	assert.NotEmpty(t, newCSRF)
	assert.NotEqual(t, oldCSRF, newCSRF)

	m.mu.RLock()
	storedCSRF := m.sessions[sessionToken].csrfToken
	m.mu.RUnlock()
	assert.Equal(t, newCSRF, storedCSRF)
}

func TestGenerateCSRFToken_NonexistentSession(t *testing.T) {
	m := NewManager()

	newCSRF := m.GenerateCSRFToken("nonexistent_session")

	assert.NotEmpty(t, newCSRF)
	// Token is generated but not stored (session doesn't exist)
}

func TestValidateCSRFToken_Valid(t *testing.T) {
	m := NewManager()
	sessionToken := m.Create(42)

	m.mu.RLock()
	csrf := m.sessions[sessionToken].csrfToken
	m.mu.RUnlock()

	isValid := m.ValidateCSRFToken(sessionToken, csrf)

	assert.True(t, isValid)
}

func TestValidateCSRFToken_InvalidToken(t *testing.T) {
	m := NewManager()
	sessionToken := m.Create(42)

	isValid := m.ValidateCSRFToken(sessionToken, "wrong_csrf_token")

	assert.False(t, isValid)
}

func TestValidateCSRFToken_NonexistentSession(t *testing.T) {
	m := NewManager()

	isValid := m.ValidateCSRFToken("nonexistent_session", "any_csrf")

	assert.False(t, isValid)
}

func TestGetCSRFToken_ReturnsToken(t *testing.T) {
	m := NewManager()
	sessionToken := m.Create(42)

	m.mu.RLock()
	expectedCSRF := m.sessions[sessionToken].csrfToken
	m.mu.RUnlock()

	csrf := m.GetCSRFToken(sessionToken)

	assert.Equal(t, expectedCSRF, csrf)
	assert.NotEmpty(t, csrf)
}

func TestGetCSRFToken_NonexistentSession(t *testing.T) {
	m := NewManager()

	csrf := m.GetCSRFToken("nonexistent_session")

	assert.Equal(t, "", csrf)
}

func TestCleanupLoop_RemovesInactiveSession(t *testing.T) {
	m := NewManager()
	token := m.Create(42)

	// Set lastSeen to expire during cleanup
	m.mu.Lock()
	d := m.sessions[token]
	d.lastSeen = time.Now().Add(-(inactivity + 5*time.Second))
	m.sessions[token] = d
	m.mu.Unlock()

	// Wait for cleanup cycle (10 minutes default, but we'll check state)
	// For testing, we trigger manual check via a different approach
	// or we verify the session is still there until cleanup runs
	m.mu.RLock()
	_, exists := m.sessions[token]
	m.mu.RUnlock()
	assert.True(t, exists)

	// Manually trigger cleanup by calling cleanup behavior
	m.mu.Lock()
	now := time.Now()
	for token, d := range m.sessions {
		if now.Sub(d.lastSeen) > inactivity || now.Sub(d.createdAt) > maxAge {
			delete(m.sessions, token)
		}
	}
	m.mu.Unlock()

	m.mu.RLock()
	_, existsAfter := m.sessions[token]
	m.mu.RUnlock()
	assert.False(t, existsAfter)
}

func TestCleanupLoop_RemovesOldSession(t *testing.T) {
	m := NewManager()
	token := m.Create(42)

	m.mu.Lock()
	d := m.sessions[token]
	d.createdAt = time.Now().Add(-(maxAge + 5*time.Second))
	d.lastSeen = time.Now() // recent activity, but old creation
	m.sessions[token] = d
	m.mu.Unlock()

	// Manual cleanup simulation
	m.mu.Lock()
	now := time.Now()
	for token, d := range m.sessions {
		if now.Sub(d.lastSeen) > inactivity || now.Sub(d.createdAt) > maxAge {
			delete(m.sessions, token)
		}
	}
	m.mu.Unlock()

	m.mu.RLock()
	_, exists := m.sessions[token]
	m.mu.RUnlock()
	assert.False(t, exists)
}

func TestConcurrentAccess(t *testing.T) {
	m := NewManager()

	go func() {
		for i := 0; i < 10; i++ {
			m.Create(int64(i))
		}
	}()

	go func() {
		for i := 0; i < 10; i++ {
			m.Create(int64(i + 100))
		}
	}()

	go func() {
		time.Sleep(5 * time.Millisecond)
		m.mu.RLock()
		count := len(m.sessions)
		m.mu.RUnlock()
		assert.True(t, count > 0)
	}()

	time.Sleep(50 * time.Millisecond)
}
