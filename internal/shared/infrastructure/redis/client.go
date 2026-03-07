package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewClient creates and validates a go-redis universal client.
// addr format: "host:port" e.g. "localhost:6379"
func NewClient(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return client, nil
}

// MustNewClient panics if Redis is unavailable — use in main() only.
func MustNewClient(ctx context.Context, addr string) *redis.Client {
	c, err := NewClient(ctx, addr)
	if err != nil {
		panic(err)
	}
	return c
}
