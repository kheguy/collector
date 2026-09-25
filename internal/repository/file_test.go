package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileProcessor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")

	processor := NewFileProcessor(path)

	require.NotNil(t, processor)
	assert.Equal(t, path, processor.path)
}

func TestFileProcessor_SaveAndRestore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	processor := NewFileProcessor(path)
	want := map[string]interface{}{
		"temperature": 12.5,
		"requests":    3,
	}

	require.NoError(t, processor.Save(want))
	got, err := processor.Restore()

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFileProcessor_RestoreMissingOrEmptyFile(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		processor := NewFileProcessor(filepath.Join(t.TempDir(), "missing.json"))

		got, err := processor.Restore()

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.json")
		require.NoError(t, os.WriteFile(path, nil, 0o600))
		processor := NewFileProcessor(path)

		got, err := processor.Restore()

		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestFileProcessor_RestoreInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"broken":`), 0o600))

	_, err := NewFileProcessor(path).Restore()

	assert.Error(t, err)
}
