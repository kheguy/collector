package middlewares

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestWithGzipCompressesSequentialResponses(t *testing.T) {
	bodies := []string{
		`{"id":"Alloc","type":"gauge","value":16777216}`,
		`{"id":"PollCount","type":"counter","delta":7}`,
	}
	handler := WithGzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := io.Copy(w, r.Body)
		require.NoError(t, err)
	}))

	responses := make([]*httptest.ResponseRecorder, len(bodies))
	for i, body := range bodies {
		req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(body))
		req.Header.Set("Accept-Encoding", "gzip")
		responses[i] = httptest.NewRecorder()
		handler.ServeHTTP(responses[i], req)
	}

	for i, response := range responses {
		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, "gzip", response.Header().Get("Content-Encoding"))
		require.Equal(t, bodies[i], string(gunzipBytes(t, response.Body.Bytes())))
	}
}

func TestWithGzipCompressesParallelResponses(t *testing.T) {
	for _, name := range []string{"Alloc", "HeapAlloc", "Sys", "CPUutilization1"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			body := fmt.Sprintf(`{"id":"%s","type":"gauge","value":12.5}`, name)
			handler := WithGzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, err := io.Copy(w, r.Body)
				require.NoError(t, err)
			}))
			req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(body))
			req.Header.Set("Accept-Encoding", "gzip")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, req)

			require.Equal(t, http.StatusOK, response.Code)
			require.Equal(t, "gzip", response.Header().Get("Content-Encoding"))
			require.Equal(t, body, string(gunzipBytes(t, response.Body.Bytes())))
		})
	}
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
