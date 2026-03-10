package jwt

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the set of custom claims embedded in every access token.
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"sub"`
	jwt.RegisteredClaims
}

type refreshEntry struct {
	userID    int64
	expiresAt time.Time
}

// Manager handles JWT signing/validation and refresh token lifecycle.
type Manager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration

	mu      sync.RWMutex
	refresh map[string]refreshEntry
}

// NewManager constructs a Manager and starts the background cleanup goroutine.
// Returns an error if secret is empty.
func NewManager(secret string, accessTTL, refreshTTL time.Duration) (*Manager, error) {
	if secret == "" {
		return nil, errors.New("jwt: secret must not be empty")
	}
	m := &Manager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		refresh:         make(map[string]refreshEntry),
	}
	go m.cleanupLoop()
	return m, nil
}

// GenerateAccessToken signs a new JWT for the given user.
func (m *Manager) GenerateAccessToken(userID int64, username string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// ValidateAccessToken parses and validates a JWT, returning its claims.
// Returns an error for expired, malformed, or wrongly-signed tokens.
func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("jwt: unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("jwt: invalid token")
	}
	return claims, nil
}

// GenerateRefreshToken creates a cryptographically random 32-byte hex token,
// stores it mapped to userID, and returns the token.
func (m *Manager) GenerateRefreshToken(userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	m.mu.Lock()
	m.refresh[token] = refreshEntry{
		userID:    userID,
		expiresAt: time.Now().Add(m.refreshTokenTTL),
	}
	m.mu.Unlock()

	return token, nil
}

// ConsumeRefreshToken validates the token, deletes it (one-time-use rotation),
// and returns the userID. Returns an error if expired or unknown.
func (m *Manager) ConsumeRefreshToken(token string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.refresh[token]
	if !ok {
		return 0, errors.New("jwt: refresh token not found")
	}
	if time.Now().After(entry.expiresAt) {
		delete(m.refresh, token)
		return 0, errors.New("jwt: refresh token expired")
	}
	delete(m.refresh, token)
	return entry.userID, nil
}

// RevokeRefreshToken removes the token from the store (used on logout).
func (m *Manager) RevokeRefreshToken(token string) {
	m.mu.Lock()
	delete(m.refresh, token)
	m.mu.Unlock()
}

// AccessTokenTTL returns the configured access token lifetime.
func (m *Manager) AccessTokenTTL() time.Duration {
	return m.accessTokenTTL
}

// cleanupLoop deletes expired refresh tokens every 10 minutes.
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		m.mu.Lock()
		for token, entry := range m.refresh {
			if now.After(entry.expiresAt) {
				delete(m.refresh, token)
			}
		}
		m.mu.Unlock()
	}
}
