package serviceaccount

import "net/http"

// CreateServer wires the /serviceaccounts routes (auth middleware + handlers) and returns an
// *http.Server listening on addr, ready for ListenAndServe/Shutdown.
//
// Route shapes (path, not trailing-slash) match the real service-account-operator HTTP client
// (internal/producer/http_client.go), which calls PUT/HEAD against the endpoint verbatim (no
// trailing slash) and joins the consumer onto the endpoint for DELETE/HEAD-by-consumer.
func CreateServer(addr string, apiKey string, manager manager) *http.Server {
	ctrl := NewController(manager)

	protected := http.NewServeMux()
	protected.HandleFunc("PUT /serviceaccounts", ctrl.CreateOrUpdate)
	protected.HandleFunc("DELETE /serviceaccounts/{consumer}", ctrl.Delete)
	protected.HandleFunc("HEAD /serviceaccounts/{consumer}", ctrl.Exists)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /serviceaccounts", ctrl.Ready)
	mux.HandleFunc("HEAD /serviceaccounts", ctrl.Ready)
	mux.Handle("/", ValidateAPIKey(apiKey, protected))

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
