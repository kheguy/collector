package database

import (
	"context"
	"testing"

	"github.com/kheguy/collector/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type clientPoolMock struct {
	*mocks.MockDBPool
	pinger *mocks.MockPinger
	closer mock.Mock
}

func newClientPoolMock(t *testing.T) *clientPoolMock {
	t.Helper()

	dbPool := &clientPoolMock{
		MockDBPool: mocks.NewMockDBPool(t),
		pinger:     mocks.NewMockPinger(t),
	}
	t.Cleanup(func() {
		dbPool.closer.AssertExpectations(t)
	})

	return dbPool
}

func (p *clientPoolMock) Ping(ctx context.Context) error {
	return p.pinger.Ping(ctx)
}

func (p *clientPoolMock) Close() {
	p.closer.Called()
}

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	dsn := "postgres://localhost/collector"

	t.Run("success", func(t *testing.T) {
		dbPool := newClientPoolMock(t)
		dbPool.pinger.EXPECT().Ping(ctx).Return(nil).Once()

		client, err := newClient(
			ctx,
			dsn,
			func(actualDSN string) error {
				assert.Equal(t, dsn, actualDSN)
				return nil
			},
			func(actualCtx context.Context, actualDSN string) (pool, error) {
				assert.Equal(t, ctx, actualCtx)
				assert.Equal(t, dsn, actualDSN)
				return dbPool, nil
			},
		)

		require.NoError(t, err)
		assert.Same(t, dbPool, client.pool)
		assert.NotNil(t, client.Storage())
	})

	t.Run("migration error", func(t *testing.T) {
		factoryCalled := false

		client, err := newClient(
			ctx,
			dsn,
			func(string) error {
				return assert.AnError
			},
			func(context.Context, string) (pool, error) {
				factoryCalled = true
				return nil, nil
			},
		)

		assert.Nil(t, client)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "apply database migrations error")
		assert.False(t, factoryCalled)
	})

	t.Run("pool creation error", func(t *testing.T) {
		client, err := newClient(
			ctx,
			dsn,
			func(string) error {
				return nil
			},
			func(context.Context, string) (pool, error) {
				return nil, assert.AnError
			},
		)

		assert.Nil(t, client)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "create database pool error")
	})

	t.Run("ping error", func(t *testing.T) {
		dbPool := newClientPoolMock(t)
		dbPool.pinger.EXPECT().Ping(ctx).Return(assert.AnError).Once()
		dbPool.closer.On("Close").Return().Once()

		client, err := newClient(
			ctx,
			dsn,
			func(string) error {
				return nil
			},
			func(context.Context, string) (pool, error) {
				return dbPool, nil
			},
		)

		assert.Nil(t, client)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "ping database error")
	})
}

func TestClient_Ping(t *testing.T) {
	ctx := context.Background()
	dbPool := newClientPoolMock(t)
	dbPool.pinger.EXPECT().Ping(ctx).Return(assert.AnError).Once()
	client := &Client{pool: dbPool}

	err := client.Ping(ctx)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestClient_Close(t *testing.T) {
	dbPool := newClientPoolMock(t)
	dbPool.closer.On("Close").Return().Once()
	client := &Client{pool: dbPool}

	client.Close()
}
