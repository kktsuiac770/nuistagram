package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	sessionMaxAge     = 24 * time.Hour
	sessionInactivity = 1 * time.Hour
	cleanupInterval   = 10 * time.Minute
	csrfTokenMaxAge   = 30 * time.Minute
)

type sessionData struct {
	userID    int64
	createdAt time.Time
	lastSeen  time.Time
	csrfToken string
}

type csrfData struct {
	token     string
	expiresAt time.Time
}

var (
	sessions   = make(map[string]sessionData)
	csrfTokens = make(map[string]csrfData)
	mu         sync.RWMutex
)

func init() {
	go cleanupExpiredSessions()
}

func cleanupExpiredSessions() {
	ticker := time.NewTicker(cleanupInterval)
	for range ticker.C {
		mu.Lock()
		now := time.Now()
		for token, data := range sessions {
			if now.Sub(data.lastSeen) > sessionInactivity || now.Sub(data.createdAt) > sessionMaxAge {
				delete(sessions, token)
			}
		}
		for token, data := range csrfTokens {
			if now.After(data.expiresAt) {
				delete(csrfTokens, token)
			}
		}
		mu.Unlock()
	}
}

func CreateSession(userID int64) string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	token := hex.EncodeToString(bytes)

	csrfBytes := make([]byte, 32)
	rand.Read(csrfBytes)
	csrfToken := hex.EncodeToString(csrfBytes)

	now := time.Now()
	mu.Lock()
	sessions[token] = sessionData{
		userID:    userID,
		createdAt: now,
		lastSeen:  now,
		csrfToken: csrfToken,
	}
	mu.Unlock()

	return token
}

func GetUserIDFromSession(token string) int64 {
	mu.Lock()
	defer mu.Unlock()

	if data, ok := sessions[token]; ok {
		now := time.Now()
		if now.Sub(data.lastSeen) > sessionInactivity || now.Sub(data.createdAt) > sessionMaxAge {
			delete(sessions, token)
			return 0
		}
		data.lastSeen = now
		sessions[token] = data
		return data.userID
	}
	return 0
}

func GetCSRFTokenFromSession(token string) string {
	mu.RLock()
	defer mu.RUnlock()

	if data, ok := sessions[token]; ok {
		return data.csrfToken
	}
	return ""
}

func DeleteSession(token string) {
	mu.Lock()
	delete(sessions, token)
	delete(csrfTokens, token)
	mu.Unlock()
}

func GenerateCSRFToken(sessionToken string) string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	token := hex.EncodeToString(bytes)

	mu.Lock()
	csrfTokens[token] = csrfData{
		token:     token,
		expiresAt: time.Now().Add(csrfTokenMaxAge),
	}
	if data, ok := sessions[sessionToken]; ok {
		data.csrfToken = token
		sessions[sessionToken] = data
	}
	mu.Unlock()

	return token
}

func ValidateCSRFToken(sessionToken, csrfToken string) bool {
	mu.RLock()
	defer mu.RUnlock()

	if data, ok := sessions[sessionToken]; ok {
		return data.csrfToken == csrfToken
	}
	return false
}

func IsSessionValid(token string) bool {
	mu.RLock()
	defer mu.RUnlock()

	data, ok := sessions[token]
	if !ok {
		return false
	}
	now := time.Now()
	return now.Sub(data.lastSeen) <= sessionInactivity && now.Sub(data.createdAt) <= sessionMaxAge
}
