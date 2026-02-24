package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

type sessionData struct {
	userID int64
}

var (
	sessions = make(map[string]sessionData)
	mu       sync.RWMutex
)

func CreateSession(userID int64) string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	token := hex.EncodeToString(bytes)

	mu.Lock()
	sessions[token] = sessionData{userID: userID}
	mu.Unlock()

	return token
}

func GetUserIDFromSession(token string) int64 {
	mu.RLock()
	defer mu.RUnlock()

	if data, ok := sessions[token]; ok {
		return data.userID
	}
	return 0
}

func DeleteSession(token string) {
	mu.Lock()
	delete(sessions, token)
	mu.Unlock()
}
