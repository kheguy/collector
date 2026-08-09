package config

import (
	"flag"
	"os"
)

type Config struct {
	Address        string
	ReportInterval int
	PollInterval   int
}

func Load() (Config, error) {
	var cfg Config
	var appFlags = flag.NewFlagSet("app", flag.ExitOnError)

	var (
		address        = appFlags.String("a", "localhost:8080", "Address of the server")
		reportInterval = appFlags.Int("r", 10, "Report interval")
		pollInterval   = appFlags.Int("p", 2, "Poll interval")
	)

	if err := appFlags.Parse(os.Args[1:]); err != nil {
		return cfg, err
	}

	return Config{
		*address,
		*reportInterval,
		*pollInterval,
	}, nil
}
