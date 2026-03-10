package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Chain ---

func TestChain_OrderPreserved(t *testing.T) {
	var order []int
	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 1)
			next.ServeHTTP(w, r)
		})
	}
	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 2)
			next.ServeHTTP(w, r)
		})
	}
	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, 3)
	})

	Chain(base, m1, m2).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, []int{1, 2, 3}, order)
}

func TestChain_NoMiddlewares(t *testing.T) {
	called := false
	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	Chain(base).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.True(t, called)
}

// --- Recovery ---

func TestRecovery_PanicReturns500(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Internal server error")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestRecovery_NoPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRecovery_PanicWithError(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var s []int
		_ = s[0] // index out of range
	}))

	w := httptest.NewRecorder()
	assert.NotPanics(t, func() {
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/panic", nil))
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- RequestID ---

func TestRequestID_GeneratesIfMissing(t *testing.T) {
	var capturedID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	assert.NotEmpty(t, capturedID)
	assert.Equal(t, capturedID, w.Header().Get("X-Request-ID"))
}

func TestRequestID_UsesExistingHeader(t *testing.T) {
	var capturedID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "my-fixed-id")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, "my-fixed-id", capturedID)
	assert.Equal(t, "my-fixed-id", w.Header().Get("X-Request-ID"))
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	var ids []string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids = append(ids, GetRequestID(r.Context()))
	}))

	for i := 0; i < 5; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}

	for i := 1; i < len(ids); i++ {
		assert.NotEqual(t, ids[0], ids[i], "request IDs should be unique")
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	id := GetRequestID(context.Background())
	assert.Equal(t, "", id)
}

// --- ResponseWriter ---

func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &ResponseWriter{ResponseWriter: rec, Status: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.Status)
	assert.True(t, rw.WroteHeader)

	// Second call is a no-op
	rw.WriteHeader(http.StatusOK)
	assert.Equal(t, http.StatusNotFound, rw.Status)
}

func TestResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &ResponseWriter{ResponseWriter: rec, Status: http.StatusOK}

	n, err := rw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.True(t, rw.WroteHeader)
	assert.Equal(t, http.StatusOK, rw.Status)
}

func TestResponseWriter_WriteHeader_ThenWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &ResponseWriter{ResponseWriter: rec, Status: http.StatusOK}

	rw.WriteHeader(http.StatusCreated)
	_, _ = rw.Write([]byte("body"))

	assert.Equal(t, http.StatusCreated, rw.Status)
	assert.Equal(t, "body", rec.Body.String())
}

// --- Logging ---

func TestLogging_PassesThrough(t *testing.T) {
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestLogging_DefaultStatus200(t *testing.T) {
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't call WriteHeader explicitly
		w.Write([]byte("ok"))
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Tracing ---

func TestTracing_PassesThrough(t *testing.T) {
	handler := Tracing(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTracing_ErrorStatus(t *testing.T) {
	handler := Tracing(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/fail", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
