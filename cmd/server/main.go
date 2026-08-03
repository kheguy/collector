package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/kheguy/collector/internal/handler"
	"github.com/kheguy/collector/internal/repository"
	"github.com/kheguy/collector/internal/service"
)

func main() {
	var appFlags = flag.NewFlagSet("app", flag.ExitOnError)
	var (
		address = appFlags.String("a", "localhost:8080", "Address of the server")
	)

	if err := appFlags.Parse(os.Args[1:]); err != nil {
		panic("Unknown flags")
	}

	r := chi.NewRouter()

	storage := repository.MakeNewMemoryStorage()

	metricsService := service.MakeNewMetricsService(storage)

	metricsHandler := handler.MakeNewMetricsHandler(metricsService)

	r.Post(`/update/{type}/{name}/{value}`, metricsHandler.UpdateHandler)
	r.Get(`/value/{type}/{name}`, metricsHandler.ValueHandler)
	r.Get(`/`, metricsHandler.HTMLListHandler)

	fmt.Printf("Server started on %s\n", *address)
	err := http.ListenAndServe(*address, r)

	if err != nil {
		panic(err)
	}
}
