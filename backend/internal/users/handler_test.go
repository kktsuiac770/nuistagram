package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"nuistagram/internal/auth"
	"nuistagram/internal/config"
	jwtpkg "nuistagram/internal/jwt"
	"nuistagram/internal/models"
	"nuistagram/internal/monitoring/metrics"
	"nuistagram/internal/ratelimit"
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
	h := New(authHandler, mockRepos.Users, mockRepos.Photos, nil, &config.Config{}, ratelimit.New())
	return h, mockRepos, jwtMgr
}

// --- APIGetMe ---

func TestAPIGetMe_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("GET", "/api/me", nil)
	w := httptest.NewRecorder()

	h.APIGetMe(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIGetMe_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	fullAlice := &models.User{
		ID: 1, Username: "alice", Bio: "hello", Avatar: "a.jpg",
		PhotoCount: 10, FollowerCount: 5, FollowingCount: 3,
	}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByIDWithCounts", int64(1), int64(1)).Return(fullAlice, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIGetMe(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "alice", body["username"])
	assert.Equal(t, "hello", body["bio"])
	assert.Equal(t, float64(10), body["photo_count"])
}

func TestAPIGetMe_RepoError(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByIDWithCounts", int64(1), int64(1)).Return(nil, errors.New("db error"))

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIGetMe(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- APIGetUser ---

func TestAPIGetUser_NotFound(t *testing.T) {
	h, mockRepos, _ := newHandler()
	mockRepos.Users.On("GetByUsernameWithCounts", "nobody", int64(0)).Return(nil, errors.New("not found"))

	req := httptest.NewRequest("GET", "/api/user/nobody", nil)
	req.SetPathValue("username", "nobody")
	w := httptest.NewRecorder()

	h.APIGetUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIGetUser_Success(t *testing.T) {
	h, mockRepos, _ := newHandler()
	bob := &models.User{ID: 2, Username: "bob", Bio: "hey", PhotoCount: 5, IsFollowing: false}
	mockRepos.Users.On("GetByUsernameWithCounts", "bob", int64(0)).Return(bob, nil)

	req := httptest.NewRequest("GET", "/api/user/bob", nil)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()

	h.APIGetUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "bob", body["username"])
	assert.Equal(t, false, body["is_following"])
}

// --- APISearchUsers ---

func TestAPISearchUsers_EmptyQuery(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("GET", "/api/search/users?q=", nil)
	w := httptest.NewRecorder()

	h.APISearchUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Empty(t, body)
}

func TestAPISearchUsers_Success(t *testing.T) {
	h, mockRepos, _ := newHandler()
	users := []models.User{
		{ID: 2, Username: "bob", Avatar: "bob.jpg", PhotoCount: 3},
	}
	mockRepos.Users.On("Search", "bo", 20).Return(users, nil)

	req := httptest.NewRequest("GET", "/api/search/users?q=bo", nil)
	w := httptest.NewRecorder()

	h.APISearchUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body, 1)
	assert.Equal(t, "bob", body[0]["username"])
	assert.Equal(t, float64(3), body[0]["photo_count"])
}

func TestAPISearchUsers_RepoError(t *testing.T) {
	h, mockRepos, _ := newHandler()
	mockRepos.Users.On("Search", "err", 20).Return([]models.User{}, errors.New("db error"))

	req := httptest.NewRequest("GET", "/api/search/users?q=err", nil)
	w := httptest.NewRecorder()

	h.APISearchUsers(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- APIUpdateProfile ---

func TestAPIUpdateProfile_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("POST", "/api/profile", nil)
	w := httptest.NewRecorder()

	h.APIUpdateProfile(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIUpdateProfile_BioTooLong(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	form := url.Values{"bio": {strings.Repeat("x", 501)}}
	req := httptest.NewRequest("POST", "/api/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.APIUpdateProfile(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIUpdateProfile_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("UpdateProfile", int64(1), "new bio").Return(nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	form := url.Values{"bio": {"new bio"}}
	req := httptest.NewRequest("POST", "/api/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.APIUpdateProfile(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"])
}

func TestAPIUpdateProfile_EmptyBio(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("UpdateProfile", int64(1), "").Return(nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	form := url.Values{"bio": {""}}
	req := httptest.NewRequest("POST", "/api/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.APIUpdateProfile(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIUpdateProfile_RepoError(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("UpdateProfile", int64(1), "bio").Return(errors.New("db error"))

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	form := url.Values{"bio": {"bio"}}
	req := httptest.NewRequest("POST", "/api/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.APIUpdateProfile(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIUploadAvatar_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("POST", "/api/avatar", nil)
	w := httptest.NewRecorder()

	h.APIUploadAvatar(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
