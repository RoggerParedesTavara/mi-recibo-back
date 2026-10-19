package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort                     string
	Environment                    string
	DBHost                         string
	DBPort                         string
	DBUser                         string
	DBPassword                     string
	DBName                         string
	DBSSLMode                      string
	ConsumptionAlertSpikeThreshold float64
}

func LoadConfig() (*Config, error) {
	// Load .env if present (ignore error if file doesn't exist)
	_ = godotenv.Load()

	spikeThreshold, err := strconv.ParseFloat(getEnv("CONSUMPTION_ALERT_SPIKE_THRESHOLD", "20.0"), 64)
	if err != nil {
		spikeThreshold = 20.0
	}

	cfg := &Config{
		ServerPort:                     getEnv("SERVER_PORT", "8080"),
		Environment:                    getEnv("ENVIRONMENT", "development"),
		DBHost:                         getEnv("DB_HOST", "localhost"),
		DBPort:                         getEnv("DB_PORT", "5432"),
		DBUser:                         getEnv("DB_USER", "postgres"),
		DBPassword:                     getEnv("DB_PASSWORD", "postgres"),
		DBName:                         getEnv("DB_NAME", "mi_recibo"),
		DBSSLMode:                      getEnv("DB_SSLMODE", "disable"),
		ConsumptionAlertSpikeThreshold: spikeThreshold,
	}

	return cfg, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
