package jwt

import (
	"testing"
	"time"

	"nuistagram/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := NewManager(config.JWTConfig{
		Secret:          "test-secret-key-for-unit-tests-32b",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})
	require.NoError(t, err)
	return m
}

func TestNewManager_EmptySecret(t *testing.T) {
	_, err := NewManager(config.JWTConfig{})
	assert.Error(t, err)
}

func TestNewManager_Success(t *testing.T) {
	m := newTestManager(t)
	assert.NotNil(t, m)
	assert.Equal(t, 15*time.Minute, m.AccessTokenTTL())
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	m := newTestManager(t)

	token, err := m.GenerateAccessToken(42, "alice")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := m.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "alice", claims.Username)
}

func TestValidateAccessToken_Expired(t *testing.T) {
	m, err := NewManager(config.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  -1 * time.Second,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})
	require.NoError(t, err)

	token, err := m.GenerateAccessToken(1, "user")
	require.NoError(t, err)

	_, err = m.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestValidateAccessToken_Malformed(t *testing.T) {
	m := newTestManager(t)
	_, err := m.ValidateAccessToken("not.a.valid.token")
	assert.Error(t, err)
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	m1, _ := NewManager(config.JWTConfig{
		Secret:          "secret-one-32-bytes-padded-12345",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})
	m2, _ := NewManager(config.JWTConfig{
		Secret:          "secret-two-32-bytes-padded-12345",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})

	token, err := m1.GenerateAccessToken(1, "user")
	require.NoError(t, err)

	_, err = m2.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestRefreshToken_ConsumeSuccess(t *testing.T) {
	m := newTestManager(t)

	token, err := m.GenerateRefreshToken(99)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	userID, err := m.ConsumeRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, int64(99), userID)
}

func TestRefreshToken_OneTimeUse(t *testing.T) {
	m := newTestManager(t)

	token, _ := m.GenerateRefreshToken(1)
	_, err := m.ConsumeRefreshToken(token)
	require.NoError(t, err)

	_, err = m.ConsumeRefreshToken(token)
	assert.Error(t, err)
}

func TestRefreshToken_NotFound(t *testing.T) {
	m := newTestManager(t)
	_, err := m.ConsumeRefreshToken("nonexistent-token")
	assert.Error(t, err)
}

func TestRefreshToken_Expired(t *testing.T) {
	m, err := NewManager(config.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: -1 * time.Second,
	})
	require.NoError(t, err)

	token, err := m.GenerateRefreshToken(1)
	require.NoError(t, err)

	_, err = m.ConsumeRefreshToken(token)
	assert.Error(t, err)
}

func TestRevokeRefreshToken(t *testing.T) {
	m := newTestManager(t)

	token, _ := m.GenerateRefreshToken(1)
	m.RevokeRefreshToken(token)

	_, err := m.ConsumeRefreshToken(token)
	assert.Error(t, err)
}

func TestRevokeRefreshToken_Idempotent(t *testing.T) {
	m := newTestManager(t)
	// Revoking a non-existent token should not panic
	m.RevokeRefreshToken("does-not-exist")
}

func TestRefreshTokenUniqueness(t *testing.T) {
	m := newTestManager(t)

	t1, err := m.GenerateRefreshToken(1)
	require.NoError(t, err)
	t2, err := m.GenerateRefreshToken(1)
	require.NoError(t, err)

	assert.NotEqual(t, t1, t2)
}

func TestAccessTokenTTL(t *testing.T) {
	m, _ := NewManager(config.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  5 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})
	assert.Equal(t, 5*time.Minute, m.AccessTokenTTL())
}
