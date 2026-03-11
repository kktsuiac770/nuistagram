package notifications

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
	h := New(authHandler, mockRepos.Notifications)
	return h, mockRepos, jwtMgr
}

func TestAPIGetNotifications_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	w := httptest.NewRecorder()

	h.APIGetNotifications(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIGetNotifications_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	actor := &models.User{ID: 2, Username: "bob", Avatar: "bob.jpg"}
	notifications := []models.Notification{
		{
			ID:        1,
			Type:      "like",
			Read:      false,
			CreatedAt: time.Now(),
			PhotoID:   10,
			CommentID: 0,
			Actor:     actor,
		},
	}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Notifications.On("GetByUserID", int64(1), 20, 0).Return(notifications, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIGetNotifications(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body, 1)
	assert.Equal(t, "like", body[0]["type"])
	assert.Equal(t, float64(10), body[0]["photo_id"])
	_, hasCommentID := body[0]["comment_id"]
	assert.False(t, hasCommentID) // zero CommentID should be omitted
}

func TestAPIGetNotifications_WithCommentID(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	actor := &models.User{ID: 2, Username: "bob"}
	notifications := []models.Notification{
		{
			ID:        2,
			Type:      "comment",
			PhotoID:   5,
			CommentID: 99,
			Actor:     actor,
			CreatedAt: time.Now(),
		},
	}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Notifications.On("GetByUserID", int64(1), 20, 0).Return(notifications, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIGetNotifications(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(99), body[0]["comment_id"])
}

func TestAPIGetNotifications_RepoError(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Notifications.On("GetByUserID", int64(1), 20, 0).Return([]models.Notification{}, errors.New("db error"))

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIGetNotifications(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIUnreadCount_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("GET", "/api/notifications/unread", nil)
	w := httptest.NewRecorder()

	h.APIUnreadCount(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIUnreadCount_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Notifications.On("GetUnreadCount", int64(1)).Return(3, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/notifications/unread", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIUnreadCount(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]int
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 3, body["count"])
}

func TestAPIUnreadCount_RepoError(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Notifications.On("GetUnreadCount", int64(1)).Return(0, errors.New("db error"))

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/notifications/unread", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIUnreadCount(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIMarkNotificationRead_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("POST", "/api/notifications/1/read", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.APIMarkNotificationRead(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIMarkNotificationRead_InvalidID(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/notifications/abc/read", nil)
	req.SetPathValue("id", "abc")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIMarkNotificationRead(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIMarkNotificationRead_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Notifications.On("MarkAsRead", int64(7), int64(1)).Return(nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/notifications/7/read", nil)
	req.SetPathValue("id", "7")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIMarkNotificationRead(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"])
}

func TestAPIMarkNotificationRead_RepoError(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Notifications.On("MarkAsRead", int64(7), int64(1)).Return(errors.New("not found"))

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/notifications/7/read", nil)
	req.SetPathValue("id", "7")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIMarkNotificationRead(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIMarkAllNotificationsRead_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("POST", "/api/notifications/read-all", nil)
	w := httptest.NewRecorder()

	h.APIMarkAllNotificationsRead(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIMarkAllNotificationsRead_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Notifications.On("MarkAllAsRead", int64(1)).Return(nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/notifications/read-all", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIMarkAllNotificationsRead(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"])
}

func TestAPIMarkAllNotificationsRead_RepoError(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Notifications.On("MarkAllAsRead", int64(1)).Return(errors.New("db error"))

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/notifications/read-all", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIMarkAllNotificationsRead(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
