package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kheguy/collector/internal/config"
	"github.com/kheguy/collector/internal/handler"
	"github.com/kheguy/collector/internal/middlewares"
	"github.com/kheguy/collector/internal/repository"
	"github.com/kheguy/collector/internal/service"
	"github.com/kheguy/collector/internal/templates"
)

func main() {
	cfg, cfgErr := config.LoadServer()

	if cfgErr != nil {
		log.Fatal("Config loading error: ", cfgErr)
	}

	renderer, tempErr := templates.MakeNewTemplateRenderer()
	if tempErr != nil {
		log.Fatal("Template loading error: ", tempErr)
	}

	r := chi.NewRouter()
	r.Use(middlewares.WithLogging)
	r.Use(middlewares.WithGzip)

	storage := repository.MakeNewMemoryStorage()

	if cfg.Restore {
		if err := storage.Restore(cfg.FileStoragePath); err != nil {
			log.Fatal("Metrics restore error: ", err)
		}
	}

	if cfg.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := storage.Save(cfg.FileStoragePath); err != nil {
					log.Printf("Metrics save error: %v", err)
				}
			}
		}()
	}

	// Я не уверен что это окей
	var saveMetrics func() error
	if cfg.StoreInterval == 0 {
		saveMetrics = func() error {
			return storage.Save(cfg.FileStoragePath)
		}
	}
	metricsService := service.MakeNewMetricsService(storage, saveMetrics)

	metricsHandler := handler.MakeNewMetricsHandler(metricsService, renderer)

	r.Post(`/update/{type}/{name}/{value}`, metricsHandler.UpdateHandler)
	r.Post(`/update`, metricsHandler.JSONUpdateHandler)
	r.Post(`/update/`, metricsHandler.JSONUpdateHandler)
	r.Get(`/value/{type}/{name}`, metricsHandler.ValueHandler)
	r.Post(`/value`, metricsHandler.JSONValueHandler)
	r.Post(`/value/`, metricsHandler.JSONValueHandler)
	r.Get(`/`, metricsHandler.HTMLListHandler)

	log.Printf("Server started on %s\n", cfg.Address)
	httpErr := http.ListenAndServe(cfg.Address, r)

	if httpErr != nil {
		log.Fatal(httpErr)
	}
}
