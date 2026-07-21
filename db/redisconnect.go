package db

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() *redis.Client {
	var rdb *redis.Client
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		log.Println("redis Connection Failed", err)
		return nil
	}
	log.Println("Redis connected successfully")
	return rdb
}
