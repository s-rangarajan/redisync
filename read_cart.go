package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

// TODO: log and report errors to monitoring tools appropriately
func ReadCartWithContext(ctx context.Context, cartReader CartReader, w http.ResponseWriter, r *http.Request) {
	// TODO: use path variables instead of request params
	cartID, ok := r.URL.Query()["cart_id"]
	if !ok || len(cartID[0]) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currentCart, err := cartReader.ReadCartWithContext(ctx, cartID[0])
	if err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(currentCart); err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
