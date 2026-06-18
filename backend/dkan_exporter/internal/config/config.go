package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	DKANUser     string `env:"DKAN_USER,required"`
	DKANPassword string `env:"DKAN_PASSWORD,required"`
}

// Parse loads environment variables and populates the Config struct.
func Parse() (Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("configuration error: %w", err)
	}

	return cfg, nil
}
