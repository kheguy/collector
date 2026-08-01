package handler

import (
	"net/http"

	"github.com/kheguy/collector/internal/service"
)

type MetricsHandler struct {
	Service *service.MetricsService
}

func MakeNewMetricsHandler(s *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		Service: s,
	}
}

func (h *MetricsHandler) UpdateHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// fmt.Printf("Got request %s\n", req.URL)

	name := req.PathValue("name")
	typeOfValue := req.PathValue("type")
	value := req.PathValue("value")

	if name == "" {
		http.Error(res, "Name is required", http.StatusNotFound)
		return
	}

	err := h.Service.UpdateMetrics(name, typeOfValue, value)

	if err != nil {
		http.Error(res, "Bad request", http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(`OK`))
}

func (h *MetricsHandler) NotFoundHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusNotFound)
	res.Write([]byte(`Not found`))
}
