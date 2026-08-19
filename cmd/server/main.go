package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kheguy/collector/internal/config"
	"github.com/kheguy/collector/internal/handler"
	"github.com/kheguy/collector/internal/middlewares"
	"github.com/kheguy/collector/internal/repository"
	"github.com/kheguy/collector/internal/service"
	"github.com/kheguy/collector/internal/templates"
)

func main() {
	cfg, cfgErr := config.Load()

	if cfgErr != nil {
		log.Fatal("Config loading error: ", cfgErr)
	}

	renderer, tempErr := templates.MakeNewTemplateRenderer()
	if tempErr != nil {
		log.Fatal("Template loading error: ", tempErr)
	}

	r := chi.NewRouter()
	r.Use(middlewares.WithLogging)

	storage := repository.MakeNewMemoryStorage()

	metricsService := service.MakeNewMetricsService(storage)

	metricsHandler := handler.MakeNewMetricsHandler(metricsService, renderer)

	r.Post(`/update/{type}/{name}/{value}`, metricsHandler.UpdateHandler)
	r.Get(`/value/{type}/{name}`, metricsHandler.ValueHandler)
	r.Get(`/`, metricsHandler.HTMLListHandler)

	log.Printf("Server started on %s\n", cfg.Address)
	httpErr := http.ListenAndServe(cfg.Address, r)

	if httpErr != nil {
		log.Fatal(httpErr)
	}
}
