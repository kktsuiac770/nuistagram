package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nuistagram/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authRequest creates a request authenticated as the given user.
// It mocks GetByID on mockRepos.Users and returns a Bearer-token request.
func authRequest(t *testing.T, srv *Server, method, path string, body http.Handler, userID int64, username string) *http.Request {
	t.Helper()
	token, err := srv.JWT.GenerateAccessToken(userID, username)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// --- writeJSON / jsonError ---

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var got map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "value", got["key"])
}

func TestJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	jsonError(w, http.StatusBadRequest, "bad request")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var got map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "bad request", got["error"])
}

// --- parseID ---

func TestParseID(t *testing.T) {
	assert.Equal(t, int64(42), parseID("42"))
	assert.Equal(t, int64(0), parseID(""))
	assert.Equal(t, int64(0), parseID("notanumber"))
	assert.Equal(t, int64(1), parseID("1"))
}

// --- currentUserID ---

func TestCurrentUserID_NilUser(t *testing.T) {
	assert.Equal(t, int64(0), currentUserID(nil))
}

func TestCurrentUserID_ValidUser(t *testing.T) {
	u := &models.User{ID: 99}
	assert.Equal(t, int64(99), currentUserID(u))
}

// --- currentUser (method on Server) ---

func TestCurrentUser_NoAuthHeader(t *testing.T) {
	srv, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/", nil)
	assert.Nil(t, srv.currentUser(req))
}

func TestCurrentUser_InvalidToken(t *testing.T) {
	srv, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	assert.Nil(t, srv.currentUser(req))
}

func TestCurrentUser_ValidToken(t *testing.T) {
	srv, mockRepos := setupTestServer()
	mockUser := &models.User{ID: 1, Username: "alice"}
	mockRepos.Users.On("GetByID", int64(1)).Return(mockUser, nil)

	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	got := srv.currentUser(req)
	assert.NotNil(t, got)
	assert.Equal(t, int64(1), got.ID)
	mockRepos.Users.AssertExpectations(t)
}

func TestCurrentUser_BearerPrefixRequired(t *testing.T) {
	srv, _ := setupTestServer()
	token, _ := srv.JWT.GenerateAccessToken(1, "alice")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", token) // missing "Bearer " prefix
	assert.Nil(t, srv.currentUser(req))
}
