package auth

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

	"nuistagram/internal/config"
	jwtpkg "nuistagram/internal/jwt"
	"nuistagram/internal/models"
	"nuistagram/internal/monitoring/metrics"
	"nuistagram/internal/repository/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
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
	h := New(mockRepos.Users, jwtMgr)
	return h, mockRepos, jwtMgr
}

func TestRegister_Success(t *testing.T) {
	h, mockRepos, _ := newHandler()

	mockRepos.Users.On("Create", "testuser", mock.AnythingOfType("string")).Return(int64(1), nil)

	form := url.Values{}
	form.Add("username", "testuser")
	form.Add("password", "Password123")

	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Register(w, req)

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
	h, _, _ := newHandler()

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

			h.Register(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestRegister_UsernameTooShort(t *testing.T) {
	h, _, _ := newHandler()

	form := url.Values{}
	form.Add("username", "ab")
	form.Add("password", "Password123")

	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_WeakPassword(t *testing.T) {
	h, _, _ := newHandler()

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

			h.Register(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	h, mockRepos, _ := newHandler()

	mockRepos.Users.On("Create", "existinguser", mock.AnythingOfType("string")).Return(int64(0), sql.ErrNoRows)

	form := url.Values{}
	form.Add("username", "existinguser")
	form.Add("password", "Password123")

	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	h, mockRepos, _ := newHandler()

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

	h.Login(w, req)

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
	h, mockRepos, _ := newHandler()

	mockRepos.Users.On("GetByUsername", "nonexistent").Return(nil, sql.ErrNoRows)

	form := url.Values{}
	form.Add("username", "nonexistent")
	form.Add("password", "Password123")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestLogin_InvalidPassword(t *testing.T) {
	h, mockRepos, _ := newHandler()

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

	h.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestLogout_Success(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("POST", "/logout", strings.NewReader(`{"refresh_token":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Logout(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]bool
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["success"])
	assert.Empty(t, w.Result().Cookies())
}

func TestRefresh_Success(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.DefaultCost)
	mockUser := &models.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)

	refreshToken, _ := jwtMgr.GenerateRefreshToken(1)

	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req := httptest.NewRequest("POST", "/api/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotEmpty(t, response["access_token"])
	assert.NotEmpty(t, response["refresh_token"])
	assert.NotEqual(t, refreshToken, response["refresh_token"])
	mockRepos.Users.AssertExpectations(t)
}

func TestRefresh_InvalidToken(t *testing.T) {
	h, _, _ := newHandler()

	body, _ := json.Marshal(map[string]string{"refresh_token": "notvalid"})
	req := httptest.NewRequest("POST", "/api/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefresh_MissingToken(t *testing.T) {
	h, _, _ := newHandler()

	req := httptest.NewRequest("POST", "/api/refresh", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Refresh(w, req)

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

func TestCurrentUser_NoAuthHeader(t *testing.T) {
	h, _, _ := newHandler()
	req := httptest.NewRequest("GET", "/", nil)
	assert.Nil(t, h.CurrentUser(req))
}

func TestCurrentUser_InvalidToken(t *testing.T) {
	h, _, _ := newHandler()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	assert.Nil(t, h.CurrentUser(req))
}

func TestCurrentUser_ValidToken(t *testing.T) {
	h, mockRepos, jwtMgr := newHandler()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)

	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	got := h.CurrentUser(req)
	assert.NotNil(t, got)
	assert.Equal(t, int64(1), got.ID)
	mockRepos.Users.AssertExpectations(t)
}

func TestCurrentUser_BearerPrefixRequired(t *testing.T) {
	h, _, jwtMgr := newHandler()
	token, _ := jwtMgr.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", token) // missing "Bearer " prefix
	assert.Nil(t, h.CurrentUser(req))
}
