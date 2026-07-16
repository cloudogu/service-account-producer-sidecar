// Package configuration reads the runtime configuration of the sidecar from environment variables.
package configuration

import (
	"fmt"
	"os"
	"time"
)

const (
	apiKeyEnv      = "API_KEY"
	createHookEnv  = "CREATE_HOOK"
	deleteHookEnv  = "DELETE_HOOK"
	existsHookEnv  = "EXISTS_HOOK"
	addrEnv        = "ADDR"
	logLevelEnv    = "LOG_LEVEL"
	hookTimeoutEnv = "HOOK_TIMEOUT"

	defaultAddr        = ":8080"
	defaultLogLevel    = "INFO"
	defaultHookTimeout = 30 * time.Second
)

// Configuration holds everything the sidecar needs to serve the service-account HTTP API.
type Configuration struct {
	// ApiKey is compared against the X-CES-SA-API-KEY request header.
	ApiKey string
	// CreateHook is the executable invoked for PUT requests (create-or-update).
	CreateHook string
	// DeleteHook is the executable invoked for DELETE requests.
	DeleteHook string
	// ExistsHook is the executable invoked for HEAD requests against a specific consumer.
	ExistsHook string
	// Addr is the listen address of the HTTP server, e.g. ":8080".
	Addr string
	// LogLevel is a slog.Level compatible string (DEBUG, INFO, WARN, ERROR).
	LogLevel string
	// HookTimeout bounds how long a single hook invocation may run.
	HookTimeout time.Duration
}

// ReadFromEnv reads the Configuration from environment variables. ApiKey, CreateHook, DeleteHook
// and ExistsHook are required; the remaining fields fall back to sane defaults.
func ReadFromEnv() (Configuration, error) {
	conf := Configuration{
		Addr:        defaultAddr,
		LogLevel:    defaultLogLevel,
		HookTimeout: defaultHookTimeout,
	}

	conf.ApiKey = os.Getenv(apiKeyEnv)
	if conf.ApiKey == "" {
		return conf, fmt.Errorf("environment variable %s is not set", apiKeyEnv)
	}

	conf.CreateHook = os.Getenv(createHookEnv)
	if conf.CreateHook == "" {
		return conf, fmt.Errorf("environment variable %s is not set", createHookEnv)
	}

	conf.DeleteHook = os.Getenv(deleteHookEnv)
	if conf.DeleteHook == "" {
		return conf, fmt.Errorf("environment variable %s is not set", deleteHookEnv)
	}

	conf.ExistsHook = os.Getenv(existsHookEnv)
	if conf.ExistsHook == "" {
		return conf, fmt.Errorf("environment variable %s is not set", existsHookEnv)
	}

	if v := os.Getenv(addrEnv); v != "" {
		conf.Addr = v
	}

	if v := os.Getenv(logLevelEnv); v != "" {
		conf.LogLevel = v
	}

	if v := os.Getenv(hookTimeoutEnv); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return conf, fmt.Errorf("environment variable %s is not a valid duration: %w", hookTimeoutEnv, err)
		}
		conf.HookTimeout = d
	}

	return conf, nil
}
