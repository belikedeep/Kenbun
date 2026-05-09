package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	DatabaseURL        string
	RedisAddrs         []string
	KafkaBrokers       []string
	ClickHouseAddr     string
	RequestTimeout     time.Duration
	MaxBodySizeKB      int
	RateLimitSyncFreq  time.Duration
	GeminiAPIKey       string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgresql://gateway:gateway@localhost:5432/gateway"),
		RedisAddrs:         strings.Split(getEnv("REDIS_ADDRS", "localhost:6379"), ","),
		KafkaBrokers:       strings.Split(getEnv("KAFKA_BROKERS", "localhost:19092"), ","),
		ClickHouseAddr:     getEnv("CLICKHOUSE_ADDR", "localhost:9000"),
		RequestTimeout:     time.Duration(getEnvInt("REQUEST_TIMEOUT_MS", 30000)) * time.Millisecond,
		MaxBodySizeKB:      getEnvInt("MAX_BODY_SIZE_KB", 512),
		RateLimitSyncFreq: time.Duration(getEnvInt("RL_SYNC_FREQ_MS", 100)) * time.Millisecond,
		GeminiAPIKey:       getEnv("GEMINI_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}
