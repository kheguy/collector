package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kheguy/collector/internal/config"
	"github.com/kheguy/collector/internal/handler"
	"github.com/kheguy/collector/internal/repository"
	"github.com/kheguy/collector/internal/service"
)

func main() {
	r := chi.NewRouter()

	config := config.Load()

	storage := repository.MakeNewMemoryStorage()

	metricsService := service.MakeNewMetricsService(storage)

	metricsHandler := handler.MakeNewMetricsHandler(metricsService)

	r.Post(`/update/{type}/{name}/{value}`, metricsHandler.UpdateHandler)
	r.Get(`/value/{type}/{name}`, metricsHandler.ValueHandler)
	r.Get(`/`, metricsHandler.HTMLListHandler)

	fmt.Printf("Server started on port %d\n", config.Port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), r)

	if err != nil {
		panic(err)
	}
}
