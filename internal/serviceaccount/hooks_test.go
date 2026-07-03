package serviceaccount

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeScript(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hook.sh")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
	return path
}

func TestHookRunner_CreateOrUpdate(t *testing.T) {
	t.Run("parses key: value stdout lines into credentials", func(t *testing.T) {
		script := writeScript(t, `#!/bin/sh
set -e
echo "some noise without a colon marker"
echo "username: service_account_jenkins_ab12cd"
echo "password: s3cr3t"
`)
		runner := &HookRunner{CreateHook: script, Timeout: time.Second}

		credentials, err := runner.CreateOrUpdate(context.Background(), "jenkins", nil)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"username": "service_account_jenkins_ab12cd",
			"password": "s3cr3t",
		}, credentials)
	})

	t.Run("passes params as named --key=value flags sorted by key, then consumer", func(t *testing.T) {
		script := writeScript(t, `#!/bin/sh
echo "args: $*"
`)
		runner := &HookRunner{CreateHook: script, Timeout: time.Second}

		credentials, err := runner.CreateOrUpdate(context.Background(), "jenkins", map[string]string{
			"permissions":          "nx-readonly",
			"fullAccessRepository": "foo",
		})

		require.NoError(t, err)
		assert.Equal(t, "--fullAccessRepository=foo --permissions=nx-readonly jenkins", credentials["args"])
	})

	t.Run("returns empty credentials when hook prints nothing parseable", func(t *testing.T) {
		script := writeScript(t, `#!/bin/sh
exit 0
`)
		runner := &HookRunner{CreateHook: script, Timeout: time.Second}

		credentials, err := runner.CreateOrUpdate(context.Background(), "jenkins", nil)

		require.NoError(t, err)
		assert.Empty(t, credentials)
	})

	t.Run("returns error on non-zero exit code", func(t *testing.T) {
		script := writeScript(t, `#!/bin/sh
echo "boom" >&2
exit 1
`)
		runner := &HookRunner{CreateHook: script, Timeout: time.Second}

		_, err := runner.CreateOrUpdate(context.Background(), "jenkins", nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "boom")
	})

	t.Run("returns error when hook exceeds timeout", func(t *testing.T) {
		script := writeScript(t, `#!/bin/sh
sleep 5
`)
		runner := &HookRunner{CreateHook: script, Timeout: 10 * time.Millisecond}

		_, err := runner.CreateOrUpdate(context.Background(), "jenkins", nil)

		require.Error(t, err)
	})
}

func TestHookRunner_Delete(t *testing.T) {
	t.Run("succeeds on exit code 0", func(t *testing.T) {
		script := writeScript(t, `#!/bin/sh
exit 0
`)
		runner := &HookRunner{DeleteHook: script, Timeout: time.Second}

		err := runner.Delete(context.Background(), "jenkins")

		assert.NoError(t, err)
	})

	t.Run("returns error on non-zero exit code", func(t *testing.T) {
		script := writeScript(t, `#!/bin/sh
exit 1
`)
		runner := &HookRunner{DeleteHook: script, Timeout: time.Second}

		err := runner.Delete(context.Background(), "jenkins")

		assert.Error(t, err)
	})
}

func TestHookRunner_Exists(t *testing.T) {
	t.Run("exit code 0 means the account exists", func(t *testing.T) {
		script := writeScript(t, `#!/bin/sh
exit 0
`)
		runner := &HookRunner{ExistsHook: script, Timeout: time.Second}

		exists, err := runner.Exists(context.Background(), "jenkins")

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("exit code 1 means the account does not exist", func(t *testing.T) {
		script := writeScript(t, `#!/bin/sh
exit 1
`)
		runner := &HookRunner{ExistsHook: script, Timeout: time.Second}

		exists, err := runner.Exists(context.Background(), "jenkins")

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("any other exit code is an error", func(t *testing.T) {
		script := writeScript(t, `#!/bin/sh
echo "boom" >&2
exit 2
`)
		runner := &HookRunner{ExistsHook: script, Timeout: time.Second}

		_, err := runner.Exists(context.Background(), "jenkins")

		require.Error(t, err)
		assert.ErrorContains(t, err, "boom")
	})

	t.Run("a failure to execute the hook is an error", func(t *testing.T) {
		runner := &HookRunner{ExistsHook: "/does/not/exist.sh", Timeout: time.Second}

		_, err := runner.Exists(context.Background(), "jenkins")

		require.Error(t, err)
	})
}
