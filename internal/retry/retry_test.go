package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDo_RetriableErrorUsesAllAttempts(t *testing.T) {
	expectedErr := errors.New("temporary error")
	attempts := 0

	err := do(
		context.Background(),
		func() error {
			attempts++
			return expectedErr
		},
		func(error) bool { return true },
		[]time.Duration{0, 0, 0},
	)

	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 4, attempts)
}

func TestDo_NonRetriableErrorStopsImmediately(t *testing.T) {
	expectedErr := errors.New("permanent error")
	attempts := 0

	err := do(
		context.Background(),
		func() error {
			attempts++
			return expectedErr
		},
		func(error) bool { return false },
		[]time.Duration{0, 0, 0},
	)

	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, attempts)
}

func TestDo_ContextCancellationInterruptsWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	err := do(
		ctx,
		func() error {
			attempts++
			cancel()
			return errors.New("temporary error")
		},
		func(error) bool { return true },
		[]time.Duration{time.Hour},
	)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, attempts)
}
