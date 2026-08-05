package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/kheguy/collector/internal/service"
)

type MetricsHandler struct {
	service *service.MetricsService
}

func MakeNewMetricsHandler(s *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		service: s,
	}
}

func (h *MetricsHandler) ValueHandler(res http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")
	// Как будто оно тут и не нужно?
	// typeOfValue := req.PathValue("type")

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
	// fmt.Printf("Got request %s\n", req.URL)

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

	htmlString := "<table><thead><th>Name</th><th>Value</th></thead><tbody>"
	for name, value := range h.service.GetAllMetrics() {
		var mValue string

		switch v := value.(type) {
		case int:
			mValue = strconv.Itoa(v)
		case float64:
			mValue = strconv.FormatFloat(v, 'f', -1, 64)
		}

		htmlString += fmt.Sprintf(`<tr><td>%s</td><td>%s</td></tr>`, name, mValue)
	}

	htmlString += `</tbody></table>`
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(htmlString))
}

func (h *MetricsHandler) NotFoundHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusNotFound)
	res.Write([]byte(`Not found`))
}
