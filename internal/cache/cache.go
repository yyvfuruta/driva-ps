package cache

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	redis *redis.Client
}

func New() (*Cache, error) {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	envVars := map[string]string{
		"REDIS_HOST": host,
		"REDIS_PORT": port,
	}

	for key, value := range envVars {
		if value == "" {
			return nil, fmt.Errorf("%s environment variable not set", key)
		}
	}

	url := fmt.Sprintf("%s:%s", host, port)
	rdb := redis.NewClient(&redis.Options{Addr: url})

	return &Cache{
		redis: rdb,
	}, nil
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.redis.Get(ctx, key).Result()
}

func (c *Cache) Set(ctx context.Context, key string, value any, expirationTime time.Duration) error {
	return c.redis.Set(ctx, key, value, expirationTime).Err()
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.redis.Ping(ctx).Err()
}
