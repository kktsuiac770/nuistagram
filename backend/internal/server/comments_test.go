package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"nuistagram/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIGetComments_InvalidPhotoID(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/photo/abc/comments", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	srv.APIGetComments(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIGetComments_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	comments := []models.Comment{
		{ID: 1, PhotoID: 10, UserID: 2, Content: "nice!", CreatedAt: time.Now(), Username: "bob"},
	}
	mockRepos.Comments.On("GetByPhotoID", int64(10), 20, 0).Return(comments, nil)

	req := httptest.NewRequest("GET", "/api/photo/10/comments", nil)
	req.SetPathValue("id", "10")
	w := httptest.NewRecorder()

	srv.APIGetComments(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body, 1)
	assert.Equal(t, "nice!", body[0]["content"])
	assert.Equal(t, "bob", body[0]["username"])
}

func TestAPIGetComments_WithPagination(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockRepos.Comments.On("GetByPhotoID", int64(5), 10, 20).Return([]models.Comment{}, nil)

	req := httptest.NewRequest("GET", "/api/photo/5/comments?limit=10&offset=20", nil)
	req.SetPathValue("id", "5")
	w := httptest.NewRecorder()

	srv.APIGetComments(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Comments.AssertExpectations(t)
}

func TestAPIGetComments_RepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockRepos.Comments.On("GetByPhotoID", int64(1), 20, 0).Return([]models.Comment{}, errors.New("db error"))

	req := httptest.NewRequest("GET", "/api/photo/1/comments", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	srv.APIGetComments(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPICreateComment_Unauthorized(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/photo/1/comment", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	srv.APICreateComment(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPICreateComment_InvalidPhotoID(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/photo/abc/comment", nil)
	req.SetPathValue("id", "abc")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APICreateComment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPICreateComment_EmptyContent(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	form := url.Values{"content": {""}}
	req := httptest.NewRequest("POST", "/api/photo/1/comment", strings.NewReader(form.Encode()))
	req.SetPathValue("id", "1")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.APICreateComment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPICreateComment_ContentTooLong(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	form := url.Values{"content": {strings.Repeat("x", 1001)}}
	req := httptest.NewRequest("POST", "/api/photo/1/comment", strings.NewReader(form.Encode()))
	req.SetPathValue("id", "1")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.APICreateComment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPICreateComment_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Comments.On("Create", int64(10), int64(1), "great photo!").Return(int64(99), nil)
	// Async notification
	mockRepos.Photos.On("GetOwner", int64(10)).Return("photo.jpg", "thumb.jpg", int64(2), nil).Maybe()
	mockRepos.Notifications.On("Create", int64(2), int64(1), "comment", int64(10), int64(99)).Return(int64(1), nil).Maybe()

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	form := url.Values{"content": {"great photo!"}}
	req := httptest.NewRequest("POST", "/api/photo/10/comment", strings.NewReader(form.Encode()))
	req.SetPathValue("id", "10")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.APICreateComment(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["success"])
	assert.Equal(t, float64(99), body["comment_id"])
}

func TestAPICreateComment_RepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Comments.On("Create", int64(1), int64(1), "oops").Return(int64(0), errors.New("db error"))

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	form := url.Values{"content": {"oops"}}
	req := httptest.NewRequest("POST", "/api/photo/1/comment", strings.NewReader(form.Encode()))
	req.SetPathValue("id", "1")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.APICreateComment(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIDeleteComment_Unauthorized(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/comment/1/delete", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	srv.APIDeleteComment(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIDeleteComment_InvalidCommentID(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/comment/abc/delete", nil)
	req.SetPathValue("id", "abc")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIDeleteComment(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIDeleteComment_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Comments.On("Delete", int64(5), int64(1)).Return(nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/comment/5/delete", nil)
	req.SetPathValue("id", "5")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIDeleteComment(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"])
}

func TestAPIDeleteComment_RepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)
	mockRepos.Comments.On("Delete", int64(5), int64(1)).Return(errors.New("forbidden"))

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("POST", "/api/comment/5/delete", nil)
	req.SetPathValue("id", "5")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	srv.APIDeleteComment(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
