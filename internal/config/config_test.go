package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAgentReadsKeyFromFlag(t *testing.T) {
	restoreArgs := replaceArgs("agent", "-k=flag-secret")
	t.Cleanup(restoreArgs)
	restoreKey := unsetEnv("KEY")
	t.Cleanup(restoreKey)

	cfg, err := LoadAgent()

	require.NoError(t, err)
	require.Equal(t, "flag-secret", cfg.Key)
}

func TestLoadServerReadsKeyFromEnvironment(t *testing.T) {
	restoreArgs := replaceArgs("server", "-k=flag-secret")
	t.Cleanup(restoreArgs)
	t.Setenv("KEY", "env-secret")

	cfg, err := LoadServer()

	require.NoError(t, err)
	require.Equal(t, "env-secret", cfg.Key)
}

func TestLoadAgentReadsRateLimitFromFlag(t *testing.T) {
	restoreArgs := replaceArgs("agent", "-l=3")
	t.Cleanup(restoreArgs)
	restoreRateLimit := unsetEnv("RATE_LIMIT")
	t.Cleanup(restoreRateLimit)

	cfg, err := LoadAgent()

	require.NoError(t, err)
	require.Equal(t, 3, cfg.RateLimit)
}

func TestLoadAgentReadsRateLimitFromEnvironment(t *testing.T) {
	restoreArgs := replaceArgs("agent", "-l=3")
	t.Cleanup(restoreArgs)
	t.Setenv("RATE_LIMIT", "5")

	cfg, err := LoadAgent()

	require.NoError(t, err)
	require.Equal(t, 5, cfg.RateLimit)
}

func TestLoadAgentRejectsInvalidRateLimit(t *testing.T) {
	restoreArgs := replaceArgs("agent", "-l=0")
	t.Cleanup(restoreArgs)
	restoreRateLimit := unsetEnv("RATE_LIMIT")
	t.Cleanup(restoreRateLimit)

	_, err := LoadAgent()

	require.EqualError(t, err, "rate limit must be at least 1, got 0")
}

func replaceArgs(args ...string) func() {
	previous := os.Args
	os.Args = args
	return func() {
		os.Args = previous
	}
}

func unsetEnv(name string) func() {
	previous, exists := os.LookupEnv(name)
	_ = os.Unsetenv(name)
	return func() {
		if exists {
			_ = os.Setenv(name, previous)
			return
		}
		_ = os.Unsetenv(name)
	}
}
