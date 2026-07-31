package main

import (
	"fmt"
	"net/http"

	"github.com/kheguy/collector/internal/config"
	"github.com/kheguy/collector/internal/handler"
	"github.com/kheguy/collector/internal/repository"
	"github.com/kheguy/collector/internal/service"
)

func main() {
	config := config.Load()

	storage := repository.MakeNewMemoryStorage()

	metricsService := service.MakeNewMetricsService(storage)

	metricsHandler := handler.MakeNewMetricsHandler(metricsService)

	mux := http.NewServeMux()
	mux.HandleFunc(`/update/{type}/{name}/{value}`, metricsHandler.UpdateHandler)
	mux.HandleFunc(`/`, metricsHandler.NotFoundHandler)

	fmt.Printf("Server started on port %d\n", config.Port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), mux)

	if err != nil {
		panic(err)
	}
}
