package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kheguy/collector/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValueHandler_Success(t *testing.T) {
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().GetMetric("test").Return("Brrrrrrbzzzzzzz").Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))

	req := httptest.NewRequest("GET", "/value/gauge/test", nil)
	req.SetPathValue("name", "test")
	w := httptest.NewRecorder()

	handler.ValueHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Brrrrrrbzzzzzzz", w.Body.String())
}

func TestValueHandler_NotFound(t *testing.T) {
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().GetMetric("TYKTOBRAT").Return("").Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))

	req := httptest.NewRequest("GET", "/value/gauge/TYKTOBRAT", nil)
	req.SetPathValue("name", "TYKTOBRAT")
	w := httptest.NewRecorder()

	handler.ValueHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateHandler_Success(t *testing.T) {
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().UpdateMetrics("test", "gauge", "404").Return(nil).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))

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
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().UpdateMetrics("test", "gauge", "404").Return(assert.AnError).Once()
	handler := MakeNewMetricsHandler(serviceMock, mocks.NewMockRenderer(t))

	req := httptest.NewRequest("POST", "/update/gauge/test/404", nil)
	req.SetPathValue("type", "gauge")
	req.SetPathValue("name", "test")
	req.SetPathValue("value", "404")
	w := httptest.NewRecorder()

	handler.UpdateHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHTMLListHandler(t *testing.T) {
	serviceMock := mocks.NewMockService(t)
	serviceMock.EXPECT().GetAllMetrics().Return(map[string]interface{}{
		"temp":  1.2,
		"count": 3,
	}).Once()

	var capturedData interface{}
	rendererMock := mocks.NewMockRenderer(t)
	rendererMock.EXPECT().Render(mock.Anything, "list", mock.Anything).Run(
		func(w http.ResponseWriter, name string, data interface{}) {
			capturedData = data
			w.WriteHeader(http.StatusOK)
		},
	).Once()

	handler := MakeNewMetricsHandler(serviceMock, rendererMock)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.HTMLListHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedData)
}

func TestNotFoundHandler(t *testing.T) {
	handler := MakeNewMetricsHandler(mocks.NewMockService(t), mocks.NewMockRenderer(t))

	req := httptest.NewRequest("GET", "/any", nil)
	w := httptest.NewRecorder()

	handler.NotFoundHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "Not found", w.Body.String())
}
