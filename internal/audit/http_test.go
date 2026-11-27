package audit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPObserver_PostsEvent(t *testing.T) {
	event := Event{Timestamp: 12345678, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.168.0.42"}
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/audit?source=collector", r.URL.String())
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		data, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		received <- string(data)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	observer, err := NewHTTPObserver(server.URL + "/audit?source=collector")
	require.NoError(t, err)

	require.NoError(t, observer.Notify(context.Background(), event))
	assert.JSONEq(t, `{"ts":12345678,"metrics":["Alloc","Frees"],"ip_address":"192.168.0.42"}`, <-received)
}

func TestHTTPObserver_ReceiverError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	observer, err := NewHTTPObserver(server.URL)
	require.NoError(t, err)

	assert.Error(t, observer.Notify(context.Background(), Event{}))
}

func TestHTTPObserver_CanceledRequest(t *testing.T) {
	observer, err := NewHTTPObserver("http://127.0.0.1:1/audit")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.ErrorIs(t, observer.Notify(ctx, Event{}), context.Canceled)
}

func TestHTTPObserver_RejectsRedirect(t *testing.T) {
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		if r.URL.Path == "/audit" {
			w.Header().Set("Location", "/redirected")
			w.WriteHeader(http.StatusFound)
		}
	}))
	defer server.Close()
	observer, err := NewHTTPObserver(server.URL + "/audit")
	require.NoError(t, err)

	assert.Error(t, observer.Notify(context.Background(), Event{}))
	assert.Len(t, requests, 1)
	assert.Equal(t, "/audit", <-requests)
}

func TestNewHTTPObserver_InvalidURL(t *testing.T) {
	for _, address := range []string{"/audit", "ftp://receiver/audit", "http:///audit", "://invalid"} {
		t.Run(address, func(t *testing.T) {
			_, err := NewHTTPObserver(address)
			assert.Error(t, err)
		})
	}
}
