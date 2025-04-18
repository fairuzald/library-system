package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type Redis struct {
	client *redis.Client
	log    *logger.Logger
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	Logger   *logger.Logger
}

func NewRedis(cfg *RedisConfig) (*Redis, error) {
	log := cfg.Logger
	if log == nil {
		log = logger.Default()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info("Successfully connected to Redis", zap.String("addr", client.Options().Addr))

	return &Redis{
		client: client,
		log:    log,
	}, nil
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = r.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}

	return nil
}

func (r *Redis) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to get key: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	return nil
}

func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	return count > 0, nil
}

func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	err := r.client.Expire(ctx, key, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiration time: %w", err)
	}

	return nil
}

func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	result, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key: %w", err)
	}

	return result, nil
}

func (r *Redis) HSet(ctx context.Context, key, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = r.client.HSet(ctx, key, field, data).Err()
	if err != nil {
		return fmt.Errorf("failed to set hash field: %w", err)
	}

	return nil
}

func (r *Redis) HGet(ctx context.Context, key, field string, dest interface{}) error {
	data, err := r.client.HGet(ctx, key, field).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("field not found: %s", field)
		}
		return fmt.Errorf("failed to get hash field: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

func (r *Redis) HDelete(ctx context.Context, key, field string) error {
	err := r.client.HDel(ctx, key, field).Err()
	if err != nil {
		return fmt.Errorf("failed to delete hash field: %w", err)
	}

	return nil
}

func (r *Redis) HExists(ctx context.Context, key, field string) (bool, error) {
	exists, err := r.client.HExists(ctx, key, field).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check hash field existence: %w", err)
	}

	return exists, nil
}
