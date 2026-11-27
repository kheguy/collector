package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kheguy/collector/internal/audit"
	models "github.com/kheguy/collector/internal/model"
	"github.com/kheguy/collector/internal/repository"
	metricservice "github.com/kheguy/collector/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingObserver struct {
	events []audit.Event
	err    error
}

func (o *recordingObserver) Notify(_ context.Context, event audit.Event) error {
	o.events = append(o.events, event)
	return o.err
}

func auditTestRouter(h *MetricsHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateHandler)
	r.Post("/update/", h.JSONUpdateHandler)
	r.Post("/updates/", h.JSONBatchUpdateHandler)
	return r
}

func TestWriteHandlersAuditEvent(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		remoteAddr string
		wantIP     string
		wantNames  []string
	}{
		{
			name: "URL update", path: "/update/gauge/Alloc/12.5",
			remoteAddr: "192.168.0.42:54321", wantIP: "192.168.0.42",
			wantNames: []string{"Alloc"},
		},
		{
			name: "JSON update", path: "/update/",
			body:       `{"id":"Frees","type":"counter","delta":3}`,
			remoteAddr: "[2001:db8::42]:54321", wantIP: "2001:db8::42",
			wantNames: []string{"Frees"},
		},
		{
			name: "batch update", path: "/updates/",
			body:       `[{"id":"Frees","type":"counter","delta":3},{"id":"Alloc","type":"gauge","value":12.5}]`,
			remoteAddr: "192.168.0.42", wantIP: "192.168.0.42",
			wantNames: []string{"Frees", "Alloc"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMemoryStorage()
			observer := &recordingObserver{}
			h := MakeNewMetricsHandler(metricservice.MakeNewMetricsService(storage), nil, observer)
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			before := time.Now().Unix()
			auditTestRouter(h).ServeHTTP(w, req)
			after := time.Now().Unix()

			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Len(t, observer.events, 1)
			event := observer.events[0]
			assert.Equal(t, tt.wantNames, event.Metrics)
			assert.Equal(t, tt.wantIP, event.IPAddress)
			assert.GreaterOrEqual(t, event.Timestamp, before)
			assert.LessOrEqual(t, event.Timestamp, after)
		})
	}
}

func TestAuditFailureDoesNotFailUpdate(t *testing.T) {
	storage := repository.NewMemoryStorage()
	require.NoError(t, storage.Set(context.Background(), "Frees", models.Counter, 4))
	failed := &recordingObserver{err: assert.AnError}
	h := MakeNewMetricsHandler(metricservice.MakeNewMetricsService(storage), nil, failed)
	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(`{"id":"Frees","type":"counter","delta":3}`))
	w := httptest.NewRecorder()

	auditTestRouter(h).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	stored, err := storage.Get(req.Context(), "Frees")
	require.NoError(t, err)
	assert.Equal(t, 7, stored)
	require.Len(t, failed.events, 1)
}
