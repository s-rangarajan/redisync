package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// TODO: define as you best see fit
const (
	semaphoreToken          = 1
	cartExpiry              = 5 * time.Minute
	blockingSemaphoreExpiry = 1 * time.Second
)

type CartUpdater interface {
	UpdateCartWithContext(context.Context, string, func(*Cart) *Cart) error
}

type RedisCartUpdater struct {
	client *redis.Client
}

func NewRedisCartUpdater(client *redis.Client) *RedisCartUpdater {
	return &RedisCartUpdater{
		client: client,
	}
}

func (r *RedisCartUpdater) UpdateCartWithContext(ctx context.Context, cartID string, updaterFunc func(*Cart) *Cart) error {
	lockingSemaphore := fmt.Sprintf("%s:%s", cartID, "mutex")
	blockingSemaphore := fmt.Sprintf("%s:%s", cartID, "block")
	for {
		// ok to ignore because ctx will expire in call anyway
		deadline, _ := ctx.Deadline()
		ok, err := r.client.SetNX(ctx, lockingSemaphore, semaphoreToken, deadline.Sub(time.Now())).Result()
		if err != nil {
			return fmt.Errorf("error acquiring lock for update: %w", err)
		}

		if !ok {
			_, err := r.client.BLPop(ctx, 0, blockingSemaphore).Result()
			if err != nil {
				return fmt.Errorf("timed out waiting for lock for update: %w", err)
			}

			continue
		}

		break
	}

	_, err := r.client.Del(ctx, blockingSemaphore).Result()
	if err != nil {
		return fmt.Errorf("error acquiring lock for update: %w", err)
	}

	cart := NewCart(cartID)
	serializedData, err := r.client.Get(ctx, cartID).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("error getting cart from redis: %w", err)
	}
	if err != redis.Nil {
		if err := json.Unmarshal([]byte(serializedData), &cart); err != nil {
			return fmt.Errorf("error unmarshaling cart from redis: %w", err)
		}
	}

	updatedCartJSON, err := json.Marshal(updaterFunc(&cart))
	if err != nil {
		return fmt.Errorf("error marshaling cart for redis: %w", err)
	}

	r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, cartID, updatedCartJSON, cartExpiry)
		pipe.Del(ctx, blockingSemaphore)
		pipe.LPush(ctx, blockingSemaphore, semaphoreToken)
		pipe.Expire(ctx, blockingSemaphore, blockingSemaphoreExpiry)
		pipe.Del(ctx, lockingSemaphore)

		return nil
	})

	return nil
}
