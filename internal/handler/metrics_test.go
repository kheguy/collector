package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kheguy/collector/internal/mocks"
	models "github.com/kheguy/collector/internal/model"
	metricservice "github.com/kheguy/collector/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValueHandler_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/value/gauge/test", nil)
	req.SetPathValue("name", "test")

	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().GetMetric(req.Context(), "test").Return("Brrrrrrbzzzzzzz", nil).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.ValueHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Brrrrrrbzzzzzzz", w.Body.String())
}

func TestValueHandler_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/value/gauge/TYKTOBRAT", nil)
	req.SetPathValue("name", "TYKTOBRAT")

	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().GetMetric(req.Context(), "TYKTOBRAT").Return("", nil).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.ValueHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestValueHandler_ServiceError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/value/gauge/test", nil)
	req.SetPathValue("name", "test")

	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().GetMetric(req.Context(), "test").Return("", assert.AnError).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.ValueHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestValueHandler_EmptyName(t *testing.T) {
	handler := MakeNewMetricsHandler(mocks.NewMockService(t), mocks.NewMockRenderer(t))
	req := httptest.NewRequest(http.MethodGet, "/value/gauge/", nil)
	w := httptest.NewRecorder()

	handler.ValueHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateHandler_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/test/404", nil)
	req.SetPathValue("type", "gauge")
	req.SetPathValue("name", "test")
	req.SetPathValue("value", "404")

	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().UpdateMetrics(req.Context(), "test", "gauge", "404").Return(nil).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.UpdateHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestUpdateHandler_ServiceError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/test/404", nil)
	req.SetPathValue("type", "gauge")
	req.SetPathValue("name", "test")
	req.SetPathValue("value", "404")

	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().UpdateMetrics(req.Context(), "test", "gauge", "404").Return(assert.AnError).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.UpdateHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateHandler_InvalidMetric(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/test/invalid", nil)
	req.SetPathValue("type", "gauge")
	req.SetPathValue("name", "test")
	req.SetPathValue("value", "invalid")

	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().
		UpdateMetrics(req.Context(), "test", "gauge", "invalid").
		Return(metricservice.ErrInvalidMetric).
		Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.UpdateHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestJSONUpdateHandler_GaugeSuccess(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/update/",
		strings.NewReader(`{"id":"temperature","type":"gauge","value":12.5}`),
	)
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().
		UpdateMetrics(req.Context(), "temperature", models.Gauge, "12.5").
		Return(nil).
		Once()
	serviceMock.EXPECT().
		GetRawMetric(req.Context(), "temperature").
		Return(12.5, nil).
		Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.JSONUpdateHandler(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var got models.Metrics
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, "temperature", got.ID)
	assert.Equal(t, models.Gauge, got.MType)
	require.NotNil(t, got.Value)
	assert.Equal(t, 12.5, *got.Value)
}

func TestJSONUpdateHandler_GetStoredMetricError(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/update/",
		strings.NewReader(`{"id":"temperature","type":"gauge","value":12.5}`),
	)
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().
		UpdateMetrics(req.Context(), "temperature", models.Gauge, "12.5").
		Return(nil).
		Once()
	serviceMock.EXPECT().
		GetRawMetric(req.Context(), "temperature").
		Return(nil, assert.AnError).
		Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.JSONUpdateHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestJSONBatchUpdateHandler_Success(t *testing.T) {
	gauge := 12.5
	delta := int64(3)
	metrics := []models.Metrics{
		{ID: "temperature", MType: models.Gauge, Value: &gauge},
		{ID: "requests", MType: models.Counter, Delta: &delta},
	}
	req := httptest.NewRequest(
		http.MethodPost,
		"/updates/",
		strings.NewReader(`[{"id":"temperature","type":"gauge","value":12.5},{"id":"requests","type":"counter","delta":3}]`),
	)
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().UpdateMetricsBatch(req.Context(), metrics).Return(nil).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.JSONBatchUpdateHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestJSONBatchUpdateHandler_InvalidJSON(t *testing.T) {
	serviceMock := mocks.NewMockService(t)
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(`[{"id":`))
	w := httptest.NewRecorder()

	handler.JSONBatchUpdateHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestJSONBatchUpdateHandler_ServiceError(t *testing.T) {
	gauge := 12.5
	metrics := []models.Metrics{{ID: "temperature", MType: models.Gauge, Value: &gauge}}
	req := httptest.NewRequest(
		http.MethodPost,
		"/updates/",
		strings.NewReader(`[{"id":"temperature","type":"gauge","value":12.5}]`),
	)
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().UpdateMetricsBatch(req.Context(), metrics).Return(assert.AnError).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.JSONBatchUpdateHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestJSONValueHandler_Success(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/value/",
		strings.NewReader(`{"id":"requests","type":"counter"}`),
	)
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().GetRawMetric(req.Context(), "requests").Return(7, nil).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.JSONValueHandler(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got models.Metrics
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, "requests", got.ID)
	assert.Equal(t, models.Counter, got.MType)
	require.NotNil(t, got.Delta)
	assert.Equal(t, int64(7), *got.Delta)
}

func TestJSONValueHandler_NotFound(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/value/",
		strings.NewReader(`{"id":"unknown","type":"counter"}`),
	)
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().GetRawMetric(req.Context(), "unknown").Return(nil, nil).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))
	w := httptest.NewRecorder()

	handler.JSONValueHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHTMLListHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().GetAllMetrics(req.Context()).Return(map[string]interface{}{
		"temp":  1.2,
		"count": 3,
	}, nil).Once()

	var capturedData interface{}
	rendererMock := mocks.NewMockRenderer(t)
	rendererMock.EXPECT().Render(mock.Anything, "list", mock.Anything).Run(
		func(w http.ResponseWriter, name string, data interface{}) {
			capturedData = data
			w.WriteHeader(http.StatusOK)
		},
	).Once()

	handler := MakeNewMetricsHandler(serviceMock, rendererMock)
	w := httptest.NewRecorder()

	handler.HTMLListHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedData)
}
