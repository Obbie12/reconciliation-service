package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	ServerAddress string `env:"SERVER_ADDRESS,required"`
	Environment   string `env:"ENVIRONMENT,required"`
	Database      DatabaseConfig
	Migration     MigrationConfig
}

type DatabaseConfig struct {
	Host     string `env:"DB_HOST,required"`
	Port     int    `env:"DB_PORT,required"`
	User     string `env:"DB_USER,required"`
	Password string `env:"DB_PASSWORD,required"`
	Name     string `env:"DB_NAME,required"`
	Params   string `env:"DB_PARAMS,required"`
}

type MigrationConfig struct {
	Dir string `env:"MIGRATION_DIR"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	config := &Config{
		ServerAddress: viper.GetString("SERVER_ADDRESS"),
		Environment:   viper.GetString("ENVIRONMENT"),
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_NAME"),
			Params:   viper.GetString("DB_PARAMS"),
		},
		Migration: MigrationConfig{
			Dir: viper.GetString("MIGRATION_DIR"),
		},
	}

	return config, nil

	// var cfg Config
	// if err := envdecode.Decode(&cfg); err != nil {
	// 	return nil, err
	// }

	// return &cfg, nil
}

// GetDSN returns the MySQL DSN string
func (c *Config) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.Params,
	)
}

// GetMigrationDBURL returns the database URL for migrations
func (c *Config) GetMigrationDBURL() string {
	return fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.Params,
	)
}
