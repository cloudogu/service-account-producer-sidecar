package serviceaccount

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HookRunner is a manager that fulfills service-account requests by executing configured shell-script hooks.
// Params are passed as named "--key=value" flags followed by the consumer name as the final, unnamed argument.
//
// CreateHook/DeleteHook signal success via exit code 0, any other exit code is a failure.
// ExistsHook exit code 0 means the service account exists, exit code 1 means it does not, any other exit code is treated as an error.
//
// Credentials are extracted from the create hook's stdout: every line of the form "key: value"
// becomes an entry in the returned credentials map. Lines that don't match this format are
// ignored, so hooks may freely log additional diagnostic output.
type HookRunner struct {
	CreateHook string
	DeleteHook string
	ExistsHook string
	Timeout    time.Duration
}

// CreateOrUpdate runs the configured create hook for the given consumer, params and behaviorParams.
// behaviorParams (e.g. rotateServiceAccountNow) are passed as "--behavior-key=value" flags,
// distinct from the "--key=value" domain params, so a hook can tell the two apart.
func (h *HookRunner) CreateOrUpdate(ctx context.Context, consumer string, params map[string]string, behaviorParams map[string]any) (map[string]string, error) {
	flags := append(paramsToFlags(params), behaviorParamsToFlags(behaviorParams)...)
	stdout, _, err := h.run(ctx, h.CreateHook, consumer, flags)
	if err != nil {
		return nil, err
	}
	return parseCredentials(stdout), nil
}

// Delete runs the configured delete hook for the given consumer.
func (h *HookRunner) Delete(ctx context.Context, consumer string) error {
	_, _, err := h.run(ctx, h.DeleteHook, consumer, nil)
	return err
}

// Exists runs the configured exists hook for the given consumer. Exit code 0 means the service
// account exists, exit code 1 means it does not; any other outcome is returned as an error.
func (h *HookRunner) Exists(ctx context.Context, consumer string) (bool, error) {
	_, exitCode, err := h.run(ctx, h.ExistsHook, consumer, nil)
	switch {
	case err == nil:
		return true, nil
	case exitCode == 1:
		return false, nil
	default:
		return false, err
	}
}

// run executes hook and returns its stdout, exit code, and an error for anything but an ordinary
// exit-code-0 completion (including a non-zero exit code, wrapped as *exec.ExitError so callers
// can inspect the exit code via errors.As).
func (h *HookRunner) run(ctx context.Context, hook string, consumer string, params []string) (string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, h.Timeout)
	defer cancel()

	args := append(append([]string{}, params...), consumer)
	slog.Debug("invoking hook", "hook", hook, "consumer", consumer, "params", params)

	cmd := exec.CommandContext(ctx, hook, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			slog.Debug("hook exited with an error", "hook", hook, "consumer", consumer, "duration", duration, "exitCode", exitErr.ExitCode())
			return "", exitErr.ExitCode(), fmt.Errorf("hook %q failed: %w (stderr: %s)", hook, err, strings.TrimSpace(stderr.String()))
		}

		slog.Debug("hook failed to run", "hook", hook, "consumer", consumer, "duration", duration, "err", err)
		return "", -1, fmt.Errorf("hook %q failed: %w (stderr: %s)", hook, err, strings.TrimSpace(stderr.String()))
	}

	slog.Debug("hook completed successfully", "hook", hook, "consumer", consumer, "duration", duration)
	return stdout.String(), 0, nil
}

// paramsToFlags turns a params map into named "--key=value" flags, sorted by key
func paramsToFlags(params map[string]string) []string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	flags := make([]string, 0, len(params))
	for _, k := range keys {
		flags = append(flags, "--"+k+"="+params[k])
	}

	return flags
}

// behaviorParamsToFlags turns a behaviorParams map into named "--behavior-key=value" flags, sorted
// by key. The "behavior-" prefix keeps these distinguishable from domain params on the hook's
// command line, since the two maps are unrelated and may otherwise collide on key names.
func behaviorParamsToFlags(behaviorParams map[string]any) []string {
	keys := make([]string, 0, len(behaviorParams))
	for k := range behaviorParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	flags := make([]string, 0, len(behaviorParams))
	for _, k := range keys {
		flags = append(flags, "--behavior-"+k+"="+stringifyBehaviorValue(behaviorParams[k]))
	}

	return flags
}

// stringifyBehaviorValue renders a decoded JSON value as a hook-CLI-friendly string.
func stringifyBehaviorValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case float64: // encoding/json decodes all JSON numbers into float64 when the target is `any`.
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func parseCredentials(output string) map[string]string {
	credentials := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		credentials[key] = value
	}
	return credentials
}
