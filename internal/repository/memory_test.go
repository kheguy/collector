package repository

import (
	"context"
	"testing"

	models "github.com/kheguy/collector/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()

	require.NotNil(t, storage)
	assert.NotNil(t, storage.data)
	assert.Empty(t, storage.data)
}

func TestMemoryStorage_SetAndGet(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		metricType string
		values     []interface{}
		want       interface{}
	}{
		{
			name:       "gauge is overwritten",
			metricName: "temperature",
			metricType: models.Gauge,
			values:     []interface{}{1.11, 3.33},
			want:       3.33,
		},
		{
			name:       "counter is accumulated",
			metricName: "requests",
			metricType: models.Counter,
			values:     []interface{}{2, 1},
			want:       3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStorage()
			ctx := context.Background()

			for _, value := range tt.values {
				require.NoError(t, storage.Set(ctx, tt.metricName, tt.metricType, value))
			}

			got, err := storage.Get(ctx, tt.metricName)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMemoryStorage_SetBatch_MixedAndDuplicateMetrics(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	require.NoError(t, storage.Set(ctx, "requests", models.Counter, 5))

	firstGauge, lastGauge := 1.5, 2.5
	firstDelta, secondDelta := int64(2), int64(3)
	metrics := []models.Metrics{
		{ID: "temperature", MType: models.Gauge, Value: &firstGauge},
		{ID: "requests", MType: models.Counter, Delta: &firstDelta},
		{ID: "temperature", MType: models.Gauge, Value: &lastGauge},
		{ID: "requests", MType: models.Counter, Delta: &secondDelta},
	}

	require.NoError(t, storage.SetBatch(ctx, metrics))
	got, err := storage.GetAll(ctx)

	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"temperature": 2.5,
		"requests":    10,
	}, got)
}

func TestMemoryStorage_SetBatch_ValidationIsAtomic(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	gauge := 1.5
	metrics := []models.Metrics{
		{ID: "temperature", MType: models.Gauge, Value: &gauge},
		{ID: "requests", MType: models.Counter},
	}

	err := storage.SetBatch(ctx, metrics)

	assert.EqualError(t, err, "counter has no delta")
	got, getErr := storage.GetAll(ctx)
	require.NoError(t, getErr)
	assert.Empty(t, got)
}

func TestMemoryStorage_Get_NotFound(t *testing.T) {
	storage := NewMemoryStorage()

	got, err := storage.Get(context.Background(), "missing")

	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMemoryStorage_GetAll_ReturnsCopy(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	require.NoError(t, storage.Set(ctx, "gauge", models.Gauge, 1.1))
	require.NoError(t, storage.Set(ctx, "counter", models.Counter, 2))

	got, err := storage.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"gauge":   1.1,
		"counter": 2,
	}, got)

	got["gauge"] = 9.9
	original, err := storage.Get(ctx, "gauge")
	require.NoError(t, err)
	assert.Equal(t, 1.1, original)
}

func TestMemoryStorage_SetAllAndClear(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()
	data := map[string]interface{}{
		"gauge":   2.5,
		"counter": 7,
	}

	require.NoError(t, storage.SetAll(ctx, data))
	got, err := storage.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, data, got)

	require.NoError(t, storage.Clear(ctx))
	got, err = storage.GetAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, got)
}
