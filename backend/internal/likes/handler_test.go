package likes

import (
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
	"nuistagram/internal/ratelimit"
	"nuistagram/internal/repository/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	h := New(authHandler, mockRepos.Likes, mockRepos.Photos, mockRepos.Notifications, &config.Config{}, ratelimit.New())
	return h, mockRepos, jwtMgr
}

func TestAPIToggleLike_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("POST", "/api/photo/1/like", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.APIToggleLike(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIToggleLike_InvalidPhotoID(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/photo/abc/like", nil)
	req.SetPathValue("id", "abc")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIToggleLike(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIToggleLike_Success_Liked(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Likes.On("Toggle", int64(42), int64(1)).Return(true, 5, nil)
	mockRepos.Photos.On("GetOwner", int64(42)).Return("photo.jpg", "thumb.jpg", int64(2), nil).Maybe()
	mockRepos.Notifications.On("Create", int64(2), int64(1), "like", int64(42), int64(0)).Return(int64(1), nil).Maybe()

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/photo/42/like", nil)
	req.SetPathValue("id", "42")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIToggleLike(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["success"])
	assert.Equal(t, true, body["is_liked"])
	assert.Equal(t, float64(5), body["like_count"])
}

func TestAPIToggleLike_Success_Unliked(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Likes.On("Toggle", int64(42), int64(1)).Return(false, 4, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/photo/42/like", nil)
	req.SetPathValue("id", "42")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIToggleLike(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, false, body["is_liked"])
	assert.Equal(t, float64(4), body["like_count"])
}

func TestAPIToggleLike_RepoError(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Likes.On("Toggle", int64(42), int64(1)).Return(false, 0, errors.New("db error"))

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/photo/42/like", nil)
	req.SetPathValue("id", "42")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIToggleLike(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIToggleLike_OwnPhoto_NoNotification(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Likes.On("Toggle", int64(42), int64(1)).Return(true, 3, nil)
	mockRepos.Photos.On("GetOwner", int64(42)).Return("photo.jpg", "thumb.jpg", int64(1), nil).Maybe()

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/photo/42/like", nil)
	req.SetPathValue("id", "42")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIToggleLike(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Notifications.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAPIGetLikers_InvalidPhotoID(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("GET", "/api/photo/nope/likers", nil)
	req.SetPathValue("id", "nope")
	w := httptest.NewRecorder()

	h.APIGetLikers(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIGetLikers_Success(t *testing.T) {
	h, mockRepos, _ := newHandler()
	likers := []models.User{
		{ID: 2, Username: "bob", Avatar: "bob.jpg"},
		{ID: 3, Username: "carol", Avatar: "carol.jpg"},
	}
	mockRepos.Likes.On("GetLikers", int64(10), 20).Return(likers, nil)

	req := httptest.NewRequest("GET", "/api/photo/10/likers", nil)
	req.SetPathValue("id", "10")
	w := httptest.NewRecorder()

	h.APIGetLikers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body, 2)
	assert.Equal(t, "bob", body[0]["username"])
}

func TestAPIGetLikers_DefaultLimit(t *testing.T) {
	h, mockRepos, _ := newHandler()
	mockRepos.Likes.On("GetLikers", int64(1), 20).Return([]models.User{}, nil)

	req := httptest.NewRequest("GET", "/api/photo/1/likers", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.APIGetLikers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Likes.AssertExpectations(t)
}

func TestAPIGetLikers_ClampLimit(t *testing.T) {
	h, mockRepos, _ := newHandler()
	mockRepos.Likes.On("GetLikers", int64(1), 20).Return([]models.User{}, nil)

	req := httptest.NewRequest("GET", "/api/photo/1/likers?limit=999", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.APIGetLikers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Likes.AssertExpectations(t)
}

func TestAPIGetLikers_RepoError(t *testing.T) {
	h, mockRepos, _ := newHandler()
	mockRepos.Likes.On("GetLikers", int64(1), 20).Return([]models.User{}, errors.New("db error"))

	req := httptest.NewRequest("GET", "/api/photo/1/likers", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.APIGetLikers(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIGetLikers_CustomLimit(t *testing.T) {
	h, mockRepos, _ := newHandler()
	mockRepos.Likes.On("GetLikers", int64(5), 10).Return([]models.User{}, nil)

	req := httptest.NewRequest("GET", "/api/photo/5/likers?limit=10", nil)
	req.SetPathValue("id", "5")
	w := httptest.NewRecorder()

	h.APIGetLikers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Likes.AssertExpectations(t)
}
