package follows

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nuistagram/internal/auth"
	"nuistagram/internal/config"
	jwtpkg "nuistagram/internal/jwt"
	"nuistagram/internal/models"
	"nuistagram/internal/monitoring/metrics"
	"nuistagram/internal/repository/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newHandler() (*Handler, *mocks.MockRepositories, *jwtpkg.Manager) {
	metrics.Init()
	mockRepos := mocks.NewMockRepositories()
	jwtConfig := config.JWTConfig{
		Secret:          "test-secret-key-for-unit-tests-32b",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
	jwtMgr, _ := jwtpkg.NewManager(jwtConfig)
	authHandler := auth.New(mockRepos.Users, jwtMgr)
	h := New(authHandler, mockRepos.Users, mockRepos.Follows, mockRepos.Notifications)
	return h, mockRepos, jwtMgr
}

func TestAPIFollowByUsername_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("POST", "/api/user/bob/follow", nil)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()

	h.APIFollowByUsername(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIFollowByUsername_UserNotFound(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Users.On("GetByUsername", "nobody").Return(nil, sql.ErrNoRows)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/nobody/follow", nil)
	req.SetPathValue("username", "nobody")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIFollowByUsername(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIFollowByUsername_CannotFollowSelf(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Users.On("GetByUsername", "alice").Return(mockUser, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/alice/follow", nil)
	req.SetPathValue("username", "alice")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIFollowByUsername(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIFollowByUsername_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	mockRepos.Follows.On("Follow", int64(1), int64(2)).Return(nil)
	mockRepos.Notifications.On("Create", int64(2), int64(1), "follow", int64(0), int64(0)).Return(int64(1), nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/bob/follow", nil)
	req.SetPathValue("username", "bob")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIFollowByUsername(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"])
	mockRepos.Follows.AssertExpectations(t)
	mockRepos.Notifications.AssertExpectations(t)
}

func TestAPIFollowByUsername_FollowRepoError(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	mockRepos.Follows.On("Follow", int64(1), int64(2)).Return(errors.New("already following"))

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/bob/follow", nil)
	req.SetPathValue("username", "bob")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIFollowByUsername(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIUnfollowByUsername_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("POST", "/api/user/bob/unfollow", nil)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()

	h.APIUnfollowByUsername(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIUnfollowByUsername_UserNotFound(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "ghost").Return(nil, sql.ErrNoRows)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/ghost/unfollow", nil)
	req.SetPathValue("username", "ghost")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIUnfollowByUsername(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIUnfollowByUsername_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	mockRepos.Follows.On("Unfollow", int64(1), int64(2)).Return(nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/bob/unfollow", nil)
	req.SetPathValue("username", "bob")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIUnfollowByUsername(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"])
}

func TestAPIUnfollowByUsername_RepoError(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	mockRepos.Follows.On("Unfollow", int64(1), int64(2)).Return(errors.New("db error"))

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/bob/unfollow", nil)
	req.SetPathValue("username", "bob")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIUnfollowByUsername(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIGetFollowStatus_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("GET", "/api/user/bob/follow-status", nil)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()

	h.APIGetFollowStatus(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIGetFollowStatus_UserNotFound(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "ghost").Return(nil, sql.ErrNoRows)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/user/ghost/follow-status", nil)
	req.SetPathValue("username", "ghost")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIGetFollowStatus(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIGetFollowStatus_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	mockRepos.Follows.On("IsFollowing", int64(1), int64(2)).Return(true)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/user/bob/follow-status", nil)
	req.SetPathValue("username", "bob")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIGetFollowStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["is_following"])
}
