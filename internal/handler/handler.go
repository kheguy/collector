package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	models "github.com/kheguy/collector/internal/model"
)

type Service interface {
	GetMetric(name string) string
	GetRawMetric(name string) interface{}
	GetAllMetrics() map[string]interface{}
	UpdateMetrics(name string, typeOfValue string, value interface{}) error
}

type Renderer interface {
	Render(w http.ResponseWriter, name string, data interface{})
}

type MetricsHandler struct {
	service  Service
	renderer Renderer
}

func MakeNewMetricsHandler(s Service, r Renderer) *MetricsHandler {
	return &MetricsHandler{
		service:  s,
		renderer: r,
	}
}

func (h *MetricsHandler) ValueHandler(res http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")

	if name == "" {
		http.Error(res, "Name is required", http.StatusBadRequest)
		return
	}

	metricValue := h.service.GetMetric(name)

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

	err := h.service.UpdateMetrics(name, typeOfValue, value)

	if err != nil {
		http.Error(res, http.StatusText(400), http.StatusBadRequest)
		return
	}

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
	if err := h.service.UpdateMetrics(metric.ID, metric.MType, value); err != nil {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	stored := h.service.GetRawMetric(metric.ID)
	if stored == nil {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	metric = makeMetric(metric.ID, metric.MType, stored)
	writeJSON(res, http.StatusOK, metric)
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

	stored := h.service.GetRawMetric(metric.ID)
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

	for name, value := range h.service.GetAllMetrics() {
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

func (h *MetricsHandler) NotFoundHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusNotFound)
	res.Write([]byte(`Not found`))
}
