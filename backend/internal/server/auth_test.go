package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jwtpkg "nuistagram/internal/jwt"
	"nuistagram/internal/models"
	"nuistagram/internal/repository"
	"nuistagram/internal/repository/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func setupTestServer() (*Server, *mocks.MockRepositories) {
	mockRepos := mocks.NewMockRepositories()
	jwtMgr, _ := jwtpkg.NewManager("test-secret-key-for-unit-tests-32b", 15*time.Minute, 7*24*time.Hour)
	srv := &Server{
		Repos: &repository.Repositories{
			Users:         mockRepos.Users,
			Nuis:          mockRepos.Nuis,
			Photos:        mockRepos.Photos,
			Albums:        mockRepos.Albums,
			Favorites:     mockRepos.Favorites,
			Follows:       mockRepos.Follows,
			Likes:         mockRepos.Likes,
			Comments:      mockRepos.Comments,
			Notifications: mockRepos.Notifications,
		},
		JWT: jwtMgr,
	}
	return srv, mockRepos
}

func TestRegister_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()

	mockRepos.Users.On("Create", "testuser", mock.AnythingOfType("string")).Return(int64(1), nil)

	form := url.Values{}
	form.Add("username", "testuser")
	form.Add("password", "Password123")

	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.Register(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotEmpty(t, response["access_token"])
	assert.NotEmpty(t, response["refresh_token"])
	assert.Equal(t, float64(900), response["expires_in"])
	assert.Equal(t, "Bearer", response["token_type"])
	assert.Empty(t, w.Result().Cookies())
	mockRepos.Users.AssertExpectations(t)
}

func TestRegister_MissingFields(t *testing.T) {
	srv, _ := setupTestServer()

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"empty both", "", ""},
		{"empty username", "", "Password123"},
		{"empty password", "testuser", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Add("username", tt.username)
			form.Add("password", tt.password)

			req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			srv.Register(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestRegister_UsernameTooShort(t *testing.T) {
	srv, _ := setupTestServer()

	form := url.Values{}
	form.Add("username", "ab")
	form.Add("password", "Password123")

	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_WeakPassword(t *testing.T) {
	srv, _ := setupTestServer()

	tests := []struct {
		name     string
		password string
	}{
		{"too short", "Pass1"},
		{"no uppercase", "password123"},
		{"no lowercase", "PASSWORD123"},
		{"no digit", "Passwordddd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Add("username", "testuser")
			form.Add("password", tt.password)

			req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			srv.Register(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	srv, mockRepos := setupTestServer()

	mockRepos.Users.On("Create", "existinguser", mock.AnythingOfType("string")).Return(int64(0), sql.ErrNoRows)

	form := url.Values{}
	form.Add("username", "existinguser")
	form.Add("password", "Password123")

	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.DefaultCost)
	mockUser := &models.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}

	mockRepos.Users.On("GetByUsername", "testuser").Return(mockUser, nil)

	form := url.Values{}
	form.Add("username", "testuser")
	form.Add("password", "Password123")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.Login(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotEmpty(t, response["access_token"])
	assert.NotEmpty(t, response["refresh_token"])
	assert.Equal(t, float64(900), response["expires_in"])
	assert.Equal(t, "Bearer", response["token_type"])
	assert.Empty(t, w.Result().Cookies())
	mockRepos.Users.AssertExpectations(t)
}

func TestLogin_InvalidUsername(t *testing.T) {
	srv, mockRepos := setupTestServer()

	mockRepos.Users.On("GetByUsername", "nonexistent").Return(nil, sql.ErrNoRows)

	form := url.Values{}
	form.Add("username", "nonexistent")
	form.Add("password", "Password123")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestLogin_InvalidPassword(t *testing.T) {
	srv, mockRepos := setupTestServer()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("CorrectPass123"), bcrypt.DefaultCost)
	mockUser := &models.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}

	mockRepos.Users.On("GetByUsername", "testuser").Return(mockUser, nil)

	form := url.Values{}
	form.Add("username", "testuser")
	form.Add("password", "WrongPassword123")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	srv.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestLogout_Success(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/logout", strings.NewReader(`{"refresh_token":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Logout(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]bool
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["success"])
	assert.Empty(t, w.Result().Cookies())
}

func TestRefresh_Success(t *testing.T) {
	srv, mockRepos := setupTestServer()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.DefaultCost)
	mockUser := &models.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)

	refreshToken, _ := srv.JWT.GenerateRefreshToken(1)

	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req := httptest.NewRequest("POST", "/api/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Refresh(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotEmpty(t, response["access_token"])
	assert.NotEmpty(t, response["refresh_token"])
	assert.NotEqual(t, refreshToken, response["refresh_token"]) // rotation: new token issued
	mockRepos.Users.AssertExpectations(t)
}

func TestRefresh_InvalidToken(t *testing.T) {
	srv, _ := setupTestServer()

	body, _ := json.Marshal(map[string]string{"refresh_token": "notvalid"})
	req := httptest.NewRequest("POST", "/api/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Refresh(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefresh_MissingToken(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/refresh", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Refresh(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"valid password", "Password123", true},
		{"too short", "Pass1", false},
		{"no uppercase", "password123", false},
		{"no lowercase", "PASSWORD123", false},
		{"no digit", "Passwordddd", false},
		{"exactly 8 chars", "Passwor1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
