package db

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() *redis.Client {
	var rdb *redis.Client
	if os.Getenv("Environment") == "Production" {
		opt, _ := redis.ParseURL(os.Getenv("Redis_url"))
		rdb = redis.NewClient(opt)
		err := rdb.Ping(context.Background()).Err()
		if err != nil {
			log.Println("redis Connection Failed", err)
			return nil
		}
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:     os.Getenv("Redis_Client"),
			Username: os.Getenv(("Redis_Username")),
			Password: os.Getenv("Redis_Password"),
			DB:       0,
		})
	}
	log.Println("Redis connected successfully")
	rdb.Set(context.Background(), "foo", "bar", 0)
	val := rdb.Get(context.Background(), "foo").Val()
	print(val)
	return rdb
}
