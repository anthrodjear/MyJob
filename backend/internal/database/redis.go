package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type RedisClient struct {
	Client *redis.Client
	Logger *zap.Logger
}

func NewRedisClient(redisURL string, logger *zap.Logger) (*RedisClient, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	if logger != nil {
		logger.Debug("Redis client created (not yet connected)")
	}

	return &RedisClient{
		Client: client,
		Logger: logger,
	}, nil
}

// Connect tests the Redis connection and returns an error if it fails.
// Call this explicitly after NewRedisClient if you need to verify connectivity.
func (r *RedisClient) Connect(ctx context.Context) error {
	if err := r.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	if r.Logger != nil {
		r.Logger.Info("Connected to Redis")
	}
	return nil
}

func (r *RedisClient) Close() error {
	return r.Client.Close()
}

func (r *RedisClient) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Rate limiting helpers
func (r *RedisClient) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	pipe := r.Client.Pipeline()

	// Get current count
	count, err := pipe.Get(ctx, key).Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, err
	}

	if count >= limit {
		return false, nil
	}

	// Increment and set expiry
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}
