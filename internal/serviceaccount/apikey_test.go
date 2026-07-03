package serviceaccount

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAPIKey(t *testing.T) {
	newHandler := func() http.Handler {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		return ValidateAPIKey("expected-key", next)
	}

	t.Run("rejects missing header", func(t *testing.T) {
		h := newHandler()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects wrong key", func(t *testing.T) {
		h := newHandler()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(apiKeyHeader, "wrong-key")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("accepts matching key", func(t *testing.T) {
		h := newHandler()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(apiKeyHeader, "expected-key")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
