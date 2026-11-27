package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/kheguy/collector/internal/audit"
	models "github.com/kheguy/collector/internal/model"
	metricservice "github.com/kheguy/collector/internal/service"
)

type Service interface {
	GetMetric(ctx context.Context, name string) (string, error)
	GetRawMetric(ctx context.Context, name string) (interface{}, error)
	GetAllMetrics(ctx context.Context) (map[string]interface{}, error)
	UpdateMetrics(ctx context.Context, name string, typeOfValue string, value interface{}) error
	UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error
}

type Renderer interface {
	Render(w http.ResponseWriter, name string, data interface{})
}

type MetricsHandler struct {
	service  Service
	renderer Renderer
	audit    *audit.Publisher
}

func MakeNewMetricsHandler(s Service, r Renderer, observers ...audit.Observer) *MetricsHandler {
	h := &MetricsHandler{
		service:  s,
		renderer: r,
	}
	if len(observers) > 0 {
		h.audit = audit.NewPublisher(observers...)
	}
	return h
}

func (h *MetricsHandler) ValueHandler(res http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")

	if name == "" {
		http.Error(res, "Name is required", http.StatusBadRequest)
		return
	}

	metricValue, err := h.service.GetMetric(req.Context(), name)

	if err != nil {
		http.Error(res, http.StatusText(500), http.StatusInternalServerError)
		return
	}

	if metricValue == "" {
		http.Error(res, http.StatusText(404), http.StatusNotFound)
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(metricValue))
}

func (h *MetricsHandler) UpdateHandler(res http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")
	typeOfValue := req.PathValue("type")
	value := req.PathValue("value")

	if name == "" {
		http.Error(res, "Name is required", http.StatusNotFound)
		return
	}

	err := h.service.UpdateMetrics(req.Context(), name, typeOfValue, value)

	if err != nil {
		writeServiceError(res, err)
		return
	}

	h.notifyAudit(req, []string{name})
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(`OK`))
}

func (h *MetricsHandler) JSONUpdateHandler(res http.ResponseWriter, req *http.Request) {
	var metric models.Metrics
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var value string
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		value = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
	case models.Counter:
		if metric.Delta == nil {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		value = strconv.FormatInt(*metric.Delta, 10)
	default:
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if metric.ID == "" {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err := h.service.UpdateMetrics(req.Context(), metric.ID, metric.MType, value); err != nil {
		writeServiceError(res, err)
		return
	}

	stored, err := h.service.GetRawMetric(req.Context(), metric.ID)

	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if stored == nil {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	metric = makeMetric(metric.ID, metric.MType, stored)
	h.notifyAudit(req, []string{metric.ID})
	writeJSON(res, http.StatusOK, metric)
}

func (h *MetricsHandler) JSONBatchUpdateHandler(res http.ResponseWriter, req *http.Request) {
	var metrics []models.Metrics
	if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateMetricsBatch(req.Context(), metrics); err != nil {
		writeServiceError(res, err)
		return
	}

	if h.audit != nil {
		names := make([]string, len(metrics))
		for i, metric := range metrics {
			names[i] = metric.ID
		}
		h.notifyAudit(req, names)
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) notifyAudit(req *http.Request, names []string) {
	if h.audit == nil {
		return
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		ip = req.RemoteAddr
	}
	event := audit.Event{
		Timestamp: time.Now().Unix(),
		Metrics:   names,
		IPAddress: ip,
	}
	if err := h.audit.Notify(req.Context(), event); err != nil {
		log.Printf("Audit error: %v", err)
	}
}

func (h *MetricsHandler) JSONValueHandler(res http.ResponseWriter, req *http.Request) {
	var metric models.Metrics
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if metric.ID == "" || (metric.MType != models.Gauge && metric.MType != models.Counter) {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	stored, err := h.service.GetRawMetric(req.Context(), metric.ID)

	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if stored == nil {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	result := makeMetric(metric.ID, metric.MType, stored)
	if result.Value == nil && result.Delta == nil {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	writeJSON(res, http.StatusOK, result)
}

func writeServiceError(res http.ResponseWriter, err error) {
	if errors.Is(err, metricservice.ErrInvalidMetric) {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func makeMetric(id string, mType string, value interface{}) models.Metrics {
	metric := models.Metrics{ID: id, MType: mType}
	switch v := value.(type) {
	case float64:
		metric.Value = &v
	case int:
		delta := int64(v)
		metric.Delta = &delta
	}
	return metric
}

func writeJSON(res http.ResponseWriter, status int, value interface{}) {
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	_ = json.NewEncoder(res).Encode(value)
}

func (h *MetricsHandler) HTMLListHandler(res http.ResponseWriter, req *http.Request) {
	type PageData struct {
		Title   string
		Metrics []models.MetricItem
	}

	metrics := make([]models.MetricItem, 0)

	m, err := h.service.GetAllMetrics(req.Context())

	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for name, value := range m {
		var mValue string
		switch v := value.(type) {
		case int:
			mValue = strconv.Itoa(v)
		case float64:
			mValue = strconv.FormatFloat(v, 'f', -1, 64)
		}

		metrics = append(metrics, models.MetricItem{
			Name:  name,
			Value: mValue,
		})
	}

	data := PageData{
		Title:   "List",
		Metrics: metrics,
	}

	h.renderer.Render(res, "list", data)
}
