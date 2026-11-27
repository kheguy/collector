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

func TestLoadServerAuditConfig(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		env       map[string]string
		auditFile string
		auditURL  string
	}{
		{
			name: "disabled by default",
		},
		{
			name:      "flags",
			args:      []string{"--audit-file=/tmp/audit.jsonl", "--audit-url=http://localhost:8081/audit"},
			auditFile: "/tmp/audit.jsonl",
			auditURL:  "http://localhost:8081/audit",
		},
		{
			name: "environment overrides flags",
			args: []string{"--audit-file=/tmp/flag-audit.jsonl", "--audit-url=http://localhost:8081/audit"},
			env: map[string]string{
				"AUDIT_FILE": "/tmp/env-audit.jsonl",
				"AUDIT_URL":  "http://localhost:8082/audit",
			},
			auditFile: "/tmp/env-audit.jsonl",
			auditURL:  "http://localhost:8082/audit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(replaceArgs(append([]string{"server"}, tt.args...)...))
			t.Cleanup(unsetEnv("AUDIT_FILE"))
			t.Cleanup(unsetEnv("AUDIT_URL"))
			for name, value := range tt.env {
				t.Setenv(name, value)
			}

			cfg, err := LoadServer()

			require.NoError(t, err)
			require.Equal(t, tt.auditFile, cfg.AuditFile)
			require.Equal(t, tt.auditURL, cfg.AuditURL)
		})
	}
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
