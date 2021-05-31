package main

// TODO: define as you best see fit
type ItemID string
type DinerID string
type ItemDetails map[DinerID]int
type Cart struct {
	CartID      string                 `json:"cart_id"`
	CartDetails map[ItemID]ItemDetails `json:"cart_details"`
}

func NewCart(cartID string) Cart {
	cart := Cart{
		CartID:      cartID,
		CartDetails: make(map[ItemID]ItemDetails),
	}
	return cart
}
