package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	PaymentService ServiceConfig
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (d DatabaseConfig) ConnectionString() string {
	return "host=" + d.Host +
		" port=" + strconv.Itoa(d.Port) +
		" user=" + d.User +
		" password=" + d.Password +
		" dbname=" + d.Name +
		" sslmode=" + d.SSLMode
}

type ServiceConfig struct {
	BaseURL string
	Timeout time.Duration
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnvInt("SERVER_PORT", 8082),
			ReadTimeout:  time.Duration(getEnvInt("SERVER_READ_TIMEOUT", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("SERVER_WRITE_TIMEOUT", 30)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnvString("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnvString("DB_USER", "acme"),
			Password: getEnvString("DB_PASSWORD", "acme"),
			Name:     getEnvString("DB_NAME", "acme_orders"),
			SSLMode:  getEnvString("DB_SSLMODE", "disable"),
		},
		PaymentService: ServiceConfig{
			BaseURL: getEnvString("PAYMENT_SERVICE_URL", "http://localhost:8083"),
			Timeout: time.Duration(getEnvInt("PAYMENT_SERVICE_TIMEOUT", 30)) * time.Second,
		},
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
