package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kheguy/collector/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestPingHandler_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	poolMock := mocks.NewMockPool(t)
	poolMock.EXPECT().
		Ping(req.Context()).
		Return(nil).
		Once()

	healthHandler := MakeNewHealthHandler(poolMock)
	healthHandler.PingHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPingHandler_Error(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	poolMock := mocks.NewMockPool(t)
	poolMock.EXPECT().
		Ping(req.Context()).
		Return(assert.AnError).
		Once()

	healthHandler := MakeNewHealthHandler(poolMock)
	healthHandler.PingHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
