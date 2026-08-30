package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
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

	saveCtx, cancelSave := context.WithCancel(context.Background())
	var saveWG sync.WaitGroup

	if cfg.StoreInterval > 0 {
		saveWG.Add(1)
		go func() {
			defer saveWG.Done()
			ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := storage.Save(cfg.FileStoragePath); err != nil {
						log.Printf("Metrics save error: %v", err)
					}
				case <-saveCtx.Done():
					if err := storage.Save(cfg.FileStoragePath); err != nil {
						log.Printf("Final metrics save error: %v", err)
					}
					return
				}
			}
		}()
	}

	metricsService := service.MakeNewMetricsService(storage, cfg.FileStoragePath, cfg.StoreInterval)

	metricsHandler := handler.MakeNewMetricsHandler(metricsService, renderer)

	r.Post(`/update/{type}/{name}/{value}`, metricsHandler.UpdateHandler)
	r.Post(`/update`, metricsHandler.JSONUpdateHandler)
	r.Post(`/update/`, metricsHandler.JSONUpdateHandler)
	r.Get(`/value/{type}/{name}`, metricsHandler.ValueHandler)
	r.Post(`/value`, metricsHandler.JSONValueHandler)
	r.Post(`/value/`, metricsHandler.JSONValueHandler)
	r.Get(`/`, metricsHandler.HTMLListHandler)

	server := &http.Server{
		Addr:    cfg.Address,
		Handler: r,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server started on %s\n", cfg.Address)
		serverErrors <- server.ListenAndServe()
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var serverErr error
	select {
	case serverErr = <-serverErrors:
	case <-signalCtx.Done():
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		cancelShutdown()
		serverErr = <-serverErrors
	}

	cancelSave()
	saveWG.Wait()

	if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
		log.Fatal(serverErr)
	}
}
