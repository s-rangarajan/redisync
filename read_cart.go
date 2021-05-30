package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

var readTimeout = 2 * time.Second

func ReadCart(w http.ResponseWriter, r *http.Request, cartReader CartReader) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), updateTimeout)
	defer cancelFunc()

	cartID, ok := r.URL.Query()["cart_id"]
	if !ok || len(cartID[0]) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currentCart, err := cartReader.ReadCartWithContext(ctx, cartID[0])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(currentCart); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	return
}
