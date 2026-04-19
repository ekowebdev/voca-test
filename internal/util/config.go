package util

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv  string
	Port    string
	DBConn  string
	DBPool               DBPoolConfig
	Origins              []string
	IdempotencyRetention int // in hours
	IdempotencyInterval  int // in hours
}

type DBPoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime int // in minutes
	MaxConnIdleTime int // in minutes
}

// LoadConfig loads configuration from environment variables
func LoadConfig(filenames ...string) *Config {
	if err := godotenv.Load(filenames...); err != nil {
		log.Println("No .env file found (or failed to load), using system environment variables")
	}

	config := &Config{
		AppEnv: getEnv("APP_ENV", "development"),
		Port:   getEnv("PORT", "8080"),
		DBConn: getDBConnString(),
		DBPool: DBPoolConfig{
			MaxConns:        getEnvAsInt32("DB_MAX_CONNS", 20),
			MinConns:        getEnvAsInt32("DB_MIN_CONNS", 2),
			MaxConnLifetime: getEnvAsInt("DB_MAX_CONN_LIFETIME", 30),
			MaxConnIdleTime: getEnvAsInt("DB_MAX_CONN_IDLE_TIME", 5),
		},
		Origins:              []string{"*"},
		IdempotencyRetention: getEnvAsInt("IDEMPOTENCY_RETENTION_HOURS", 24),
		IdempotencyInterval:  getEnvAsInt("IDEMPOTENCY_CLEANUP_INTERVAL_HOURS", 1),
	}

	return config
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}

func getEnvAsInt32(key string, fallback int32) int32 {
	return int32(getEnvAsInt(key, int(fallback)))
}

func getDBConnString() string {
	return "postgres://" +
		getEnv("DB_USER", "postgres") + ":" +
		getEnv("DB_PASSWORD", "") + "@" +
		getEnv("DB_HOST", "localhost") + ":" +
		getEnv("DB_PORT", "5432") + "/" +
		getEnv("DB_NAME", "voca_test") + "?sslmode=disable"
}
