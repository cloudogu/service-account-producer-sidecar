package serviceaccount

import (
	"crypto/subtle"
	"net/http"
)

const apiKeyHeader = "X-CES-SA-API-KEY"

// ValidateAPIKey wraps next with a check that the X-CES-SA-API-KEY request header matches apiKey,
// per the service-account-operator OpenAPI security scheme.
func ValidateAPIKey(apiKey string, next http.Handler) http.Handler {
	expected := []byte(apiKey)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actual := []byte(r.Header.Get(apiKeyHeader))
		if subtle.ConstantTimeCompare(expected, actual) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or missing " + apiKeyHeader})
			return
		}
		next.ServeHTTP(w, r)
	})
}
