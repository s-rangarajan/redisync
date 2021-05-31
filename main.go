package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

// TODO: define as you best see fit
var readTimeout = 2 * time.Second
var updateTimeout = 2 * time.Second

func main() {
	// TODO: read from env/config file as you best see fit
	client, err := NewRedisClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	if err != nil {
		panic(err)
	}

	// TODO: replace with signal handling context
	ctx := context.TODO()

	cartUpdater := NewRedisCartUpdater(client)
	cartReader := NewRedisCartReader(client)

	// TODO: use a router of your choice and path variables instead of reqeust params
	http.HandleFunc("/read_cart", func(w http.ResponseWriter, r *http.Request) {
		// TODO: handle with signal context
		ctx, _ := context.WithTimeout(ctx, readTimeout)
		ReadCartWithContext(ctx, cartReader, w, r)
	})
	http.HandleFunc("/update_cart", func(w http.ResponseWriter, r *http.Request) {
		// TODO: handle with signal context
		ctx, _ := context.WithTimeout(ctx, readTimeout)
		UpdateCartWithContext(ctx, cartUpdater, w, r)
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
