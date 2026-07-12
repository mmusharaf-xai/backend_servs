package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiterRedis struct {
	client *redis.Client
	window time.Duration
}

func NewRateLimiterRedis(client *redis.Client) *RateLimiterRedis {
	return &RateLimiterRedis{client: client, window: 15 * time.Minute}
}

func (r *RateLimiterRedis) IncrementLoginAttempts(ctx context.Context, key string) (int64, error) {
	rKey := "login_attempts:" + key
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, rKey)
	pipe.Expire(ctx, rKey, r.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("increment login attempts: %w", err)
	}
	return incr.Val(), nil
}

func (r *RateLimiterRedis) GetLoginAttempts(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Get(ctx, "login_attempts:"+key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get login attempts: %w", err)
	}
	return val, nil
}
