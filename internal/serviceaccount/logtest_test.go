package serviceaccount

import (
	"bytes"
	"log/slog"
	"testing"
)

// captureLogs redirects the slog default logger to a buffer for the duration of the test and
// restores the previous default logger afterwards.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}
