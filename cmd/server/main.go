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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kheguy/collector/internal/config"
	"github.com/kheguy/collector/internal/handler"
	"github.com/kheguy/collector/internal/middlewares"
	"github.com/kheguy/collector/internal/repository"
	"github.com/kheguy/collector/internal/service"
	"github.com/kheguy/collector/internal/templates"
	"github.com/kheguy/collector/migrations"
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
	if cfg.DBAddress != "" {
		if err := migrations.Up(cfg.DBAddress); err != nil {
			log.Fatal("Database migration error: ", err)
		}

		// В общем, тут сначала был Coon, но я почитал, что безопаснее пулл, поэтому вот так
		pool, err := pgxpool.New(context.Background(), cfg.DBAddress)
		if err != nil {
			log.Fatal("Unable to connect to database: ", err)
		}
		defer pool.Close()

		storage = repository.NewDBStorage(pool)

		// Он же по сути только для кейса с БД нужен
		healthHandler := handler.MakeNewHealthHandler(pool)
		r.Get(`/ping`, healthHandler.PingHandler)
	} else {
		storage = repository.NewMemoryStorage()
		fileProccessor := repository.NewFileProcessor(cfg.FileStoragePath)

		if cfg.Restore {
			restoredData, err := fileProccessor.Restore()
			if err != nil {
				log.Fatal("Metrics restore error: ", err)
			}
			storage.SetAll(restoredData, context.Background())
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

	metricsService := service.MakeNewMetricsService(storage)

	metricsHandler := handler.MakeNewMetricsHandler(metricsService, renderer)

	handler.MakeNewCommonHandler()

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
