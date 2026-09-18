package retry

import (
	"context"
	"time"
)

var defaultDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

// Вынес отдельно так как не хочется дублировать
func Do(ctx context.Context, operation func() error, isRetriable func(error) bool) error {
	return do(ctx, operation, isRetriable, defaultDelays)
}

func do(
	ctx context.Context,
	operation func() error,
	isRetriable func(error) bool,
	delays []time.Duration,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	err := operation()
	if err == nil {
		return nil
	}

	for _, delay := range delays {
		if !isRetriable(err) {
			return err
		}

		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		err = operation()
		if err == nil {
			return nil
		}
	}

	return err
}
