package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

var updateTimeout = 2 * time.Second

type UpdateCartRequest map[Item]Quantity

func UpdateCart(w http.ResponseWriter, r *http.Request, cartUpdater CartUpdater) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), updateTimeout)
	defer cancelFunc()

	cartID, ok := r.URL.Query()["cart_id"]
	if !ok || len(cartID[0]) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var request UpdateCartRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	var finalCart Cart
	err := cartUpdater.UpdateCartWithContext(ctx, cartID[0], func(currentCart Cart) Cart {
		finalCart = compareAndUpdateCart(currentCart, request)
		return finalCart
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(finalCart); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	return
}

func compareAndUpdateCart(currentCart Cart, request UpdateCartRequest) Cart {
	return Cart{}
}
