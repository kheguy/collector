package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kheguy/collector/internal/agent"
	"github.com/kheguy/collector/internal/config"
	"github.com/kheguy/collector/internal/repository"
)

func main() {
	cfg, cfgErr := config.LoadAgent()

	if cfgErr != nil {
		log.Fatal("Config loading error: ", cfgErr)
	}

	storage := repository.MakeNewMemoryStorage()

	addressWithProtocol := cfg.Address

	if !strings.Contains(addressWithProtocol, "http://") && !strings.Contains(addressWithProtocol, "https://") {
		addressWithProtocol = "http://" + addressWithProtocol
	}

	metricsAgent := agent.NewAgent(storage, addressWithProtocol, *http.DefaultClient)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	metricsAgent.Run(ctx, time.Duration(cfg.PollInterval)*time.Second, time.Duration(cfg.ReportInterval)*time.Second)
}
