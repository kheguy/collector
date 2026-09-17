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
	storageMock.EXPECT().Get("some_metrics", ctx).Return(111111, nil).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetMetric("some_metrics", ctx)

	require.NoError(t, err)
	assert.Equal(t, "111111", result)
}

func TestGetMetric_Float64(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get("some_metrics", ctx).Return(11.111, nil).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetMetric("some_metrics", ctx)

	require.NoError(t, err)
	assert.Equal(t, "11.111", result)
}

func TestGetMetric_NotFound(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get("what??????????", ctx).Return(nil, nil).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetMetric("what??????????", ctx)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetMetric_StorageError(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get("some_metrics", ctx).Return(nil, assert.AnError).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetMetric("some_metrics", ctx)

	assert.Empty(t, result)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestGetRawMetric(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get("PollCount", ctx).Return(3, nil).Once()
	svc := MakeNewMetricsService(storageMock)

	result, err := svc.GetRawMetric("PollCount", ctx)

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
	storageMock.EXPECT().Set("oneone", models.Gauge, 11.11111, ctx).Return(nil).Once()
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics("oneone", models.Gauge, "11.11111", ctx)

	assert.NoError(t, err)
}

func TestUpdateMetrics_Counter_Success(t *testing.T) {
	ctx := context.Background()
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Set("requests", models.Counter, 11, ctx).Return(nil).Once()
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics("requests", models.Counter, "11", ctx)

	assert.NoError(t, err)
}

func TestUpdateMetrics_Counter_InvalidString(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics("requests", models.Counter, "NaN", context.Background())

	assert.ErrorIs(t, err, errParseInt)
}

func TestUpdateMetrics_UnknownType(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics("test", "what????????", "123", context.Background())

	assert.ErrorIs(t, err, errUnknownType)
}
