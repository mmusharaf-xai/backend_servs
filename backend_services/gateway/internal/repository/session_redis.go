package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
)

type SessionRedisCache struct {
	client *redis.Client
}

func NewSessionRedisCache(client *redis.Client) *SessionRedisCache {
	return &SessionRedisCache{client: client}
}

func (c *SessionRedisCache) Get(ctx context.Context, tokenHash string) (*domain.RefreshSession, error) {
	val, err := c.client.Get(ctx, sessionKey(tokenHash)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get session: %w", err)
	}

	var s domain.RefreshSession
	if err := json.Unmarshal([]byte(val), &s); err != nil {
		return nil, fmt.Errorf("redis unmarshal session: %w", err)
	}
	return &s, nil
}

func (c *SessionRedisCache) Set(ctx context.Context, tokenHash string, session *domain.RefreshSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("redis marshal session: %w", err)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return nil // already expired, don't cache
	}

	if err := c.client.Set(ctx, sessionKey(tokenHash), data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set session: %w", err)
	}
	return nil
}

func (c *SessionRedisCache) Delete(ctx context.Context, tokenHash string) error {
	if err := c.client.Del(ctx, sessionKey(tokenHash)).Err(); err != nil {
		return fmt.Errorf("redis del session: %w", err)
	}
	return nil
}

func sessionKey(tokenHash string) string {
	return "refresh_session:" + tokenHash
}