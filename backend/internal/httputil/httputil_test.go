package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nuistagram/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusCreated, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var got map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "value", got["key"])
}

func TestJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	JSONError(w, http.StatusBadRequest, "bad request")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var got map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "bad request", got["error"])
}

func TestParseID(t *testing.T) {
	assert.Equal(t, int64(42), ParseID("42"))
	assert.Equal(t, int64(0), ParseID(""))
	assert.Equal(t, int64(0), ParseID("notanumber"))
	assert.Equal(t, int64(1), ParseID("1"))
}

func TestCurrentUserID_NilUser(t *testing.T) {
	assert.Equal(t, int64(0), CurrentUserID(nil))
}

func TestCurrentUserID_ValidUser(t *testing.T) {
	u := &models.User{ID: 99}
	assert.Equal(t, int64(99), CurrentUserID(u))
}
