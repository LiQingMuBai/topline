package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetEnvLangList(t *testing.T) {
	const envKey = "SUPPORTED_LANGS"

	oldValue, existed := os.LookupEnv(envKey)
	defer func() {
		if existed {
			_ = os.Setenv(envKey, oldValue)
			return
		}
		_ = os.Unsetenv(envKey)
	}()

	require.NoError(t, os.Setenv(envKey, " zh, en ,ZH, fr "))
	require.Equal(t, []string{"zh", "en", "fr"}, getEnvLangList(envKey, "zh"))

	require.NoError(t, os.Setenv(envKey, ""))
	require.Equal(t, []string{"zh"}, getEnvLangList(envKey, "zh"))
}
