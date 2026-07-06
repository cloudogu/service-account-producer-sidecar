package serviceaccount

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
)

// manager fulfills service-account requests. HookRunner is the production implementation.
type manager interface {
	CreateOrUpdate(ctx context.Context, consumer string, params map[string]string) (map[string]string, error)
	Delete(ctx context.Context, consumer string) error
	Exists(ctx context.Context, consumer string) (bool, error)
}

// Controller implements the HTTP handlers of the service-account-operator's HTTP client
// (github.com/cloudogu/service-account-operator/internal/producer/http_client.go).
type Controller struct {
	manager manager
}

// NewController creates a Controller backed by the given manager.
func NewController(manager manager) *Controller {
	return &Controller{manager: manager}
}

type createOrUpdateRequest struct {
	Consumer string            `json:"consumer"`
	Params   map[string]string `json:"params"`
}

// CreateOrUpdate handles PUT /serviceaccounts. On success it returns 201 with the credentials
// produced by the create hook as the raw top-level JSON body (the operator decodes the response
// directly into map[string]string), or 204 if the hook succeeded but produced no credentials.
func (ctrl *Controller) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	var request createOrUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		slog.Warn("rejecting create-or-update request: invalid JSON body", "err", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if request.Consumer == "" {
		slog.Warn("rejecting create-or-update request: consumer must not be empty")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "consumer must not be empty"})
		return
	}

	credentials, err := ctrl.manager.CreateOrUpdate(r.Context(), request.Consumer, request.Params)
	if err != nil {
		slog.Error("create-or-update hook failed", "consumer", request.Consumer, "params", request.Params, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if len(credentials) == 0 {
		slog.Info("service account created/updated, hook returned no credentials", "consumer", request.Consumer)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	slog.Info("service account created/updated", "consumer", request.Consumer, "credentialKeys", sortedKeys(credentials))
	writeJSON(w, http.StatusCreated, credentials)
}

// Delete handles DELETE /serviceaccounts/{consumer}.
func (ctrl *Controller) Delete(w http.ResponseWriter, r *http.Request) {
	consumer := r.PathValue("consumer")
	if consumer == "" {
		slog.Warn("rejecting delete request: consumer must not be empty")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "consumer must not be empty"})
		return
	}

	if err := ctrl.manager.Delete(r.Context(), consumer); err != nil {
		slog.Error("delete hook failed", "consumer", consumer, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	slog.Info("service account deleted", "consumer", consumer)
	w.WriteHeader(http.StatusNoContent)
}

// Exists handles HEAD /serviceaccounts/{consumer}. It responds 200 if a service account exists
// for the consumer, 404 if not - the operator's Exists() call treats any other status as an error.
func (ctrl *Controller) Exists(w http.ResponseWriter, r *http.Request) {
	consumer := r.PathValue("consumer")
	if consumer == "" {
		slog.Warn("rejecting exists request: consumer must not be empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	exists, err := ctrl.manager.Exists(r.Context(), consumer)
	if err != nil {
		slog.Error("exists hook failed", "consumer", consumer, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	slog.Debug("checked service account existence", "consumer", consumer, "exists", exists)

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Ready handles HEAD /serviceaccounts. It always responds 204 - the operator's Ready() call only
// treats 401 and 5xx as not-ready, so this simply confirms the process is up and authenticated.
func (ctrl *Controller) Ready(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
