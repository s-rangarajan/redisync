package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type CartUpdater interface {
	UpdateCartWithContext(context.Context, string, func(Cart) Cart) error
}

type RedisCartUpdater struct {
	client *redis.Client
}

func NewRedisCartUpdater(client *redis.Client) *RedisCartUpdater {
	return &RedisCartUpdater{
		client: client,
	}
}

func (r *RedisCartUpdater) UpdateCartWithContext(ctx context.Context, cartID string, updaterFunc func(Cart) Cart) error {
	serializedData, err := r.client.Get(ctx, cartID).Result()
	if err != nil {
		return fmt.Errorf("error getting cart from redis: %w", err)
	}

	var cart Cart
	if err := json.Unmarshal([]byte(serializedData), &cart); err != nil {
		return fmt.Errorf("error unmarshaling cart JSON: %w", err)
	}

	updaterFunc(cart)

	// lock / block
	// set
	// release

	return nil
}
