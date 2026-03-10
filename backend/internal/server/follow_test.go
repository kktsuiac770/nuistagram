package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"nuistagram/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIFollowByUsername_Unauthorized(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/user/bob/follow", nil)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()

	srv.APIFollowByUsername(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIFollowByUsername_UserNotFound(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Users.On("GetByUsername", "nobody").Return(nil, sql.ErrNoRows)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/nobody/follow", nil)
	req.SetPathValue("username", "nobody")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIFollowByUsername(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIFollowByUsername_CannotFollowSelf(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Users.On("GetByUsername", "alice").Return(mockUser, nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/alice/follow", nil)
	req.SetPathValue("username", "alice")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIFollowByUsername(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIFollowByUsername_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	mockRepos.Follows.On("Follow", int64(1), int64(2)).Return(nil)
	mockRepos.Notifications.On("Create", int64(2), int64(1), "follow", int64(0), int64(0)).Return(int64(1), nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/bob/follow", nil)
	req.SetPathValue("username", "bob")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIFollowByUsername(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"])
	mockRepos.Follows.AssertExpectations(t)
	mockRepos.Notifications.AssertExpectations(t)
}

func TestAPIFollowByUsername_FollowRepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	mockRepos.Follows.On("Follow", int64(1), int64(2)).Return(errors.New("already following"))

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/bob/follow", nil)
	req.SetPathValue("username", "bob")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIFollowByUsername(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIUnfollowByUsername_Unauthorized(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/user/bob/unfollow", nil)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()

	srv.APIUnfollowByUsername(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIUnfollowByUsername_UserNotFound(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "ghost").Return(nil, sql.ErrNoRows)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/ghost/unfollow", nil)
	req.SetPathValue("username", "ghost")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIUnfollowByUsername(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIUnfollowByUsername_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	mockRepos.Follows.On("Unfollow", int64(1), int64(2)).Return(nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/bob/unfollow", nil)
	req.SetPathValue("username", "bob")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIUnfollowByUsername(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"])
}

func TestAPIUnfollowByUsername_RepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	mockRepos.Follows.On("Unfollow", int64(1), int64(2)).Return(errors.New("db error"))

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/user/bob/unfollow", nil)
	req.SetPathValue("username", "bob")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIUnfollowByUsername(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
