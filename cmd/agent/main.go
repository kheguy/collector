package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kheguy/collector/internal/agent"
	"github.com/kheguy/collector/internal/config"
	"github.com/kheguy/collector/internal/repository"
)

func main() {
	cfg, cfgErr := config.Load()

	if cfgErr != nil {
		log.Fatal("Config loading error: ", cfgErr)
	}

	storage := repository.MakeNewMemoryStorage()

	// Возможно, стоит вынести в отделбную либу
	addressWithProtocol := cfg.Address

	if !strings.Contains(addressWithProtocol, "http://") && !strings.Contains(addressWithProtocol, "https://") {
		addressWithProtocol = "http://" + addressWithProtocol
	}

	agent := agent.NewAgent(storage, addressWithProtocol, *http.DefaultClient)

	agent.Start(time.Duration(cfg.PollInterval)*time.Second, time.Duration(cfg.ReportInterval)*time.Second)

	// Мне показалось, что с сигналом будет правильнее, посмотрел в доке
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	agent.Stop()
}
