package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server              ServerConfig
	Database            DatabaseConfig
	Redis               RedisConfig
	Kafka               KafkaConfig
	PaymentService      ServiceConfig
	UserService         ServiceConfig
	NotificationService ServiceConfig
	Features            FeatureFlags
	TaxRate             float64
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

func (d DatabaseConfig) ConnectionString() string {
	return "host=" + d.Host +
		" port=" + strconv.Itoa(d.Port) +
		" user=" + d.User +
		" password=" + d.Password +
		" dbname=" + d.Name +
		" sslmode=" + d.SSLMode
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	TTL      time.Duration
}

type KafkaConfig struct {
	Brokers       []string
	ConsumerGroup string
	OrdersTopic   string
	PaymentsTopic string
}

type ServiceConfig struct {
	BaseURL string
	Timeout time.Duration
	APIKey  string
}

type FeatureFlags struct {
	EnableV1API          bool
	EnableLegacyPayments bool
	EnableOrderEvents    bool
	EnableOrderCaching   bool
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnvInt("SERVER_PORT", 8082),
			ReadTimeout:  time.Duration(getEnvInt("SERVER_READ_TIMEOUT", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("SERVER_WRITE_TIMEOUT", 30)) * time.Second,
			IdleTimeout:  time.Duration(getEnvInt("SERVER_IDLE_TIMEOUT", 60)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:         getEnvString("DB_HOST", "localhost"),
			Port:         getEnvInt("DB_PORT", 5432),
			User:         getEnvString("DB_USER", "acme"),
			Password:     getEnvString("DB_PASSWORD", "acme"),
			Name:         getEnvString("DB_NAME", "acme_orders"),
			SSLMode:      getEnvString("DB_SSLMODE", "disable"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
			MaxLifetime:  time.Duration(getEnvInt("DB_MAX_LIFETIME", 5)) * time.Minute,
		},
		Redis: RedisConfig{
			Host:     getEnvString("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnvString("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			TTL:      time.Duration(getEnvInt("REDIS_TTL", 300)) * time.Second,
		},
		Kafka: KafkaConfig{
			Brokers:       []string{getEnvString("KAFKA_BROKERS", "localhost:9092")},
			ConsumerGroup: getEnvString("KAFKA_CONSUMER_GROUP", "orders-service"),
			OrdersTopic:   getEnvString("KAFKA_ORDERS_TOPIC", "orders"),
			PaymentsTopic: getEnvString("KAFKA_PAYMENTS_TOPIC", "payments"),
		},
		PaymentService: ServiceConfig{
			BaseURL: getEnvString("PAYMENT_SERVICE_URL", "http://localhost:8083"),
			Timeout: time.Duration(getEnvInt("PAYMENT_SERVICE_TIMEOUT", 30)) * time.Second,
			APIKey:  getEnvString("PAYMENT_SERVICE_API_KEY", ""),
		},
		UserService: ServiceConfig{
			BaseURL: getEnvString("USER_SERVICE_URL", "http://localhost:8081"),
			Timeout: time.Duration(getEnvInt("USER_SERVICE_TIMEOUT", 10)) * time.Second,
			APIKey:  getEnvString("USER_SERVICE_API_KEY", ""),
		},
		NotificationService: ServiceConfig{
			BaseURL: getEnvString("NOTIFICATION_SERVICE_URL", "http://localhost:8084"),
			Timeout: time.Duration(getEnvInt("NOTIFICATION_SERVICE_TIMEOUT", 10)) * time.Second,
			APIKey:  getEnvString("NOTIFICATION_SERVICE_API_KEY", ""),
		},
		Features: FeatureFlags{
			EnableV1API:          getEnvBool("ENABLE_V1_API", true),
			EnableLegacyPayments: getEnvBool("ENABLE_LEGACY_PAYMENTS", true),
			EnableOrderEvents:    getEnvBool("ENABLE_ORDER_EVENTS", true),
			EnableOrderCaching:   getEnvBool("ENABLE_ORDER_CACHING", true),
		},
		// Updated by platform team in Q4 2023
		TaxRate: getEnvFloat("TAX_RATE", 0.088),
	}
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
