package agent

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompress_ResultsAreIndependent(t *testing.T) {
	data := []string{
		`{"id":"PollCount","type":"counter","delta":100}`,
		`{"id":"Alloc","type":"gauge","value":16777216}`,
	}
	compressed := make([][]byte, len(data))
	for i, body := range data {
		var err error
		compressed[i], err = compress([]byte(body))
		require.NoError(t, err)
	}

	for i, body := range compressed {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		require.NoError(t, err)
		got, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		require.Equal(t, data[i], string(got))
	}
}

func TestCompress_Parallel(t *testing.T) {
	for _, name := range []string{"Alloc", "HeapAlloc", "Sys", "CPUutilization1"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data := bytes.Repeat([]byte(name), 1000)
			body, err := compress(data)
			require.NoError(t, err)
			reader, err := gzip.NewReader(bytes.NewReader(body))
			require.NoError(t, err)
			got, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.NoError(t, reader.Close())
			require.Equal(t, data, got)
		})
	}
}
