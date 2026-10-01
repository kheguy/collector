package middlewares

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kheguy/collector/internal/signature"
)

func TestWithSignatureAcceptsValidRequestAndSignsResponse(t *testing.T) {
	const key = "secret"
	requestBody := []byte(`{"id":"PollCount","type":"counter","delta":1}`)
	responseBody := []byte(`{"status":"ok"}`)

	handler := WithSignature(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(responseBody)
		require.NoError(t, err)
	}))
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(requestBody))
	req.Header.Set(signature.Header, signature.Calculate(requestBody, key))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, responseBody, recorder.Body.Bytes())
	assert.Equal(t, signature.Calculate(responseBody, key), recorder.Header().Get(signature.Header))
}

func TestWithSignatureRejectsInvalidRequest(t *testing.T) {
	const key = "secret"
	handlerCalled := false
	handler := WithSignature(key)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalled = true
	}))
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBufferString("changed"))
	req.Header.Set(signature.Header, signature.Calculate([]byte("original"), key))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.False(t, handlerCalled)
	assert.Equal(t, signature.Calculate(recorder.Body.Bytes(), key), recorder.Header().Get(signature.Header))
}

func TestWithSignatureAcceptsRequestWithoutSignature(t *testing.T) {
	const key = "secret"
	handlerCalled := false
	handler := WithSignature(key)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status":"ok"}`))
		require.NoError(t, err)
	}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBufferString(`{"id":"metric"}`)))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, handlerCalled)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	assert.NotEmpty(t, recorder.Header().Get(signature.Header))
}

func TestWithSignatureDisabledWithoutKey(t *testing.T) {
	handler := WithSignature("")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/update/", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, recorder.Header().Get(signature.Header))
}

func TestSignatureAndGzipCoverTransmittedBodies(t *testing.T) {
	const key = "secret"
	requestBody := []byte(`{"id":"PollCount","type":"counter","delta":1}`)
	compressedRequest := gzipBytes(t, requestBody)
	responseBody := []byte(`{"status":"ok"}`)

	handler := WithSignature(key)(WithGzip(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(responseBody)
		require.NoError(t, err)
	})))
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(compressedRequest))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set(signature.Header, signature.Calculate(compressedRequest, key))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, signature.Calculate(recorder.Body.Bytes(), key), recorder.Header().Get(signature.Header))
	assert.Equal(t, responseBody, gunzipBytes(t, recorder.Body.Bytes()))
}
