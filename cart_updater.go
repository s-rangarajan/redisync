package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	for {
		deadline, ok := ctx.Deadline()
		if !ok {
			return ctx.Err()
		}

		ok, err := r.client.SetNX(ctx, fmt.Sprintf("%s:%s", cartID, "mutex"), 1, time.Now().Sub(deadline)).Result()
		if err != nil {
			return fmt.Errorf("error acquiring lock for update: %w", err)
		}

		if !ok {
			_, err := r.client.BLPop(ctx, 0, fmt.Sprintf("%s:%s", cartID, "block")).Result()
			if err != nil {
				return fmt.Errorf("timed out waiting for lock for update: %w", err)
			}

			continue
		}

		break
	}

	_, err := r.client.Del(ctx, fmt.Sprintf("%s:%s", cartID, "block")).Result()
	if err != nil {
		return fmt.Errorf("error acquiring lock for update: %w", err)
	}

	serializedData, err := r.client.Get(ctx, cartID).Result()
	if err != nil {
		return fmt.Errorf("error getting cart from redis: %w", err)
	}

	var cart Cart
	if err := json.Unmarshal([]byte(serializedData), &cart); err != nil {
		return fmt.Errorf("error unmarshaling cart from redis: %w", err)
	}

	updatedCartJSON, err := json.Marshal(updaterFunc(cart))
	if err != nil {
		return fmt.Errorf("error marshaling cart for redis: %w", err)
	}

	r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, cartID, updatedCartJSON, 5*time.Minute)
		pipe.Del(ctx, fmt.Sprintf("%s:%s", cartID, "block"))
		pipe.LPush(ctx, fmt.Sprintf("%s:%s", cartID, "block"), 1)
		pipe.Expire(ctx, fmt.Sprintf("%s:%s", cartID, "block"), 1*time.Second)
		pipe.Del(ctx, fmt.Sprintf("%s:%s", cartID, "mutex"))

		return nil
	})

	return nil
}
