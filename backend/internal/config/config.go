package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port string

	DatabaseURL string

	Neo4jURI      string
	Neo4jUser     string
	Neo4jPassword string

	RedisURL string

	JWTSecret          string
	JWTAccessTTLSec    int
	JWTRefreshTTLSec   int

	TMDBAPIKey string

	AxonURL    string
	AxonAPIKey string
	AxonModel  string

	LangfuseURL       string
	LangfusePublicKey string
	LangfuseSecretKey string

	LogLevel string
}

// Load reads configuration from environment variables. Returns an error if any
// required variable is missing.
func Load() (*Config, error) {
	c := &Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      mustGetEnv("DATABASE_URL"),
		Neo4jURI:         mustGetEnv("NEO4J_URI"),
		Neo4jUser:        mustGetEnv("NEO4J_USER"),
		Neo4jPassword:    mustGetEnv("NEO4J_PASSWORD"),
		RedisURL:         mustGetEnv("REDIS_URL"),
		JWTSecret:        mustGetEnv("JWT_SECRET"),
		TMDBAPIKey:       getEnv("TMDB_API_KEY", ""), // optional at startup; required only for ingestion
		AxonURL:           mustGetEnv("AXON_URL"),
		AxonAPIKey:        getEnv("AXON_API_KEY", ""),
		AxonModel:         getEnv("AXON_MODEL", "claude-opus-4-6"),
		LangfuseURL:       getEnv("LANGFUSE_URL", ""),
		LangfusePublicKey: getEnv("LANGFUSE_PUBLIC_KEY", ""),
		LangfuseSecretKey: getEnv("LANGFUSE_SECRET_KEY", ""),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		JWTAccessTTLSec:  getEnvInt("JWT_ACCESS_TTL_SEC", 900),   // 15 min
		JWTRefreshTTLSec: getEnvInt("JWT_REFRESH_TTL_SEC", 2592000), // 30 days
	}

	var missing []string
	checkRequired := map[string]string{
		"DATABASE_URL":   c.DatabaseURL,
		"NEO4J_URI":      c.Neo4jURI,
		"NEO4J_USER":     c.Neo4jUser,
		"NEO4J_PASSWORD": c.Neo4jPassword,
		"REDIS_URL":      c.RedisURL,
		"JWT_SECRET":     c.JWTSecret,
		"AXON_URL":       c.AxonURL,
	}
	for k, v := range checkRequired {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return c, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	return os.Getenv(key)
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
