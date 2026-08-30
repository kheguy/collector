package config

import (
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
)

type AgentConfig struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

type ServerConfig struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

// Разделил так как уже путаница началась
func LoadAgent() (AgentConfig, error) {
	cfg := AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
	}

	flags := flag.NewFlagSet("agent", flag.ContinueOnError)
	flags.StringVar(&cfg.Address, "a", cfg.Address, "Address of the server")
	flags.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval")
	flags.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval")

	if err := flags.Parse(os.Args[1:]); err != nil {
		return AgentConfig{}, err
	}
	if err := env.Parse(&cfg); err != nil {
		return AgentConfig{}, err
	}
	return cfg, nil
}

func LoadServer() (ServerConfig, error) {
	cfg := ServerConfig{
		Address:         "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "/tmp/collector-metrics.json",
		Restore:         true,
	}

	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.StringVar(&cfg.Address, "a", cfg.Address, "Address of the server")
	flags.IntVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "Store interval")
	flags.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File storage path")
	flags.BoolVar(&cfg.Restore, "r", cfg.Restore, "Restore metrics")

	if err := flags.Parse(os.Args[1:]); err != nil {
		return ServerConfig{}, err
	}
	if err := env.Parse(&cfg); err != nil {
		return ServerConfig{}, err
	}
	return cfg, nil
}
