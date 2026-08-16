package service

import (
	"testing"

	"github.com/kheguy/collector/internal/mocks"
	models "github.com/kheguy/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestMakeNewMetricsService(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	svc := MakeNewMetricsService(storageMock)
	assert.NotNil(t, svc)
	assert.Equal(t, storageMock, svc.storage)
}

func TestGetMetric_Int(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get("some_metrics").Return(111111).Once()
	svc := MakeNewMetricsService(storageMock)

	result := svc.GetMetric("some_metrics")
	assert.Equal(t, "111111", result)
}

func TestGetMetric_Float64(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get("some_metrics").Return(11.111).Once()
	svc := MakeNewMetricsService(storageMock)

	result := svc.GetMetric("some_metrics")
	assert.Equal(t, "11.111", result)
}

func TestGetMetric_NotFound(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Get("what??????????").Return(nil).Once()
	svc := MakeNewMetricsService(storageMock)

	result := svc.GetMetric("what??????????")
	assert.Equal(t, "", result)
}

func TestGetAllMetrics(t *testing.T) {
	expected := map[string]interface{}{
		"a": 1.1,
		"b": 2,
	}
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().GetAll().Return(expected).Once()
	svc := MakeNewMetricsService(storageMock)

	result := svc.GetAllMetrics()
	assert.Equal(t, expected, result)
}

func TestUpdateMetrics_Gauge_Success(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Set("oneone", models.Gauge, 11.11111).Return(nil).Once()
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics("oneone", models.Gauge, "11.11111")

	assert.NoError(t, err)
}

func TestUpdateMetrics_Counter_Success(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	storageMock.EXPECT().Set("requests", models.Counter, 11).Return(nil).Once()
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics("requests", models.Counter, "11")

	assert.NoError(t, err)
}

func TestUpdateMetrics_Counter_InvalidString(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics("requests", models.Counter, "NaN")

	assert.ErrorIs(t, err, parseIntError)
}

func TestUpdateMetrics_UnknownType(t *testing.T) {
	storageMock := mocks.NewMockStorage(t)
	svc := MakeNewMetricsService(storageMock)

	err := svc.UpdateMetrics("test", "what????????", "123")

	assert.ErrorIs(t, err, uknownTypeError)
}
