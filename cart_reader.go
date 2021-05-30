package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type CartReader interface {
	ReadCartWithContext(context.Context, string) (Cart, error)
}

type RedisCartReader struct {
	client *redis.Client
}

func NewRedisCartReader(client *redis.Client) *RedisCartReader {
	return &RedisCartReader{
		client: client,
	}
}

func (r *RedisCartReader) ReadCartWithContext(ctx context.Context, cartID string) (Cart, error) {
	serializedData, err := r.client.Get(ctx, cartID).Result()
	if err != nil {
		return Cart{}, fmt.Errorf("error getting cart from redis: %w", err)
	}

	var cart Cart
	if err := json.Unmarshal([]byte(serializedData), &cart); err != nil {
		return Cart{}, fmt.Errorf("error unmarshaling cart JSON: %w", err)
	}

	return cart, nil
}
