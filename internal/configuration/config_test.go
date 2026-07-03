package configuration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv(apiKeyEnv, "secret")
	t.Setenv(createHookEnv, "/hooks/create.sh")
	t.Setenv(deleteHookEnv, "/hooks/delete.sh")
	t.Setenv(existsHookEnv, "/hooks/exists.sh")
}

func TestReadFromEnv(t *testing.T) {
	t.Run("fails when API_KEY is missing", func(t *testing.T) {
		t.Setenv(createHookEnv, "/hooks/create.sh")
		t.Setenv(deleteHookEnv, "/hooks/delete.sh")
		t.Setenv(existsHookEnv, "/hooks/exists.sh")

		_, err := ReadFromEnv()

		require.Error(t, err)
		assert.ErrorContains(t, err, apiKeyEnv)
	})

	t.Run("fails when CREATE_HOOK is missing", func(t *testing.T) {
		t.Setenv(apiKeyEnv, "secret")
		t.Setenv(deleteHookEnv, "/hooks/delete.sh")
		t.Setenv(existsHookEnv, "/hooks/exists.sh")

		_, err := ReadFromEnv()

		require.Error(t, err)
		assert.ErrorContains(t, err, createHookEnv)
	})

	t.Run("fails when DELETE_HOOK is missing", func(t *testing.T) {
		t.Setenv(apiKeyEnv, "secret")
		t.Setenv(createHookEnv, "/hooks/create.sh")
		t.Setenv(existsHookEnv, "/hooks/exists.sh")

		_, err := ReadFromEnv()

		require.Error(t, err)
		assert.ErrorContains(t, err, deleteHookEnv)
	})

	t.Run("fails when EXISTS_HOOK is missing", func(t *testing.T) {
		t.Setenv(apiKeyEnv, "secret")
		t.Setenv(createHookEnv, "/hooks/create.sh")
		t.Setenv(deleteHookEnv, "/hooks/delete.sh")

		_, err := ReadFromEnv()

		require.Error(t, err)
		assert.ErrorContains(t, err, existsHookEnv)
	})

	t.Run("fails on invalid HOOK_TIMEOUT", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv(hookTimeoutEnv, "not-a-duration")

		_, err := ReadFromEnv()

		require.Error(t, err)
		assert.ErrorContains(t, err, hookTimeoutEnv)
	})

	t.Run("applies defaults", func(t *testing.T) {
		setRequiredEnv(t)

		conf, err := ReadFromEnv()

		require.NoError(t, err)
		assert.Equal(t, "secret", conf.ApiKey)
		assert.Equal(t, "/hooks/create.sh", conf.CreateHook)
		assert.Equal(t, "/hooks/delete.sh", conf.DeleteHook)
		assert.Equal(t, "/hooks/exists.sh", conf.ExistsHook)
		assert.Equal(t, defaultAddr, conf.Addr)
		assert.Equal(t, defaultLogLevel, conf.LogLevel)
		assert.Equal(t, defaultHookTimeout, conf.HookTimeout)
	})

	t.Run("overrides defaults", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv(addrEnv, ":9090")
		t.Setenv(logLevelEnv, "DEBUG")
		t.Setenv(hookTimeoutEnv, "5s")

		conf, err := ReadFromEnv()

		require.NoError(t, err)
		assert.Equal(t, ":9090", conf.Addr)
		assert.Equal(t, "DEBUG", conf.LogLevel)
		assert.Equal(t, 5*time.Second, conf.HookTimeout)
	})
}
