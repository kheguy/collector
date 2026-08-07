package handler

import (
	"net/http"
	"strconv"

	models "github.com/kheguy/collector/internal/model"
)

type Service interface {
	GetMetric(name string) string
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
		http.Error(res, "Not foud", http.StatusNotFound)
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
		http.Error(res, "Bad request", http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(`OK`))
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
