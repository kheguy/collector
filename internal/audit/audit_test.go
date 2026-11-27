package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type observerFunc func(context.Context, Event) error

func (f observerFunc) Notify(ctx context.Context, event Event) error {
	return f(ctx, event)
}

func TestPublisher_Notify(t *testing.T) {
	event := Event{Timestamp: 12345678, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.168.0.42"}
	firstErr := errors.New("file failed")
	lastErr := errors.New("HTTP failed")
	var received []Event
	observer := func(err error) Observer {
		return observerFunc(func(_ context.Context, event Event) error {
			received = append(received, event)
			return err
		})
	}
	publisher := NewPublisher(observer(firstErr), observer(nil), observer(lastErr))

	err := publisher.Notify(context.Background(), event)

	assert.ErrorIs(t, err, firstErr)
	assert.ErrorIs(t, err, lastErr)
	assert.Equal(t, []Event{event, event, event}, received)
}
