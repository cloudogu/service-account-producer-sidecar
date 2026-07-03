package serviceaccount

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubManager struct {
	createOrUpdateFunc func(ctx context.Context, consumer string, params map[string]string) (map[string]string, error)
	deleteFunc         func(ctx context.Context, consumer string) error
	existsFunc         func(ctx context.Context, consumer string) (bool, error)
}

func (s *stubManager) CreateOrUpdate(ctx context.Context, consumer string, params map[string]string) (map[string]string, error) {
	return s.createOrUpdateFunc(ctx, consumer, params)
}

func (s *stubManager) Delete(ctx context.Context, consumer string) error {
	return s.deleteFunc(ctx, consumer)
}

func (s *stubManager) Exists(ctx context.Context, consumer string) (bool, error) {
	return s.existsFunc(ctx, consumer)
}

func newTestRequest(method, path, body string) (*http.Request, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req, httptest.NewRecorder()
}

func TestController_CreateOrUpdate(t *testing.T) {
	t.Run("returns 400 on invalid JSON", func(t *testing.T) {
		ctrl := NewController(&stubManager{})
		req, w := newTestRequest(http.MethodPut, "/serviceaccounts/", "not-json")

		ctrl.CreateOrUpdate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 when consumer is empty", func(t *testing.T) {
		ctrl := NewController(&stubManager{})
		req, w := newTestRequest(http.MethodPut, "/serviceaccounts/", `{"consumer":""}`)

		ctrl.CreateOrUpdate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 500 when hook fails", func(t *testing.T) {
		ctrl := NewController(&stubManager{
			createOrUpdateFunc: func(ctx context.Context, consumer string, params map[string]string) (map[string]string, error) {
				return nil, errors.New("boom")
			},
		})
		req, w := newTestRequest(http.MethodPut, "/serviceaccounts/", `{"consumer":"jenkins"}`)

		ctrl.CreateOrUpdate(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("returns 201 with credentials on success", func(t *testing.T) {
		var gotConsumer string
		var gotParams map[string]string
		ctrl := NewController(&stubManager{
			createOrUpdateFunc: func(ctx context.Context, consumer string, params map[string]string) (map[string]string, error) {
				gotConsumer = consumer
				gotParams = params
				return map[string]string{"username": "u", "password": "p"}, nil
			},
		})
		req, w := newTestRequest(http.MethodPut, "/serviceaccounts/", `{"consumer":"jenkins","params":{"permissions":"nx-readonly"}}`)

		ctrl.CreateOrUpdate(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "jenkins", gotConsumer)
		assert.Equal(t, map[string]string{"permissions": "nx-readonly"}, gotParams)

		// The operator's HTTP client decodes the response body directly into map[string]string -
		// no "credentials" wrapper object.
		var body map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, map[string]string{"username": "u", "password": "p"}, body)
	})

	t.Run("returns 204 when hook produces no credentials", func(t *testing.T) {
		ctrl := NewController(&stubManager{
			createOrUpdateFunc: func(ctx context.Context, consumer string, params map[string]string) (map[string]string, error) {
				return map[string]string{}, nil
			},
		})
		req, w := newTestRequest(http.MethodPut, "/serviceaccounts/", `{"consumer":"jenkins"}`)

		ctrl.CreateOrUpdate(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestController_Delete(t *testing.T) {
	t.Run("returns 500 when hook fails", func(t *testing.T) {
		ctrl := NewController(&stubManager{
			deleteFunc: func(ctx context.Context, consumer string) error { return errors.New("boom") },
		})
		req, w := newTestRequest(http.MethodDelete, "/serviceaccounts/jenkins", "")
		req.SetPathValue("consumer", "jenkins")

		ctrl.Delete(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("returns 204 on success", func(t *testing.T) {
		var gotConsumer string
		ctrl := NewController(&stubManager{
			deleteFunc: func(ctx context.Context, consumer string) error {
				gotConsumer = consumer
				return nil
			},
		})
		req, w := newTestRequest(http.MethodDelete, "/serviceaccounts/jenkins", "")
		req.SetPathValue("consumer", "jenkins")

		ctrl.Delete(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "jenkins", gotConsumer)
	})
}

func TestController_Exists(t *testing.T) {
	t.Run("returns 200 when the account exists", func(t *testing.T) {
		ctrl := NewController(&stubManager{
			existsFunc: func(ctx context.Context, consumer string) (bool, error) { return true, nil },
		})
		req, w := newTestRequest(http.MethodHead, "/serviceaccounts/jenkins", "")
		req.SetPathValue("consumer", "jenkins")

		ctrl.Exists(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 when the account does not exist", func(t *testing.T) {
		ctrl := NewController(&stubManager{
			existsFunc: func(ctx context.Context, consumer string) (bool, error) { return false, nil },
		})
		req, w := newTestRequest(http.MethodHead, "/serviceaccounts/jenkins", "")
		req.SetPathValue("consumer", "jenkins")

		ctrl.Exists(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 500 when the hook fails", func(t *testing.T) {
		ctrl := NewController(&stubManager{
			existsFunc: func(ctx context.Context, consumer string) (bool, error) { return false, errors.New("boom") },
		})
		req, w := newTestRequest(http.MethodHead, "/serviceaccounts/jenkins", "")
		req.SetPathValue("consumer", "jenkins")

		ctrl.Exists(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestController_Ready(t *testing.T) {
	t.Run("always returns 204", func(t *testing.T) {
		ctrl := NewController(&stubManager{})
		req, w := newTestRequest(http.MethodHead, "/serviceaccounts", "")

		ctrl.Ready(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
