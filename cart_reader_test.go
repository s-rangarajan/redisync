package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func TestRedisReadCartWithContextReturnsErrorIfErrorRetrievingKeyFromRedis(t *testing.T) {
	client := MustRedisTestClient()
	reader := NewRedisCartReader(client)
	cartID := uuid.NewV4().String()
	_, err := client.LPush(context.Background(), cartID, "irrelevant").Result()
	require.NoError(t, err)
	defer func() {
		client.Del(context.Background(), cartID).Result()
	}()

	_, err = reader.ReadCartWithContext(context.Background(), cartID)

	require.Error(t, err)
	require.Regexp(t, "error getting cart", err.Error())
}

func TestRedisReadCartWithContextReturnsErrorIfReturnedDataIsNotJSONSerialized(t *testing.T) {
	client := MustRedisTestClient()
	reader := NewRedisCartReader(client)
	cartID := uuid.NewV4().String()
	_, err := client.Set(context.Background(), cartID, "totally not JSON", 100*time.Millisecond).Result()
	require.NoError(t, err)

	_, err = reader.ReadCartWithContext(context.Background(), cartID)

	require.Error(t, err)
	require.Regexp(t, "error unmarshaling cart", err.Error())
}

func TestRedisReadCartWithContextReturnsErrorIfReturnedDataIsInvalidJSON(t *testing.T) {
	client := MustRedisTestClient()
	reader := NewRedisCartReader(client)
	cartID := uuid.NewV4().String()
	// cart_id should be string
	_, err := client.Set(context.Background(), cartID, `{"cart_id": 1}`, 100*time.Millisecond).Result()
	require.NoError(t, err)

	_, err = reader.ReadCartWithContext(context.Background(), cartID)

	require.Error(t, err)
	require.Regexp(t, "error unmarshaling cart", err.Error())
}

func TestRedisReadCartWithContextReturnsErrorIfContextHasExpired(t *testing.T) {
	client := MustRedisTestClient()
	reader := NewRedisCartReader(client)
	cartID := uuid.NewV4().String()
	ctx, _ := context.WithTimeout(context.Background(), 0*time.Second)
	_, err := client.Set(context.Background(), cartID, `"valid JSON"`, 100*time.Millisecond).Result()
	require.NoError(t, err)

	_, err = reader.ReadCartWithContext(ctx, cartID)

	require.Error(t, err)
	require.Regexp(t, "context deadline exceeded", err.Error())
}

func TestRedisReadCartWithContextReturnsAnEmptyCartIfNotPresent(t *testing.T) {
	reader := NewRedisCartReader(MustRedisTestClient())
	cartID := uuid.NewV4().String()

	cart, err := reader.ReadCartWithContext(context.Background(), cartID)

	require.NoError(t, err)
	require.Equal(t, cartID, cart.CartID)
	require.Empty(t, cart.CartDetails)
}

func TestRedisReadCartWithContextReturnsSavedCartIfPresentAndValidJSON(t *testing.T) {
	client := MustRedisTestClient()
	reader := NewRedisCartReader(client)
	cartID := uuid.NewV4().String()
	ctx, _ := context.WithTimeout(context.Background(), 100*time.Second)
	_, err := client.Set(
		context.Background(),
		cartID,
		fmt.Sprintf(`{"cart_id": "%s", "cart_details": {"food": {"diner": 1}}}`, cartID),
		100*time.Millisecond,
	).Result()
	require.NoError(t, err)

	cart, err := reader.ReadCartWithContext(ctx, cartID)

	require.NoError(t, err)
	require.Equal(t, cartID, cart.CartID)
	require.Equal(t, 1, cart.CartDetails["food"]["diner"])
}
