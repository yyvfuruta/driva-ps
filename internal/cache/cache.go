package cache

import (
	"fmt"
	"os"

	"github.com/go-redis/redis/v8"
)

func NewConnection() (*redis.Client, error) {
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

	return rdb, nil
}
