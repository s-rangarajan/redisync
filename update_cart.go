package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// TODO: define as you best see fit
var updateTimeout = 2 * time.Second

// TODO: log and report errors to monitoring tools appropriately
func UpdateCart(w http.ResponseWriter, r *http.Request, cartUpdater CartUpdater) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), updateTimeout)
	defer cancelFunc()

	var updates Cart
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&updates); err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	var finalCart *Cart
	// TODO: move cartID to path variable
	err := cartUpdater.UpdateCartWithContext(ctx, updates.CartID, func(currentCart *Cart) *Cart {
		finalCart = compareAndUpdateCart(currentCart, updates)
		return finalCart
	})
	if err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(finalCart); err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// TODO: define conflict resolution logic as you best see fit
func compareAndUpdateCart(currentCart *Cart, updates Cart) *Cart {
	for itemID := range updates.CartDetails {
		if _, ok := currentCart.CartDetails[itemID]; !ok {
			currentCart.CartDetails[itemID] = updates.CartDetails[itemID]
			continue
		}

		for dinerID := range updates.CartDetails[itemID] {
			currentCart.CartDetails[itemID][dinerID] = updates.CartDetails[itemID][dinerID]
		}
	}

	return currentCart
}
