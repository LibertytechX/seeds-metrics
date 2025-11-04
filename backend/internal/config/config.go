package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig // SeedsMetrics database (read-write)
	DjangoDatabase DatabaseConfig // Django database (read-only)
	Redis          RedisConfig
	CORS           CORSConfig
	Logging        LoggingConfig
	ETL            ETLConfig
	Metrics        MetricsConfig
}

type ServerConfig struct {
	Port    string
	Host    string
	GinMode string
}

type DatabaseConfig struct {
	Host               string
	Port               string
	User               string
	Password           string
	DBName             string
	SSLMode            string
	MaxConnections     int
	MaxIdleConnections int
	ConnMaxLifetime    time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	CacheTTL int
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

type LoggingConfig struct {
	Level  string
	Format string
}

type ETLConfig struct {
	BatchSize      int
	WorkerInterval time.Duration
}

type MetricsConfig struct {
	CalculationInterval time.Duration
	CacheEnabled        bool
}

func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Port:    getEnv("SERVER_PORT", "8080"),
			Host:    getEnv("SERVER_HOST", "0.0.0.0"),
			GinMode: getEnv("GIN_MODE", "release"),
		},
		Database: DatabaseConfig{
			Host:               getEnv("DB_HOST", "localhost"),
			Port:               getEnv("DB_PORT", "5432"),
			User:               getEnv("DB_USER", "analytics_user"),
			Password:           getEnv("DB_PASSWORD", "analytics_password"),
			DBName:             getEnv("DB_NAME", "analytics_db"),
			SSLMode:            getEnv("DB_SSLMODE", "disable"),
			MaxConnections:     getEnvAsInt("DB_MAX_CONNECTIONS", 25),
			MaxIdleConnections: getEnvAsInt("DB_MAX_IDLE_CONNECTIONS", 5),
			ConnMaxLifetime:    getEnvAsDuration("DB_CONNECTION_MAX_LIFETIME", 5*time.Minute),
		},
		DjangoDatabase: DatabaseConfig{
			Host:               getEnv("DJANGO_DB_HOST", "localhost"),
			Port:               getEnv("DJANGO_DB_PORT", "5432"),
			User:               getEnv("DJANGO_DB_USER", "metricsuser"),
			Password:           getEnv("DJANGO_DB_PASSWORD", ""),
			DBName:             getEnv("DJANGO_DB_NAME", "savings"),
			SSLMode:            getEnv("DJANGO_DB_SSLMODE", "require"),
			MaxConnections:     getEnvAsInt("DJANGO_DB_MAX_CONNECTIONS", 10),
			MaxIdleConnections: getEnvAsInt("DJANGO_DB_MAX_IDLE_CONNECTIONS", 2),
			ConnMaxLifetime:    getEnvAsDuration("DJANGO_DB_CONNECTION_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			CacheTTL: getEnvAsInt("REDIS_CACHE_TTL", 900),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
			AllowedMethods: getEnvAsSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowedHeaders: getEnvAsSlice("CORS_ALLOWED_HEADERS", []string{"Origin", "Content-Type", "Accept", "Authorization"}),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		ETL: ETLConfig{
			BatchSize:      getEnvAsInt("ETL_BATCH_SIZE", 1000),
			WorkerInterval: getEnvAsDuration("ETL_WORKER_INTERVAL", 15*time.Minute),
		},
		Metrics: MetricsConfig{
			CalculationInterval: getEnvAsDuration("METRICS_CALCULATION_INTERVAL", 30*time.Minute),
			CacheEnabled:        getEnvAsBool("METRICS_CACHE_ENABLED", true),
		},
	}

	return config, nil
}

func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	var result []string
	for _, v := range splitString(valueStr, ",") {
		result = append(result, v)
	}
	return result
}

func splitString(s, sep string) []string {
	var result []string
	current := ""
	for _, char := range s {
		if string(char) == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
