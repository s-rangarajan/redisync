package main

import (
	uuid "github.com/satori/go.uuid"
)

type Item struct {
	ItemID   int    `json:"item_id"`
	ItemName string `json:"item_name"`
}

type Diner struct {
	DinerID   int    `json:"diner_id"`
	DinerName string `json:"diner_name"`
}

type Quantity map[Diner]int
type Cart struct {
	CartID string            `json:"cart_id"`
	State  map[Item]Quantity `json:"state"`
}

func NewCart() Cart {
	cart := Cart{
		CartID: uuid.NewV4().String(),
		State:  make(map[Item]Quantity),
	}
	return cart
}

func (c Cart) SetItemAmountForDiner(item Item, diner Diner, amount int) {
	if amount < 0 {
		return
	}

	if _, ok := c.State[item]; !ok {
		c.State[item] = Quantity{
			diner: amount,
		}

		return
	}

	c.State[item][diner] = amount
}
