package service

import (
	"context"
	"testing"

	"github.com/kheguy/collector/internal/mocks"
	models "github.com/kheguy/collector/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeNewMetricsService(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	svc := MakeNewMetricsService(storageMock)

	assert.NotNil(t, svc)
	assert.Equal(t, storageMock, svc.storage)
}

func TestGetMetric_Int(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get(ctx, "some_metrics").Return(111111, nil).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetMetric(ctx, "some_metrics")

	require.NoError(t, err)
	assert.Equal(t, "111111", result)
}

func TestGetMetric_Float64(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get(ctx, "some_metrics").Return(11.111, nil).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetMetric(ctx, "some_metrics")

	require.NoError(t, err)
	assert.Equal(t, "11.111", result)
}

func TestGetMetric_NotFound(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get(ctx, "what??????????").Return(nil, nil).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetMetric(ctx, "what??????????")

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetMetric_StorageError(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get(ctx, "some_metrics").Return(nil, assert.AnError).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetMetric(ctx, "some_metrics")

	assert.Empty(t, result)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestGetRawMetric(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get(ctx, "PollCount").Return(3, nil).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetRawMetric(ctx, "PollCount")

	require.NoError(t, err)
	assert.Equal(t, 3, result)
}

func TestGetAllMetrics(t *testing.T) {
	ctx := context.Background()
	expected := map[string]interface{}{
		"a": 1.1,
		"b": 2,
	}
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().GetAll(ctx).Return(expected, nil).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetAllMetrics(ctx)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestUpdateMetrics_Gauge_Success(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Set(ctx, "oneone", models.Gauge, 11.11111).Return(nil).Once()
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics(ctx, "oneone", models.Gauge, "11.11111")

	assert.NoError(t, err)
}

func TestUpdateMetrics_Counter_Success(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Set(ctx, "requests", models.Counter, 11).Return(nil).Once()
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics(ctx, "requests", models.Counter, "11")

	assert.NoError(t, err)
}

func TestUpdateMetrics_Counter_InvalidString(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics(context.Background(), "requests", models.Counter, "NaN")

	assert.ErrorIs(t, err, errParseInt)
}

func TestUpdateMetrics_UnknownType(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics(context.Background(), "test", "what????????", "123")

	assert.ErrorIs(t, err, errUnknownType)
	assert.ErrorIs(t, err, ErrInvalidMetric)
}

func TestUpdateMetricsBatch_StorageError(t *testing.T) {
	ctx := context.Background()
	gauge := 12.5
	metrics := []models.Metrics{{ID: "temperature", MType: models.Gauge, Value: &gauge}}
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().SetBatch(ctx, metrics).Return(assert.AnError).Once()
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetricsBatch(ctx, metrics)

	assert.ErrorIs(t, err, assert.AnError)
	assert.NotErrorIs(t, err, ErrInvalidMetric)
}

func TestUpdateMetricsBatch_InvalidMetric(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetricsBatch(context.Background(), []models.Metrics{{ID: "requests", MType: models.Counter}})

	assert.ErrorIs(t, err, ErrInvalidMetric)
}
