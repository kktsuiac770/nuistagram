package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nuistagram/internal/models"
	"nuistagram/internal/repository/mocks"

	"github.com/stretchr/testify/assert"
)

func setupFollowTest(t *testing.T, userID int64) *http.Request {
	sessionToken := CreateSession(userID)
	req := httptest.NewRequest("POST", "/api/user/2/follow", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionToken})
	req.SetPathValue("id", "2")
	return req
}

func mockCurrentUser(mockRepos *mocks.MockRepositories, userID int64, username string) {
	mockRepos.Users.On("GetByID", userID).Return(&models.User{
		ID:       userID,
		Username: username,
	}, nil)
}

func TestAPIFollow_Success(t *testing.T) {
	mockRepos := setupAuthMocks()
	req := setupFollowTest(t, 1)

	mockCurrentUser(mockRepos, 1, "testuser")
	mockRepos.Follows.On("Follow", int64(1), int64(2)).Return(nil)
	mockRepos.Notifications.On("Create", int64(2), int64(1), "follow", int64(0), int64(0)).Return(int64(1), nil)

	w := httptest.NewRecorder()
	APIFollow(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]bool
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["success"])
	mockRepos.Follows.AssertExpectations(t)
}

func TestAPIFollow_Unauthorized(t *testing.T) {
	setupAuthMocks()

	req := httptest.NewRequest("POST", "/api/user/2/follow", nil)
	req.SetPathValue("id", "2")
	w := httptest.NewRecorder()

	APIFollow(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIFollow_InvalidUserID(t *testing.T) {
	mockRepos := setupAuthMocks()
	req := setupFollowTest(t, 1)

	mockCurrentUser(mockRepos, 1, "testuser")
	req.SetPathValue("id", "invalid")
	w := httptest.NewRecorder()

	APIFollow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIFollow_CannotFollowSelf(t *testing.T) {
	mockRepos := setupAuthMocks()
	req := httptest.NewRequest("POST", "/api/user/1/follow", nil)
	sessionToken := CreateSession(1)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionToken})
	req.SetPathValue("id", "1")

	mockCurrentUser(mockRepos, 1, "testuser")
	w := httptest.NewRecorder()

	APIFollow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIUnfollow_Success(t *testing.T) {
	mockRepos := setupAuthMocks()
	req := setupFollowTest(t, 1)

	mockCurrentUser(mockRepos, 1, "testuser")
	mockRepos.Follows.On("Unfollow", int64(1), int64(2)).Return(nil)

	w := httptest.NewRecorder()
	APIUnfollow(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]bool
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["success"])
	mockRepos.Follows.AssertExpectations(t)
}

func TestAPIUnfollow_Unauthorized(t *testing.T) {
	setupAuthMocks()

	req := httptest.NewRequest("POST", "/api/user/2/unfollow", nil)
	req.SetPathValue("id", "2")
	w := httptest.NewRecorder()

	APIUnfollow(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIFollowStatus_Following(t *testing.T) {
	mockRepos := setupAuthMocks()
	req := setupFollowTest(t, 1)

	mockCurrentUser(mockRepos, 1, "testuser")
	mockRepos.Follows.On("IsFollowing", int64(1), int64(2)).Return(true)

	w := httptest.NewRecorder()
	APIFollowStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]bool
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["is_following"])
	mockRepos.Follows.AssertExpectations(t)
}

func TestAPIFollowStatus_NotFollowing(t *testing.T) {
	mockRepos := setupAuthMocks()
	req := setupFollowTest(t, 1)

	mockCurrentUser(mockRepos, 1, "testuser")
	mockRepos.Follows.On("IsFollowing", int64(1), int64(2)).Return(false)

	w := httptest.NewRecorder()
	APIFollowStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]bool
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(t, response["is_following"])
	mockRepos.Follows.AssertExpectations(t)
}

func TestAPIGetFollowers_Success(t *testing.T) {
	mockRepos := setupAuthMocks()

	req := httptest.NewRequest("GET", "/api/user/1/followers", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	followers := []models.User{
		{ID: 2, Username: "follower1"},
		{ID: 3, Username: "follower2"},
	}
	mockRepos.Follows.On("GetFollowers", int64(1), 50).Return(followers, nil)

	APIGetFollowers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Follows.AssertExpectations(t)
}

func TestAPIGetFollowing_Success(t *testing.T) {
	mockRepos := setupAuthMocks()

	req := httptest.NewRequest("GET", "/api/user/1/following", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	following := []models.User{
		{ID: 2, Username: "following1"},
		{ID: 3, Username: "following2"},
	}
	mockRepos.Follows.On("GetFollowing", int64(1), 50).Return(following, nil)

	APIGetFollowing(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Follows.AssertExpectations(t)
}

func TestAPIGetFollowers_InvalidUserID(t *testing.T) {
	setupAuthMocks()

	req := httptest.NewRequest("GET", "/api/user/invalid/followers", nil)
	req.SetPathValue("id", "invalid")
	w := httptest.NewRecorder()

	APIGetFollowers(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
