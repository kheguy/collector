package agent

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryClient_Do(t *testing.T) {
	ctx := context.Background()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://collector.test", bytes.NewBufferString("metrics"))
	require.NoError(t, err)

	attempts := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		body, readErr := io.ReadAll(req.Body)
		require.NoError(t, readErr)
		assert.Equal(t, "metrics", string(body))
		if attempts == 1 {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})}
	retryClient := &RetryClient{
		client: client,
		retry: func(_ context.Context, operation func() error, isRetriable func(error) bool) error {
			err := operation()
			if err != nil && isRetriable(err) {
				return operation()
			}
			return err
		},
	}

	response, err := retryClient.Do(request)

	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, 2, attempts)
}

func TestIsRetriableNetworkError(t *testing.T) {
	networkError := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "network error", err: networkError, want: true},
		{name: "wrapped network error", err: &url.Error{Op: "Post", URL: "http://collector.test", Err: networkError}, want: true},
		{name: "write error", err: &net.OpError{Op: "write", Net: "tcp", Err: errors.New("connection reset")}, want: false},
		{name: "generic error", err: errors.New("invalid request"), want: false},
		{name: "canceled", err: context.Canceled, want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRetriableNetworkError(tt.err))
		})
	}
}
