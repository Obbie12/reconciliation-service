package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MySQLDSN      string
	RedisAddr     string
	ServerAddress string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		MySQLDSN:      os.Getenv("MYSQL_DSN"),
		RedisAddr:     os.Getenv("REDIS_ADDR"),
		ServerAddress: os.Getenv("SERVER_ADDRESS"),
	}

	if cfg.ServerAddress == "" {
		cfg.ServerAddress = ":8080"
	}

	if cfg.MySQLDSN == "" {
		return nil, errors.New("MYSQL_DSN is required")
	}

	return cfg, nil
}
