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

func mockLikeUser(mockRepos *mocks.MockRepositories, userID int64, username string) {
	mockRepos.Users.On("GetByID", userID).Return(&models.User{
		ID:       userID,
		Username: username,
	}, nil)
}

func setupLikeTest(t *testing.T, userID int64) (*http.Request, string) {
	sessionToken := CreateSession(userID)
	req := httptest.NewRequest("POST", "/api/photo/1/like", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionToken})
	req.SetPathValue("id", "1")
	return req, sessionToken
}

func TestAPIToggleLike_Success(t *testing.T) {
	mockRepos := setupAuthMocks()
	req, _ := setupLikeTest(t, 1)

	mockLikeUser(mockRepos, 1, "testuser")
	mockRepos.Likes.On("Toggle", int64(1), int64(1)).Return(true, nil)
	mockRepos.Photos.On("GetByID", int64(1), int64(1)).Return(&models.PhotoWithNuis{
		Photo: models.Photo{ID: 1, UserID: 2},
	}, "testuser", nil)
	mockRepos.Likes.On("GetLikeCount", int64(1)).Return(5, nil)
	mockRepos.Notifications.On("Create", int64(2), int64(1), "like", int64(1), int64(0)).Return(int64(1), nil)

	w := httptest.NewRecorder()
	APIToggleLike(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["success"].(bool))
	assert.True(t, response["is_liked"].(bool))
	assert.Equal(t, float64(5), response["like_count"].(float64))
	mockRepos.Likes.AssertExpectations(t)
	mockRepos.Photos.AssertExpectations(t)
}

func TestAPIToggleLike_Unauthorized(t *testing.T) {
	setupAuthMocks()

	req := httptest.NewRequest("POST", "/api/photo/1/like", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	APIToggleLike(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIToggleLike_InvalidPhotoID(t *testing.T) {
	mockRepos := setupAuthMocks()
	req, _ := setupLikeTest(t, 1)

	mockLikeUser(mockRepos, 1, "testuser")
	req.SetPathValue("id", "invalid")
	w := httptest.NewRecorder()

	APIToggleLike(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIToggleLike_Unlike(t *testing.T) {
	mockRepos := setupAuthMocks()
	req, _ := setupLikeTest(t, 1)

	mockLikeUser(mockRepos, 1, "testuser")
	mockRepos.Likes.On("Toggle", int64(1), int64(1)).Return(false, nil)
	mockRepos.Likes.On("GetLikeCount", int64(1)).Return(4, nil)

	w := httptest.NewRecorder()
	APIToggleLike(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(t, response["is_liked"].(bool))
	mockRepos.Likes.AssertExpectations(t)
}

func TestAPIGetLikers_Success(t *testing.T) {
	mockRepos := setupAuthMocks()

	req := httptest.NewRequest("GET", "/api/photo/1/likers?limit=10", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	likers := []models.User{
		{ID: 1, Username: "user1", Avatar: ""},
		{ID: 2, Username: "user2", Avatar: ""},
	}
	mockRepos.Likes.On("GetLikers", int64(1), 10).Return(likers, nil)

	APIGetLikers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Len(t, response, 2)
	assert.Equal(t, "user1", response[0]["username"])
	mockRepos.Likes.AssertExpectations(t)
}

func TestAPIGetLikers_DefaultLimit(t *testing.T) {
	mockRepos := setupAuthMocks()

	req := httptest.NewRequest("GET", "/api/photo/1/likers", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	mockRepos.Likes.On("GetLikers", int64(1), 20).Return([]models.User{}, nil)

	APIGetLikers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Likes.AssertExpectations(t)
}

func TestAPIGetLikers_InvalidPhotoID(t *testing.T) {
	setupAuthMocks()

	req := httptest.NewRequest("GET", "/api/photo/invalid/likers", nil)
	req.SetPathValue("id", "invalid")
	w := httptest.NewRecorder()

	APIGetLikers(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
