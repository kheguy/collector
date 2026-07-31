package config

type Config struct {
	Host string
	Port int
}

func Load() Config {
	return Config{
		Host: "localhost",
		Port: 8080,
	}
}
