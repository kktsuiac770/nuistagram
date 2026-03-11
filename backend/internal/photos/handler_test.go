package photos

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
	"nuistagram/internal/ratelimit"
	"nuistagram/internal/repository"
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
	h := New(
		authHandler, mockRepos.Users, mockRepos.Photos, mockRepos.Nuis,
		mockRepos.Albums, mockRepos.Favorites, nil, &config.Config{}, ratelimit.New(),
	)
	return h, mockRepos, jwtMgr
}

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
	h, mockRepos, _ := newHandler()
	paginationResult := &repository.PaginationResult{Photos: []models.PhotoWithNuis{}}
	mockRepos.Photos.On("GetAll", 1, int64(0)).Return(paginationResult, nil)

	req := httptest.NewRequest("GET", "/api/photos", nil)
	w := httptest.NewRecorder()

	h.APIGetPhotos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Photos.AssertExpectations(t)
}

func TestAPIGetPhotos_FollowingFeed(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	paginationResult := &repository.PaginationResult{Photos: []models.PhotoWithNuis{}}
	mockRepos.Photos.On("GetFollowingFeed", int64(1), 1).Return(paginationResult, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/api/photos?feed=following", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIGetPhotos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Photos.AssertExpectations(t)
}

func TestAPIGetPhotos_TagSearch(t *testing.T) {
	h, mockRepos, _ := newHandler()
	paginationResult := &repository.PaginationResult{Photos: []models.PhotoWithNuis{}}
	mockRepos.Photos.On("Search", []string{"nature"}, "or", 1, int64(0)).Return(paginationResult, nil)

	req := httptest.NewRequest("GET", "/api/photos?tags=nature", nil)
	w := httptest.NewRecorder()

	h.APIGetPhotos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Photos.AssertExpectations(t)
}

func TestAPIGetPhotos_RepoError(t *testing.T) {
	h, mockRepos, _ := newHandler()
	mockRepos.Photos.On("GetAll", 1, int64(0)).Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/api/photos", nil)
	w := httptest.NewRecorder()

	h.APIGetPhotos(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIGetPhotos_PageDefault(t *testing.T) {
	h, mockRepos, _ := newHandler()
	paginationResult := &repository.PaginationResult{Photos: []models.PhotoWithNuis{}}
	mockRepos.Photos.On("GetAll", 1, int64(0)).Return(paginationResult, nil)

	req := httptest.NewRequest("GET", "/api/photos?page=0", nil)
	w := httptest.NewRecorder()

	h.APIGetPhotos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Photos.AssertExpectations(t)
}

// --- APIGetPhoto ---

func TestAPIGetPhoto_NotFound(t *testing.T) {
	h, mockRepos, _ := newHandler()
	mockRepos.Photos.On("GetByID", int64(99), int64(0)).Return(nil, "", sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/api/photo/99", nil)
	req.SetPathValue("id", "99")
	w := httptest.NewRecorder()

	h.APIGetPhoto(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIGetPhoto_Success(t *testing.T) {
	h, mockRepos, _ := newHandler()
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

	h.APIGetPhoto(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body PhotoResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "alice", body.Username)
	assert.Equal(t, int64(1), body.ID)
}

// --- APIGetNuis ---

func TestAPIGetNuis_Success(t *testing.T) {
	h, mockRepos, _ := newHandler()
	nuis := []models.Nui{{ID: 1, Name: "nature"}, {ID: 2, Name: "travel"}}
	mockRepos.Nuis.On("GetAll").Return(nuis, nil)

	req := httptest.NewRequest("GET", "/api/nuis", nil)
	w := httptest.NewRecorder()

	h.APIGetNuis(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIGetNuis_RepoError(t *testing.T) {
	h, mockRepos, _ := newHandler()
	mockRepos.Nuis.On("GetAll").Return([]models.Nui{}, errors.New("db error"))

	req := httptest.NewRequest("GET", "/api/nuis", nil)
	w := httptest.NewRecorder()

	h.APIGetNuis(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- APIToggleFavorite ---

func TestAPIToggleFavorite_Unauthorized(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("POST", "/api/photo/1/favorite", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.APIToggleFavorite(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIToggleFavorite_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Favorites.On("Toggle", int64(5), int64(1)).Return(true, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/photo/5/favorite", nil)
	req.SetPathValue("id", "5")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIToggleFavorite(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["is_favorite"])
}

func TestAPIToggleFavorite_RepoError(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Favorites.On("Toggle", int64(5), int64(1)).Return(false, errors.New("db error"))

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/photo/5/favorite", nil)
	req.SetPathValue("id", "5")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.APIToggleFavorite(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- APIGetUserPhotos ---

func TestAPIGetUserPhotos_UserNotFound(t *testing.T) {
	h, mockRepos, _ := newHandler()
	mockRepos.Users.On("GetByUsername", "ghost").Return(nil, sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/api/user/ghost/photos", nil)
	req.SetPathValue("username", "ghost")
	w := httptest.NewRecorder()

	h.APIGetUserPhotos(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIGetUserPhotos_Success(t *testing.T) {
	h, mockRepos, _ := newHandler()
	bob := &models.User{ID: 2, Username: "bob"}
	mockRepos.Users.On("GetByUsername", "bob").Return(bob, nil)
	paginationResult := &repository.PaginationResult{Photos: []models.PhotoWithNuis{}}
	mockRepos.Photos.On("GetByUser", int64(2), 1, int64(0)).Return(paginationResult, nil)

	req := httptest.NewRequest("GET", "/api/user/bob/photos", nil)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()

	h.APIGetUserPhotos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
