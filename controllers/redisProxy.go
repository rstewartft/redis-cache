package controllers

import (
	"context"
	"errors"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
)

type (
	RedisProxy interface {
		Get(ctx context.Context, key string) (string, error)
	}

	redisProxy struct {
		cache       *expirable.LRU[string, string]
		redisClient *redis.Client
	}
)

func NewRedisProxyController(cache *expirable.LRU[string, string], redisClient *redis.Client) RedisProxy {
	return &redisProxy{
		cache:       cache,
		redisClient: redisClient,
	}
}

// Get retrieves a value from the cache or redis
// if the key is not currently in the cache, it will be added
func (c *redisProxy) Get(ctx context.Context, key string) (string, error) {
	// check to see if the key is in the cache
	if value, ok := c.cache.Get(key); ok {
		return value, nil
	}

	// if the key is not already in the cache, check redis
	value, err := c.redisClient.Get(ctx, key).Result()
	switch {
	case errors.Is(err, redis.Nil):
		return "", errors.New("key not found")
	default:
		if err != nil {
			return "", err
		}
	}
	// add the key to the cache
	c.cache.Add(key, value)
	return value, nil
}
