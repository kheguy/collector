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
	"github.com/kheguy/collector/internal/database"
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

	var cancelSave context.CancelFunc = func() {}
	var saveWG sync.WaitGroup
	var storage service.Storage
	var pinger handler.Pinger
	if cfg.DBAddress != "" {
		dbClient, err := database.NewClient(context.Background(), cfg.DBAddress)
		if err != nil {
			log.Fatal("Database initialization error: ", err)
		}
		defer dbClient.Close()

		storage = dbClient.Storage()
		pinger = dbClient
	} else {
		memoryStorage := repository.NewMemoryStorage()
		storage = memoryStorage
		pinger = memoryStorage

		fileProccessor := repository.NewFileProcessor(cfg.FileStoragePath)
		if cfg.Restore || cfg.StoreInterval > 0 {
			pinger = fileProccessor
		}

		if cfg.Restore {
			restoredData, err := fileProccessor.Restore()
			if err != nil {
				log.Fatal("Metrics restore error: ", err)
			}
			storage.SetAll(context.Background(), restoredData)
		}

		saveCtx, cancel := context.WithCancel(context.Background())
		cancelSave = cancel

		if cfg.StoreInterval > 0 {
			saveWG.Add(1)
			go func() {
				defer saveWG.Done()
				ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						data, err := storage.GetAll(context.Background())
						if err != nil {
							log.Printf("Metrics save error: %v", err)
						}
						if err := fileProccessor.Save(data); err != nil {
							log.Printf("Metrics save error: %v", err)
						}
					case <-saveCtx.Done():
						data, err := storage.GetAll(context.Background())
						if err != nil {
							log.Printf("Final metrics save error: %v", err)
						}
						if err := fileProccessor.Save(data); err != nil {
							log.Printf("Final metrics save error: %v", err)
						}
						return
					}
				}
			}()
		}
	}

	healthHandler := handler.MakeNewHealthHandler(pinger)
	r.Get(`/ping`, healthHandler.PingHandler)

	metricsService := service.MakeNewMetricsService(storage)

	metricsHandler := handler.MakeNewMetricsHandler(metricsService, renderer)

	r.Post(`/update/{type}/{name}/{value}`, metricsHandler.UpdateHandler)
	r.Post(`/update`, metricsHandler.JSONUpdateHandler)
	r.Post(`/update/`, metricsHandler.JSONUpdateHandler)
	r.Post(`/updates`, metricsHandler.JSONBatchUpdateHandler)
	r.Post(`/updates/`, metricsHandler.JSONBatchUpdateHandler)
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
