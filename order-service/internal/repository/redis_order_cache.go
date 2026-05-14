package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"order-service/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisOrderCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisOrderCache(client *redis.Client) domain.OrderCache {
	return &redisOrderCache{
		client: client,
		ctx:    context.Background(),
	}
}

func (r *redisOrderCache) Get(id string) (*domain.Order, error) {
	key := fmt.Sprintf("order:%s", id)
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var order domain.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *redisOrderCache) Set(order *domain.Order, ttl time.Duration) error {
	key := fmt.Sprintf("order:%s", order.ID)
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, key, data, ttl).Err()
}

func (r *redisOrderCache) Delete(id string) error {
	key := fmt.Sprintf("order:%s", id)
	return r.client.Del(r.ctx, key).Err()
}
