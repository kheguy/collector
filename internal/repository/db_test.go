package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kheguy/collector/internal/mocks"
	models "github.com/kheguy/collector/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type batchResultsStub struct {
	pgx.BatchResults
	err error
}

func (r *batchResultsStub) Close() error {
	return r.err
}

type batchTxStub struct {
	pgx.Tx
	batch      *pgx.Batch
	results    pgx.BatchResults
	committed  bool
	rolledBack bool
}

func (tx *batchTxStub) SendBatch(_ context.Context, batch *pgx.Batch) pgx.BatchResults {
	tx.batch = batch
	return tx.results
}

func (tx *batchTxStub) Commit(context.Context) error {
	tx.committed = true
	return nil
}

func (tx *batchTxStub) Rollback(context.Context) error {
	tx.rolledBack = true
	return nil
}

func TestDBStorage_Get(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		mType  string
		delta  pgtype.Int8
		value  pgtype.Float8
		result interface{}
	}{
		{
			name:   "gauge",
			mType:  models.Gauge,
			value:  pgtype.Float8{Float64: 12.5, Valid: true},
			result: 12.5,
		},
		{
			name:   "counter",
			mType:  models.Counter,
			delta:  pgtype.Int8{Int64: 7, Valid: true},
			result: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := mocks.NewMockDBPool(t)
			row := mocks.NewMockPGXRow(t)
			pool.EXPECT().
				QueryRow(ctx, "SELECT mtype, delta, value FROM metrics WHERE name = $1", "metric").
				Return(row).
				Once()
			row.EXPECT().
				Scan(mock.Anything, mock.Anything, mock.Anything).
				Run(func(dest ...any) {
					*dest[0].(*string) = tt.mType
					*dest[1].(*pgtype.Int8) = tt.delta
					*dest[2].(*pgtype.Float8) = tt.value
				}).
				Return(nil).
				Once()

			result, err := NewDBStorage(pool).Get(ctx, "metric")

			require.NoError(t, err)
			assert.Equal(t, tt.result, result)
		})
	}
}

func TestDBStorage_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	pool := mocks.NewMockDBPool(t)
	row := mocks.NewMockPGXRow(t)
	pool.EXPECT().
		QueryRow(ctx, "SELECT mtype, delta, value FROM metrics WHERE name = $1", "missing").
		Return(row).
		Once()
	row.EXPECT().
		Scan(mock.Anything, mock.Anything, mock.Anything).
		Return(pgx.ErrNoRows).
		Once()

	result, err := NewDBStorage(pool).Get(ctx, "missing")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDBStorage_Set(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		mType     string
		value     interface{}
		queryPart string
		queryArgs []interface{}
	}{
		{
			name:      "gauge",
			mType:     models.Gauge,
			value:     12.5,
			queryPart: "value = EXCLUDED.value",
			queryArgs: []interface{}{"metric", models.Gauge, 12.5},
		},
		{
			name:      "counter",
			mType:     models.Counter,
			value:     3,
			queryPart: "delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta",
			queryArgs: []interface{}{"metric", models.Counter, int64(3)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := mocks.NewMockDBPool(t)
			query := mock.MatchedBy(func(query string) bool {
				return strings.Contains(query, "ON CONFLICT (name)") && strings.Contains(query, tt.queryPart)
			})
			pool.EXPECT().
				Exec(ctx, query, tt.queryArgs...).
				Return(pgconn.CommandTag{}, nil).
				Once()

			err := NewDBStorage(pool).Set(ctx, "metric", tt.mType, tt.value)

			assert.NoError(t, err)
		})
	}
}

func TestDBStorage_GetAll_ReturnsQueryError(t *testing.T) {
	ctx := context.Background()
	pool := mocks.NewMockDBPool(t)
	pool.EXPECT().
		Query(ctx, "SELECT name, mtype, delta, value FROM metrics").
		Return(nil, assert.AnError).
		Once()

	_, err := NewDBStorage(pool).GetAll(ctx)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestDBStorage_SetBatch_UsesBatchAndPreservesDuplicates(t *testing.T) {
	ctx := context.Background()
	pool := mocks.NewMockDBPool(t)
	results := &batchResultsStub{}
	tx := &batchTxStub{results: results}
	pool.EXPECT().Begin(ctx).Return(tx, nil).Once()

	firstGauge, lastGauge := 12.5, 15.5
	firstDelta, secondDelta := int64(3), int64(4)
	metrics := []models.Metrics{
		{ID: "temperature", MType: models.Gauge, Value: &firstGauge},
		{ID: "requests", MType: models.Counter, Delta: &firstDelta},
		{ID: "temperature", MType: models.Gauge, Value: &lastGauge},
		{ID: "requests", MType: models.Counter, Delta: &secondDelta},
	}

	err := NewDBStorage(pool).SetBatch(ctx, metrics)

	require.NoError(t, err)
	require.NotNil(t, tx.batch)
	require.Len(t, tx.batch.QueuedQueries, 4)
	assert.Equal(t, []interface{}{"temperature", models.Gauge, firstGauge}, tx.batch.QueuedQueries[0].Arguments)
	assert.Equal(t, []interface{}{"requests", models.Counter, firstDelta}, tx.batch.QueuedQueries[1].Arguments)
	assert.Equal(t, []interface{}{"temperature", models.Gauge, lastGauge}, tx.batch.QueuedQueries[2].Arguments)
	assert.Equal(t, []interface{}{"requests", models.Counter, secondDelta}, tx.batch.QueuedQueries[3].Arguments)
	assert.True(t, tx.committed)
	assert.True(t, tx.rolledBack)
}

func TestDBStorage_Clear(t *testing.T) {
	ctx := context.Background()
	pool := mocks.NewMockDBPool(t)
	pool.EXPECT().
		Exec(ctx, "DELETE FROM metrics").
		Return(pgconn.CommandTag{}, nil).
		Once()

	err := NewDBStorage(pool).Clear(ctx)

	assert.NoError(t, err)
}

func TestIsRetriablePostgresError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "connection SQLSTATE", err: &pgconn.PgError{Code: "08006"}, want: true},
		{name: "transaction rollback SQLSTATE", err: &pgconn.PgError{Code: "40001"}, want: true},
		{name: "other SQLSTATE", err: &pgconn.PgError{Code: "23505"}, want: false},
		{name: "canceled", err: context.Canceled, want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRetriablePostgresError(tt.err))
		})
	}
}
