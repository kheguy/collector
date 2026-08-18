package config

import (
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func Load() (Config, error) {
	cfg := Config{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
	}

	var appFlags = flag.NewFlagSet("app", flag.ExitOnError)

	appFlags.StringVar(&cfg.Address, "a", cfg.Address, "Address of the server")
	appFlags.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval")
	appFlags.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval")

	if err := appFlags.Parse(os.Args[1:]); err != nil {
		return Config{}, err
	}

	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
