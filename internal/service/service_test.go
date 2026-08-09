package service

import (
	"testing"

	models "github.com/kheguy/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

type MockStorage struct {
	GetFunc    func(name string) interface{}
	GetAllFunc func() map[string]interface{}
	SetFunc    func(name string, mType string, value interface{}) interface{}
}

func (m *MockStorage) Get(name string) interface{} {
	if m.GetFunc != nil {
		return m.GetFunc(name)
	}
	return nil
}

func (m *MockStorage) GetAll() map[string]interface{} {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return make(map[string]interface{})
}

func (m *MockStorage) Set(name string, mType string, value interface{}) interface{} {
	if m.SetFunc != nil {
		return m.SetFunc(name, mType, value)
	}
	return nil
}

func TestMakeNewMetricsService(t *testing.T) {
	mock := &MockStorage{}
	svc := MakeNewMetricsService(mock)
	assert.NotNil(t, svc)
	assert.Equal(t, mock, svc.storage)
}

func TestGetMetric_Int(t *testing.T) {
	mock := &MockStorage{
		GetFunc: func(name string) interface{} {
			return 111111
		},
	}
	svc := MakeNewMetricsService(mock)

	result := svc.GetMetric("some_metrics")
	assert.Equal(t, "111111", result)
}

func TestGetMetric_Float64(t *testing.T) {
	mock := &MockStorage{
		GetFunc: func(name string) interface{} {
			return 11.111
		},
	}
	svc := MakeNewMetricsService(mock)

	result := svc.GetMetric("some_metrics")
	assert.Equal(t, "11.111", result)
}

func TestGetMetric_NotFound(t *testing.T) {
	mock := &MockStorage{
		GetFunc: func(name string) interface{} {
			return nil
		},
	}
	svc := MakeNewMetricsService(mock)

	result := svc.GetMetric("what??????????")
	assert.Equal(t, "", result)
}

func TestGetAllMetrics(t *testing.T) {
	expected := map[string]interface{}{
		"a": 1.1,
		"b": 2,
	}
	mock := &MockStorage{
		GetAllFunc: func() map[string]interface{} {
			return expected
		},
	}
	svc := MakeNewMetricsService(mock)

	result := svc.GetAllMetrics()
	assert.Equal(t, expected, result)
}

func TestUpdateMetrics_Gauge_Success(t *testing.T) {
	var capturedName, capturedType string
	var capturedValue interface{}

	mock := &MockStorage{
		SetFunc: func(name string, mType string, value interface{}) interface{} {
			capturedName = name
			capturedType = mType
			capturedValue = value
			return nil
		},
	}
	svc := MakeNewMetricsService(mock)

	err := svc.UpdateMetrics("oneone", models.Gauge, "11.11111")

	assert.NoError(t, err)
	assert.Equal(t, "oneone", capturedName)
	assert.Equal(t, models.Gauge, capturedType)
	assert.Equal(t, 11.11111, capturedValue)
}

func TestUpdateMetrics_Counter_Success(t *testing.T) {
	var capturedName, capturedType string
	var capturedValue interface{}

	mock := &MockStorage{
		SetFunc: func(name string, mType string, value interface{}) interface{} {
			capturedName = name
			capturedType = mType
			capturedValue = value
			return nil
		},
	}
	svc := MakeNewMetricsService(mock)

	err := svc.UpdateMetrics("requests", models.Counter, "11")

	assert.NoError(t, err)
	assert.Equal(t, "requests", capturedName)
	assert.Equal(t, models.Counter, capturedType)
	assert.Equal(t, 11, capturedValue)
}

func TestUpdateMetrics_Counter_InvalidString(t *testing.T) {
	mock := &MockStorage{}
	svc := MakeNewMetricsService(mock)

	err := svc.UpdateMetrics("requests", models.Counter, "NaN")

	assert.ErrorIs(t, err, parseIntError)
}

func TestUpdateMetrics_UnknownType(t *testing.T) {
	mock := &MockStorage{}
	svc := MakeNewMetricsService(mock)

	err := svc.UpdateMetrics("test", "what????????", "123")

	assert.ErrorIs(t, err, uknownTypeError)
}
