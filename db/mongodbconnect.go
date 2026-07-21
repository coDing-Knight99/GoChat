package db

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func ConnectDB() *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		log.Println(err)
		return nil
	}
	err1 := client.Ping(ctx, nil)
	if err1 != nil {
		log.Println("Error in Pinging MongoDB",err1)
		return nil
	}
	log.Println("Mongo DB connected successfully")
	return client
}
