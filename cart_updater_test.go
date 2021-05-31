package main

import (
	"context"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func TestRedisUpdateCartWithContextReturnsErrorIfErrorAcquiringLock(t *testing.T) {
	client := MustRedisTestClient()
	updater := NewRedisCartUpdater(client)
	cartID := uuid.NewV4().String()

	// will error because deadline will be invalid given that context has no deadline
	err := updater.UpdateCartWithContext(context.Background(), cartID, func(cart *Cart) *Cart { return cart })

	require.Error(t, err)
	require.Regexp(t, "error acquiring lock", err.Error())
}

func TestRedisUpdateCartWithContextReturnsErrorIfItTimesOutWaitingOnLock(t *testing.T) {
	client := MustRedisTestClient()
	updater := NewRedisCartUpdater(client)
	cartID := uuid.NewV4().String()
	ctx, _ := context.WithTimeout(context.Background(), 50*time.Millisecond)
	// lock out semaphore token for time > ctx deadline
	// hacky, but better than having flappy tests due to races
	// trying to call multiple UpdateCartWithContext in parallel
	_, err := client.SetNX(ctx, lockingSemaphore(cartID), semaphoreToken, 60*time.Millisecond).Result()
	require.NoError(t, err)

	err = updater.UpdateCartWithContext(ctx, cartID, func(cart *Cart) *Cart { return cart })

	require.Error(t, err)
	require.Regexp(t, "timed out waiting for lock", err.Error())
}

func TestRedisUpdateCartWithContextReturnsErrorIfErrorRetrievingExistingCartFromRedis(t *testing.T) {
	client := MustRedisTestClient()
	updater := NewRedisCartUpdater(client)
	cartID := uuid.NewV4().String()
	ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, err := client.LPush(ctx, cartID, "irrelevant").Result()
	require.NoError(t, err)
	defer func() {
		client.Del(ctx, cartID).Result()
	}()

	err = updater.UpdateCartWithContext(ctx, cartID, func(cart *Cart) *Cart { return cart })

	require.Error(t, err)
	require.Regexp(t, "error getting existing cart", err.Error())
}

func TestRedisUpdateCartWithContextReturnsErrorIfExistingCartIsInvalidJSON(t *testing.T) {
	client := MustRedisTestClient()
	updater := NewRedisCartUpdater(client)
	cartID := uuid.NewV4().String()
	ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, err := client.Set(ctx, cartID, "totally not JSON", 100*time.Millisecond).Result()
	require.NoError(t, err)

	err = updater.UpdateCartWithContext(ctx, cartID, func(cart *Cart) *Cart { return cart })

	require.Error(t, err)
	require.Regexp(t, "error unmarshaling existing cart", err.Error())
}

func TestRedisUpdateCartWithContextReturnsErrorIfErrorSavingCart(t *testing.T) {
	client := MustRedisTestClient()
	updater := NewRedisCartUpdater(client)
	cartID := uuid.NewV4().String()
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Millisecond)

	err := updater.UpdateCartWithContext(
		ctx,
		cartID,
		// only way to test this, as implemented
		// is to trigger a context timeout
		func(cart *Cart) *Cart {
			<-ctx.Done()
			return cart
		},
	)

	require.Error(t, err)
	require.Regexp(t, "error saving cart", err.Error())
}

func TestRedisUpdateCartWithContextSavesCart(t *testing.T) {
	client := MustRedisTestClient()
	updater := NewRedisCartUpdater(client)
	cartID := uuid.NewV4().String()
	ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)

	err := updater.UpdateCartWithContext(
		ctx,
		cartID,
		func(cart *Cart) *Cart {
			return &Cart{
				CartID: cartID,
				CartDetails: map[ItemID]ItemDetails{
					"food": map[DinerID]int{
						"diner": 1,
					},
				},
			}
		},
	)

	require.NoError(t, err)
	reader := NewRedisCartReader(client)
	cart, err := reader.ReadCartWithContext(ctx, cartID)
	require.NoError(t, err)
	require.Equal(t, cartID, cart.CartID)
	require.Equal(t, 1, cart.CartDetails["food"]["diner"])
}

func TestRedisUpdateCartWithContextBlocksUntilLockIsAvailableAndSavesCart(t *testing.T) {
	client := MustRedisTestClient()
	updater := NewRedisCartUpdater(client)
	cartID := uuid.NewV4().String()
	ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, err := client.SetNX(ctx, lockingSemaphore(cartID), semaphoreToken, 20*time.Millisecond).Result()
	go func() {
		time.Sleep(20 * time.Millisecond)
		client.Del(ctx, lockingSemaphore(cartID))
		client.LPush(ctx, blockingSemaphore(cartID), semaphoreToken)
	}()
	require.NoError(t, err)

	err = updater.UpdateCartWithContext(
		ctx,
		cartID,
		func(cart *Cart) *Cart {
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			require.Less(t, deadline.Sub(time.Now()), 80*time.Millisecond)

			return &Cart{
				CartID: cartID,
				CartDetails: map[ItemID]ItemDetails{
					"food": map[DinerID]int{
						"diner": 1,
					},
				},
			}
		},
	)

	require.NoError(t, err)
	reader := NewRedisCartReader(client)
	cart, err := reader.ReadCartWithContext(ctx, cartID)
	require.NoError(t, err)
	require.Equal(t, cartID, cart.CartID)
	require.Equal(t, 1, cart.CartDetails["food"]["diner"])
}
