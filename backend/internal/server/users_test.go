package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"nuistagram/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPISearchUsers_EmptyQuery(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/search/users?q=", nil)
	w := httptest.NewRecorder()

	srv.APISearchUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Empty(t, body)
}

func TestAPISearchUsers_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	users := []models.User{
		{ID: 2, Username: "bob", Avatar: "bob.jpg", PhotoCount: 3},
	}
	mockRepos.Users.On("Search", "bo", 20).Return(users, nil)

	req := httptest.NewRequest("GET", "/api/search/users?q=bo", nil)
	w := httptest.NewRecorder()

	srv.APISearchUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body, 1)
	assert.Equal(t, "bob", body[0]["username"])
	assert.Equal(t, float64(3), body[0]["photo_count"])
}

func TestAPISearchUsers_RepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockRepos.Users.On("Search", "err", 20).Return([]models.User{}, errors.New("db error"))

	req := httptest.NewRequest("GET", "/api/search/users?q=err", nil)
	w := httptest.NewRecorder()

	srv.APISearchUsers(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIUpdateProfile_Unauthorized(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/profile", nil)
	w := httptest.NewRecorder()

	srv.APIUpdateProfile(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIUpdateProfile_BioTooLong(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	form := url.Values{"bio": {strings.Repeat("x", 501)}}
	req := httptest.NewRequest("POST", "/api/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.APIUpdateProfile(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIUpdateProfile_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("UpdateProfile", int64(1), "new bio").Return(nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	form := url.Values{"bio": {"new bio"}}
	req := httptest.NewRequest("POST", "/api/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.APIUpdateProfile(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"])
}

func TestAPIUpdateProfile_EmptyBio(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("UpdateProfile", int64(1), "").Return(nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	form := url.Values{"bio": {""}}
	req := httptest.NewRequest("POST", "/api/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.APIUpdateProfile(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIUpdateProfile_RepoError(t *testing.T) {
	srv, mockRepos := setupTestServer()
	alice := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(alice, nil)
	mockRepos.Users.On("UpdateProfile", int64(1), "bio").Return(errors.New("db error"))

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	form := url.Values{"bio": {"bio"}}
	req := httptest.NewRequest("POST", "/api/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.APIUpdateProfile(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIUploadAvatar_Unauthorized(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/avatar", nil)
	w := httptest.NewRecorder()

	srv.APIUploadAvatar(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
