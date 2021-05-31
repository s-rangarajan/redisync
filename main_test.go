package main

import (
	"log"

	"github.com/go-redis/redis/v8"
)

func MustRedisTestClient() *redis.Client {
	client, err := NewRedisClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	if err != nil {
		log.Fatalf(err.Error())
	}

	return client
}
