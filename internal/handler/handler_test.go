package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockService struct {
	GetMetricFunc     func(name string) string
	GetAllMetricsFunc func() map[string]interface{}
	UpdateMetricsFunc func(name string, typeOfValue string, value interface{}) error
}

func (m *MockService) GetMetric(name string) string {
	if m.GetMetricFunc != nil {
		return m.GetMetricFunc(name)
	}
	return ""
}

func (m *MockService) GetAllMetrics() map[string]interface{} {
	if m.GetAllMetricsFunc != nil {
		return m.GetAllMetricsFunc()
	}
	return nil
}

func (m *MockService) UpdateMetrics(name string, typeOfValue string, value interface{}) error {
	if m.UpdateMetricsFunc != nil {
		return m.UpdateMetricsFunc(name, typeOfValue, value)
	}
	return nil
}

type MockRenderer struct {
	RenderFunc func(w http.ResponseWriter, name string, data interface{})
}

func (m *MockRenderer) Render(w http.ResponseWriter, name string, data interface{}) {
	if m.RenderFunc != nil {
		m.RenderFunc(w, name, data)
	}
}

func TestValueHandler_Success(t *testing.T) {
	mock := &MockService{
		GetMetricFunc: func(name string) string {
			return "Brrrrrrbzzzzzzz"
		},
	}
	handler := MakeNewMetricsHandler(mock, &MockRenderer{})

	req := httptest.NewRequest("GET", "/value/gauge/test", nil)
	req.SetPathValue("name", "test")
	w := httptest.NewRecorder()

	handler.ValueHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Brrrrrrbzzzzzzz", w.Body.String())
}

func TestValueHandler_NotFound(t *testing.T) {
	mock := &MockService{
		GetMetricFunc: func(name string) string {
			return ""
		},
	}
	handler := MakeNewMetricsHandler(mock, &MockRenderer{})

	req := httptest.NewRequest("GET", "/value/gauge/TYKTOBRAT", nil)
	req.SetPathValue("name", "TYKTOBRAT")
	w := httptest.NewRecorder()

	handler.ValueHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateHandler_Success(t *testing.T) {
	mock := &MockService{
		UpdateMetricsFunc: func(name string, typeOfValue string, value interface{}) error {
			return nil
		},
	}
	handler := MakeNewMetricsHandler(mock, &MockRenderer{})

	req := httptest.NewRequest("POST", "/update/gauge/test/404", nil)
	req.SetPathValue("type", "gauge")
	req.SetPathValue("name", "test")
	req.SetPathValue("value", "404")
	w := httptest.NewRecorder()

	handler.UpdateHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestUpdateHandler_ServiceError(t *testing.T) {
	mock := &MockService{
		UpdateMetricsFunc: func(name string, typeOfValue string, value interface{}) error {
			return assert.AnError
		},
	}
	handler := MakeNewMetricsHandler(mock, &MockRenderer{})

	req := httptest.NewRequest("POST", "/update/gauge/test/404", nil)
	req.SetPathValue("type", "gauge")
	req.SetPathValue("name", "test")
	req.SetPathValue("value", "404")
	w := httptest.NewRecorder()

	handler.UpdateHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHTMLListHandler(t *testing.T) {
	mock := &MockService{
		GetAllMetricsFunc: func() map[string]interface{} {
			return map[string]interface{}{
				"temp":  1.2,
				"count": 3,
			}
		},
	}

	var capturedData interface{}
	renderer := &MockRenderer{
		RenderFunc: func(w http.ResponseWriter, name string, data interface{}) {
			capturedData = data
			w.WriteHeader(http.StatusOK)
		},
	}

	handler := MakeNewMetricsHandler(mock, renderer)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.HTMLListHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedData)
}

func TestNotFoundHandler(t *testing.T) {
	handler := MakeNewMetricsHandler(&MockService{}, &MockRenderer{})

	req := httptest.NewRequest("GET", "/any", nil)
	w := httptest.NewRecorder()

	handler.NotFoundHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "Not found", w.Body.String())
}
