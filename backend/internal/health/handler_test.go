package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthz(t *testing.T) {
	h := New(nil)
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	h.Healthz(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
}

func TestReadyz_DBNil_Returns503(t *testing.T) {
	h := New(nil) // nil DB causes HealthCheck to fail

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	h.Readyz(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "not ready", body["status"])
	checks, ok := body["checks"].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, checks["database"])
}
