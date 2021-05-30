package main

type Order map[Item]int

func NewOrder(cart Cart) Order {
	order := make(Order)
	for item := range cart.State {
		var total int
		for diner := range cart.State[item] {
			total += cart.State[item][diner]
		}

		order[item] = total
	}

	return order
}
