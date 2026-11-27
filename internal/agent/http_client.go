package agent

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/kheguy/collector/internal/retry"
)

// HTTPClient выполняет HTTP запросы
type HTTPClient interface {
	// Do отправляет запрос и возвращает ответ сервера
	Do(*http.Request) (*http.Response, error)
}

// RetryClient повторяет запросы при ошибках установления соединения
type RetryClient struct {
	client HTTPClient
	retry  func(context.Context, func() error, func(error) bool) error
}

// NewRetryClient создаёт HTTP клиент с повторными попытками подключения
func NewRetryClient(client HTTPClient) *RetryClient {
	return &RetryClient{client: client, retry: retry.Do}
}

// Do отправляет запрос и повторяет его при ошибке установления соединения
// Для повторной отправки тела запроса должно быть задано поле GetBody
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

// bodyclose:handled
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
