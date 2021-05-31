package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

type MockCartUpdater struct {
	TestUpdateCartWithContext func(context.Context, string, func(*Cart) *Cart) error
}

func (m *MockCartUpdater) UpdateCartWithContext(ctx context.Context, cartID string, updaterFunc func(*Cart) *Cart) error {
	return m.TestUpdateCartWithContext(ctx, cartID, updaterFunc)
}

func TestUpdateCartWithContextReturnsErrorIfRequestIsNotJSON(t *testing.T) {
	request, err := http.NewRequest("POST", "/update_cart", bytes.NewBuffer([]byte("totally not JSON")))
	require.NoError(t, err)

	response := httptest.NewRecorder()
	// can be nil because should not be invoked
	UpdateCartWithContext(context.Background(), &MockCartUpdater{}, response, request)

	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
}

func TestUpdateCartWithContextReturnsErrorIfInvalidJSON(t *testing.T) {
	// cart_id should be a string
	request, err := http.NewRequest("POST", "/update_cart", bytes.NewBuffer([]byte(`{"cart_id": 1}`)))
	require.NoError(t, err)

	response := httptest.NewRecorder()
	// can be nil because should not be invoked
	UpdateCartWithContext(context.Background(), &MockCartUpdater{}, response, request)

	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
}

func TestUpdateCartWithContextReturnsErrorIfErrorUpdatingCart(t *testing.T) {
	id := uuid.NewV4().String()
	request, err := http.NewRequest("POST", "/update_cart", bytes.NewBuffer([]byte(fmt.Sprintf(`{"cart_id": "%s"}`, id))))
	require.NoError(t, err)

	response := httptest.NewRecorder()
	UpdateCartWithContext(
		context.Background(),
		&MockCartUpdater{
			TestUpdateCartWithContext: func(ctx context.Context, cartID string, _ func(*Cart) *Cart) error {
				require.Equal(t, id, cartID)

				return fmt.Errorf("some error")
			},
		},
		response,
		request,
	)

	require.Equal(t, http.StatusInternalServerError, response.Code)
}

func TestUpdateCartWithContextReturnsErrorIfContextTimesOut(t *testing.T) {
	id := uuid.NewV4().String()
	request, err := http.NewRequest("POST", "/update_cart", bytes.NewBuffer([]byte(fmt.Sprintf(`{"cart_id": "%s"}`, id))))
	require.NoError(t, err)

	ctx, _ := context.WithTimeout(context.Background(), 0*time.Second)
	response := httptest.NewRecorder()
	// can be nil because should not be invoked
	UpdateCartWithContext(
		ctx,
		&MockCartUpdater{
			TestUpdateCartWithContext: func(ctx context.Context, cartID string, updaterFunc func(*Cart) *Cart) error {
				return ctx.Err()
			},
		},
		response,
		request,
	)

	require.Equal(t, http.StatusInternalServerError, response.Code)
}

func TestUpdateCartWithContextReplacesEmptyCart(t *testing.T) {
	id := uuid.NewV4().String()
	request, err := http.NewRequest(
		"POST",
		"/update_cart",
		bytes.NewBuffer([]byte(fmt.Sprintf(`{"cart_id": "%s", "cart_details": {"food": {"diner": 1}}}`, id))),
	)
	require.NoError(t, err)

	response := httptest.NewRecorder()
	// can be nil because should not be invoked
	UpdateCartWithContext(
		context.Background(),
		&MockCartUpdater{
			TestUpdateCartWithContext: func(ctx context.Context, cartID string, updaterFunc func(*Cart) *Cart) error {
				require.Equal(t, id, cartID)

				cart := NewCart(id)
				updaterFunc(&cart)

				return nil
			},
		},
		response,
		request,
	)

	require.Equal(t, http.StatusOK, response.Code)
	var cart Cart
	require.NoError(t, json.NewDecoder(response.Body).Decode(&cart))
	require.Equal(t, id, cart.CartID)
	require.Equal(t, 1, cart.CartDetails["food"]["diner"])
}

func TestUpdateCartWithContextModifiesExistingCart(t *testing.T) {
	id := uuid.NewV4().String()
	request, err := http.NewRequest(
		"POST",
		"/update_cart",
		bytes.NewBuffer([]byte(fmt.Sprintf(`{"cart_id": "%s", "cart_details": {"food": {"diner2": 1}}}`, id))),
	)
	require.NoError(t, err)

	response := httptest.NewRecorder()
	// can be nil because should not be invoked
	UpdateCartWithContext(
		context.Background(),
		&MockCartUpdater{
			TestUpdateCartWithContext: func(ctx context.Context, cartID string, updaterFunc func(*Cart) *Cart) error {
				require.Equal(t, id, cartID)

				updaterFunc(&Cart{
					CartID: id,
					CartDetails: map[ItemID]ItemDetails{
						"food": map[DinerID]int{
							"diner1": 1,
							"diner2": 2,
						},
					},
				})

				return nil
			},
		},
		response,
		request,
	)

	require.Equal(t, http.StatusOK, response.Code)
	var cart Cart
	require.NoError(t, json.NewDecoder(response.Body).Decode(&cart))
	require.Equal(t, id, cart.CartID)
	require.Equal(t, 1, cart.CartDetails["food"]["diner1"])
	require.Equal(t, 1, cart.CartDetails["food"]["diner2"])
}

// TODO: implement tests for compareAndUpdateCart
