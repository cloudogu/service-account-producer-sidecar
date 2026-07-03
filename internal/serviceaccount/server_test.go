package serviceaccount

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateServer(t *testing.T) {
	stub := &stubManager{
		createOrUpdateFunc: func(ctx context.Context, consumer string, params map[string]string) (map[string]string, error) {
			return map[string]string{"username": "u"}, nil
		},
		deleteFunc: func(ctx context.Context, consumer string) error { return nil },
		existsFunc: func(ctx context.Context, consumer string) (bool, error) { return true, nil },
	}
	srv := CreateServer(":8080", "expected-key", stub)

	require.Equal(t, ":8080", srv.Addr)

	// PUT goes to the endpoint verbatim, without a trailing slash - matches the real
	// service-account-operator HTTP client (internal/producer/http_client.go), which never
	// appends anything to the endpoint for PUT.
	t.Run("PUT /serviceaccounts requires api key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/serviceaccounts", nil)

		srv.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("PUT /serviceaccounts succeeds with valid key and body", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/serviceaccounts", strings.NewReader(`{"consumer":"jenkins"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(apiKeyHeader, "expected-key")

		srv.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("PUT /serviceaccounts/:consumer does not match the exact PUT /serviceaccounts route", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/serviceaccounts/jenkins", strings.NewReader(`{"consumer":"jenkins"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(apiKeyHeader, "expected-key")

		srv.Handler.ServeHTTP(w, req)

		// The path matches the DELETE/HEAD /serviceaccounts/{consumer} pattern, just not for this method.
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("DELETE /serviceaccounts/:consumer requires api key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/serviceaccounts/jenkins", nil)

		srv.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("DELETE /serviceaccounts/:consumer succeeds with valid key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/serviceaccounts/jenkins", nil)
		req.Header.Set(apiKeyHeader, "expected-key")

		srv.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("HEAD /serviceaccounts/:consumer requires api key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodHead, "/serviceaccounts/jenkins", nil)

		srv.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("HEAD /serviceaccounts/:consumer succeeds with valid key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodHead, "/serviceaccounts/jenkins", nil)
		req.Header.Set(apiKeyHeader, "expected-key")

		srv.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("HEAD /serviceaccounts does not require an api key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodHead, "/serviceaccounts", nil)

		srv.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("GET /serviceaccounts does not require an api key either - Kubernetes httpGet probes can only send GET", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/serviceaccounts", nil)

		srv.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
