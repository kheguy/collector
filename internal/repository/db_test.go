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

			result, err := NewDBStorage(pool).Get("metric", ctx)

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

	result, err := NewDBStorage(pool).Get("missing", ctx)

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

			err := NewDBStorage(pool).Set("metric", tt.mType, tt.value, ctx)

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
