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
	for {
		// ok to ignore because ctx will expire in call anyway
		deadline, _ := ctx.Deadline()
		ok, err := r.client.SetNX(ctx, lockingSemaphore(cartID), semaphoreToken, deadline.Sub(time.Now())).Result()
		if err != nil {
			return fmt.Errorf("error acquiring lock for update: %w", err)
		}

		if !ok {
			_, err := r.client.BLPop(ctx, 0, blockingSemaphore(cartID)).Result()
			if err != nil {
				return fmt.Errorf("timed out waiting for lock for update: %w", err)
			}

			continue
		}

		break
	}

	_, err := r.client.Del(ctx, blockingSemaphore(cartID)).Result()
	if err != nil {
		return fmt.Errorf("error acquiring lock for update: %w", err)
	}

	cart := NewCart(cartID)
	serializedData, err := r.client.Get(ctx, cartID).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("error getting existing cart from redis: %w", err)
	}
	if err != redis.Nil {
		if err := json.Unmarshal([]byte(serializedData), &cart); err != nil {
			return fmt.Errorf("error unmarshaling existing cart from redis: %w", err)
		}
	}

	// as implemented the cart cannot failing marshaling
	// and so cannot be tested easily without hacks and
	// so we will not test this error path
	updatedCartJSON, err := json.Marshal(updaterFunc(&cart))
	if err != nil {
		return fmt.Errorf("error marshaling cart for redis: %w", err)
	}

	cmdErrs, err := r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, cartID, updatedCartJSON, cartExpiry).Result()
		pipe.Del(ctx, blockingSemaphore(cartID)).Result()
		pipe.LPush(ctx, blockingSemaphore(cartID), semaphoreToken)
		pipe.Expire(ctx, blockingSemaphore(cartID), blockingSemaphoreExpiry)
		pipe.Del(ctx, lockingSemaphore(cartID))

		return nil
	})

	if err != nil {
		return fmt.Errorf("error saving cart in redis: %w", err)
	}

	for index := range cmdErrs {
		if cmdErrs[index].Err() != nil {
			return fmt.Errorf("error saving cart in redis: %w", cmdErrs[index].Err())
		}
	}

	return nil
}

// internal function extracted purely for use in tests
func lockingSemaphore(cartID string) string {
	return fmt.Sprintf("%s:%s", cartID, "mutex")
}

// internal function extracted purely for use in tests
func blockingSemaphore(cartID string) string {
	return fmt.Sprintf("%s:%s", cartID, "block")
}
