package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nuistagram/internal/models"
	"nuistagram/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- parseTags ---

func TestParseTags_Empty(t *testing.T) {
	assert.Nil(t, parseTags(""))
}

func TestParseTags_Single(t *testing.T) {
	assert.Equal(t, []string{"nature"}, parseTags("nature"))
}

func TestParseTags_Multiple(t *testing.T) {
	assert.Equal(t, []string{"nature", "travel"}, parseTags("nature,travel"))
}

func TestParseTags_Whitespace(t *testing.T) {
	assert.Equal(t, []string{"nature", "travel"}, parseTags(" nature , travel "))
}

func TestParseTags_SkipsBlank(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, parseTags("a,,b"))
}

// --- convertToResponse ---

func TestConvertToResponse(t *testing.T) {
	result := &repository.PaginationResult{
		Photos: []models.PhotoWithNuis{
			{
				Photo: models.Photo{
					ID:           1,
					Filename:     "photo.jpg",
					Thumbnail:    "thumb.jpg",
					UserID:       10,
					Description:  "test",
					TakenAt:      time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
					CreatedAt:    time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC),
					IsFavorite:   true,
					Username:     "alice",
					LikeCount:    3,
					IsLiked:      true,
					CommentCount: 2,
				},
				NuiNames: nil,
			},
		},
		CurrentPage: 1,
		TotalPages:  5,
		TotalCount:  50,
		HasPrev:     false,
		HasNext:     true,
		Pages:       []int{1, 2, 3},
	}

	resp := convertToResponse(result)
	assert.Equal(t, 1, resp.CurrentPage)
	assert.Equal(t, 5, resp.TotalPages)
	assert.Equal(t, 50, resp.TotalCount)
	assert.True(t, resp.HasNext)
	assert.False(t, resp.HasPrev)
	assert.Len(t, resp.Photos, 1)
	assert.Equal(t, "2024-01-15", resp.Photos[0].TakenAt)
	assert.Equal(t, "2024-01-16 12:00:00", resp.Photos[0].CreatedAt)
}

func TestConvertToResponse_ZeroTakenAt(t *testing.T) {
	result := &repository.PaginationResult{
		Photos: []models.PhotoWithNuis{
			{Photo: models.Photo{ID: 1, CreatedAt: time.Now()}},
		},
	}
	resp := convertToResponse(result)
	assert.Equal(t, "", resp.Photos[0].TakenAt)
}

// --- APIGetPhotos ---

func TestAPIGetPhotos_DefaultFeed(t *testing.T) {
	srv, mockRepos := setupTestServer()
	paginationResult := &repository.PaginationResult{Photos: []models.PhotoWithNuis{}}
	mockRepos.Photos.On("GetAll", 1, int64(0)).Return(paginationResult, nil)

	req := httptest.NewRequest("GET", "/api/photos", nil)
	w := httptest.NewRecorder()

	srv.APIGetPhotos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Photos.AssertExpectations(t)
}

func TestAPIGetPhotos_FollowingFeed(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	paginationResult := &repository.PaginationResult{Photos: []models.PhotoWithNuis{}}
	mockRepos.Photos.On("GetFollowingFeed", int64(1), 1).Return(paginationResult, nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/photos?feed=following", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIGetPhotos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Photos.AssertExpectations(t)
}

func TestAPIGetPhotos_TagSearch(t *testing.T) {
	srv, mockRepos := setupTestServer()
	paginationResult := &repository.PaginationResult{Photos: []models.PhotoWithNuis{}}
	mockRepos.Photos.On("Search", []string{"nature"}, "or", 1, int64(0)).Return(paginationResult, nil)

	req := httptest.NewRequest("GET", "/api/photos?tags=nature", nil)
	w := httptest.NewRecorder()

	srv.APIGetPhotos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Photos.AssertExpectations(t)
}

func TestAPIGetPhotos_RepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockRepos.Photos.On("GetAll", 1, int64(0)).Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/api/photos", nil)
	w := httptest.NewRecorder()

	srv.APIGetPhotos(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIGetPhotos_PageDefault(t *testing.T) {
	srv, mockRepos := setupTestServer()
	paginationResult := &repository.PaginationResult{Photos: []models.PhotoWithNuis{}}
	// page=0 or negative should default to 1
	mockRepos.Photos.On("GetAll", 1, int64(0)).Return(paginationResult, nil)

	req := httptest.NewRequest("GET", "/api/photos?page=0", nil)
	w := httptest.NewRecorder()

	srv.APIGetPhotos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Photos.AssertExpectations(t)
}

// --- APIGetPhoto ---

func TestAPIGetPhoto_NotFound(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockRepos.Photos.On("GetByID", int64(99), int64(0)).Return(nil, "", sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/api/photo/99", nil)
	req.SetPathValue("id", "99")
	w := httptest.NewRecorder()

	srv.APIGetPhoto(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIGetPhoto_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	photo := &models.PhotoWithNuis{
		Photo: models.Photo{
			ID:        1,
			Filename:  "photo.jpg",
			CreatedAt: time.Now(),
			TakenAt:   time.Now(),
		},
		NuiNames: []string{"nature"},
	}
	mockRepos.Photos.On("GetByID", int64(1), int64(0)).Return(photo, "alice", nil)

	req := httptest.NewRequest("GET", "/api/photo/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	srv.APIGetPhoto(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body PhotoResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "alice", body.Username)
	assert.Equal(t, int64(1), body.ID)
}

// --- APIGetMe ---

func TestAPIGetMe_Unauthorized(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/me", nil)
	w := httptest.NewRecorder()

	srv.APIGetMe(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIGetMe_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	fullAlice := &models.User{
		ID: 1, Username: "alice", Bio: "hello", Avatar: "a.jpg",
		PhotoCount: 10, FollowerCount: 5, FollowingCount: 3,
	}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByIDWithCounts", int64(1), int64(1)).Return(fullAlice, nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIGetMe(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "alice", body["username"])
	assert.Equal(t, "hello", body["bio"])
	assert.Equal(t, float64(10), body["photo_count"])
}

func TestAPIGetMe_RepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByIDWithCounts", int64(1), int64(1)).Return(nil, errors.New("db error"))

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIGetMe(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- APIGetNuis ---

func TestAPIGetNuis_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	nuis := []models.Nui{{ID: 1, Name: "nature"}, {ID: 2, Name: "travel"}}
	mockRepos.Nuis.On("GetAll").Return(nuis, nil)

	req := httptest.NewRequest("GET", "/api/nuis", nil)
	w := httptest.NewRecorder()

	srv.APIGetNuis(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIGetNuis_RepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockRepos.Nuis.On("GetAll").Return([]models.Nui{}, errors.New("db error"))

	req := httptest.NewRequest("GET", "/api/nuis", nil)
	w := httptest.NewRecorder()

	srv.APIGetNuis(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- APIToggleFavorite ---

func TestAPIToggleFavorite_Unauthorized(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/photo/1/favorite", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	srv.APIToggleFavorite(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIToggleFavorite_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Favorites.On("Toggle", int64(5), int64(1)).Return(true, nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/photo/5/favorite", nil)
	req.SetPathValue("id", "5")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIToggleFavorite(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["is_favorite"])
}

func TestAPIToggleFavorite_RepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Favorites.On("Toggle", int64(5), int64(1)).Return(false, errors.New("db error"))

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/photo/5/favorite", nil)
	req.SetPathValue("id", "5")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIToggleFavorite(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- APIGetUser ---

func TestAPIGetUser_NotFound(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockRepos.Users.On("GetByUsernameWithCounts", "nobody", int64(0)).Return(nil, sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/api/user/nobody", nil)
	req.SetPathValue("username", "nobody")
	w := httptest.NewRecorder()

	srv.APIGetUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIGetUser_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	bob := &models.User{ID: 2, Username: "bob", Bio: "hey", PhotoCount: 5, IsFollowing: false}
	mockRepos.Users.On("GetByUsernameWithCounts", "bob", int64(0)).Return(bob, nil)

	req := httptest.NewRequest("GET", "/api/user/bob", nil)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()

	srv.APIGetUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "bob", body["username"])
	assert.Equal(t, false, body["is_following"])
}

// --- APIGetUserPhotos ---

func TestAPIGetUserPhotos_UserNotFound(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockRepos.Users.On("GetByUsername", "ghost").Return(nil, sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/api/user/ghost/photos", nil)
	req.SetPathValue("username", "ghost")
	w := httptest.NewRecorder()

	srv.APIGetUserPhotos(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIGetUserPhotos_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	paginationResult := &repository.PaginationResult{Photos: []models.PhotoWithNuis{}}
	mockRepos.Photos.On("GetByUser", int64(2), 1, int64(0)).Return(paginationResult, nil)

	req := httptest.NewRequest("GET", "/api/user/bob/photos", nil)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()

	srv.APIGetUserPhotos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- APIGetFollowStatus ---

func TestAPIGetFollowStatus_Unauthorized(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/user/bob/follow-status", nil)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()

	srv.APIGetFollowStatus(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIGetFollowStatus_UserNotFound(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "ghost").Return(nil, sql.ErrNoRows)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/user/ghost/follow-status", nil)
	req.SetPathValue("username", "ghost")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIGetFollowStatus(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIGetFollowStatus_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	mockRepos.Follows.On("IsFollowing", int64(1), int64(2)).Return(true)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/user/bob/follow-status", nil)
	req.SetPathValue("username", "bob")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIGetFollowStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["is_following"])
}
