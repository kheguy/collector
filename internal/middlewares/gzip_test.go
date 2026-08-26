package middlewares

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithGzipDecompressesRequestAndCompressesJSONResponse(t *testing.T) {
	requestBody := []byte(`{"id":"PollCount","type":"counter","delta":1}`)
	compressedBody := gzipBytes(t, requestBody)

	handler := WithGzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, requestBody, body)

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(body)
		require.NoError(t, err)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(compressedBody))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "gzip", recorder.Header().Get("Content-Encoding"))
	require.Equal(t, requestBody, gunzipBytes(t, recorder.Body.Bytes()))
}

func TestWithGzipCompressesHTMLResponse(t *testing.T) {
	handler := WithGzip(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := w.Write([]byte("<h1>Metrics</h1>"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(t, "gzip", recorder.Header().Get("Content-Encoding"))
	require.Equal(t, []byte("<h1>Metrics</h1>"), gunzipBytes(t, recorder.Body.Bytes()))
}

func TestWithGzipDoesNotCompressUnsupportedContentType(t *testing.T) {
	handler := WithGzip(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Empty(t, recorder.Header().Get("Content-Encoding"))
	require.Equal(t, "OK", recorder.Body.String())
}

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	_, err := writer.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func gunzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer reader.Close()
	result, err := io.ReadAll(reader)
	require.NoError(t, err)
	return result
}
