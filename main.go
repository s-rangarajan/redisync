package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

func main() {
	client, err := NewRedisClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	if err != nil {
		panic(err)
	}

	cartUpdater := NewRedisCartUpdater(client)
	cartReader := NewRedisCartReader(client)

	http.HandleFunc("/read_cart", func(w http.ResponseWriter, r *http.Request) {
		ReadCart(w, r, cartReader)
	})
	http.HandleFunc("/update_cart", func(w http.ResponseWriter, r *http.Request) {
		UpdateCart(w, r, cartUpdater)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func NewRedisClient(options *redis.Options) (*redis.Client, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelFunc()

	client := redis.NewClient(options)
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("error setting up new redis client: %w", err)
	}

	return client, nil
}
