package agent

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/kheguy/collector/internal/retry"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type RetryClient struct {
	client HTTPClient
	retry  func(context.Context, func() error, func(error) bool) error
}

func NewRetryClient(client HTTPClient) *RetryClient {
	return &RetryClient{client: client, retry: retry.Do}
}

func (c *RetryClient) Do(req *http.Request) (*http.Response, error) {
	var response *http.Response
	attemptNumber := 0
	err := c.retry(req.Context(), func() error {
		attempt := req
		if attemptNumber > 0 {
			attempt = req.Clone(req.Context())
			if req.Body != nil {
				if req.GetBody == nil {
					return errors.New("request body cannot be replayed")
				}
				body, err := req.GetBody()
				if err != nil {
					return err
				}
				attempt.Body = body
			}
		}
		attemptNumber++

		var err error
		response, err = sendRequest(c.client, attempt)
		if err != nil && response != nil {
			response.Body.Close()
			response = nil
		}
		return err
	}, isRetriableNetworkError)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func sendRequest(client HTTPClient, req *http.Request) (*http.Response, error) {
	return client.Do(req)
}

func isRetriableNetworkError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial"
}
