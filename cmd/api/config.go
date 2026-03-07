package main

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DSN      string // DATABASE_URL
	RedisAddr string // REDIS_ADDR
	Port     int    // PORT
}

func loadConfig() Config {
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://florist:florist@localhost:5432/florist?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	return Config{
		DSN:       dsn,
		RedisAddr: redisAddr,
		Port:      port,
	}
}
