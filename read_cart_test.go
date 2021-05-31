package main

import (
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

type MockCartReader struct {
	TestReadCartWithContext func(context.Context, string) (Cart, error)
}

func (m *MockCartReader) ReadCartWithContext(ctx context.Context, cartID string) (Cart, error) {
	return m.TestReadCartWithContext(ctx, cartID)
}

func TestReadCartWithContextReturnsErrorIfCartIDNotIncludedInRequestParams(t *testing.T) {
	request, err := http.NewRequest("GET", "/read_cart", nil)
	require.NoError(t, err)

	response := httptest.NewRecorder()
	// can be nil because should not be invoked
	ReadCartWithContext(context.Background(), &MockCartReader{}, response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestReadCartWithContextReturnsErrorIfErrorReadingCart(t *testing.T) {
	id := uuid.NewV4().String()
	request, err := http.NewRequest("GET", fmt.Sprintf("/read_cart?cart_id=%s", id), nil)
	require.NoError(t, err)

	response := httptest.NewRecorder()
	ReadCartWithContext(
		context.Background(),
		&MockCartReader{
			TestReadCartWithContext: func(ctx context.Context, cartID string) (Cart, error) {
				require.Equal(t, id, cartID)

				return Cart{}, fmt.Errorf("some error")
			},
		},
		response,
		request,
	)

	require.Equal(t, http.StatusInternalServerError, response.Code)
}

func TestReadCartWithContextReturnsErrorIfContextTimesOut(t *testing.T) {
	id := uuid.NewV4().String()
	request, err := http.NewRequest("GET", fmt.Sprintf("/read_cart?cart_id=%s", id), nil)
	require.NoError(t, err)

	ctx, _ := context.WithTimeout(context.Background(), 0*time.Second)
	response := httptest.NewRecorder()
	ReadCartWithContext(
		ctx,
		&MockCartReader{
			TestReadCartWithContext: func(ctx context.Context, cartID string) (Cart, error) {
				return Cart{}, ctx.Err()
			},
		},
		response,
		request,
	)

	require.Equal(t, http.StatusInternalServerError, response.Code)
}

func TestReadCartWithContextReturnsCart(t *testing.T) {
	id := uuid.NewV4().String()
	request, err := http.NewRequest("GET", fmt.Sprintf("/read_cart?cart_id=%s", id), nil)
	require.NoError(t, err)

	response := httptest.NewRecorder()
	ReadCartWithContext(
		context.Background(),
		&MockCartReader{
			TestReadCartWithContext: func(ctx context.Context, cartID string) (Cart, error) {
				require.Equal(t, id, cartID)

				return Cart{
					CartID: id,
					CartDetails: map[ItemID]ItemDetails{
						"food": map[DinerID]int{
							"diner": 1,
						},
					},
				}, nil
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
