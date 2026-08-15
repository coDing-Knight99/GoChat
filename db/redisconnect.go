package db

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() *redis.Client {
	var rdb *redis.Client
	rdb = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("Redis_Client"),
		Username: os.Getenv(("Redis_Username")),
		Password: os.Getenv("Redis_Password"),
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
