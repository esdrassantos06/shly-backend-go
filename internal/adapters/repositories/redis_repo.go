package repositories

import (
	"context"
	"github.com/esdrassantos06/go-shortener/internal/core/ports"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisRepo struct {
	Client *redis.Client
}

func NewRedisRepo(client *redis.Client) ports.CacheRepository {
	return &RedisRepo{Client: client}
}

func (r *RedisRepo) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

func (r *RedisRepo) Set(ctx context.Context, key string, value string, ttlSeconds int) error {
	return r.Client.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Err()
}

func (r *RedisRepo) IncrementCounter(ctx context.Context, key string) error {
	return r.Client.Incr(ctx, key).Err()
}
