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
	mongoURI := os.Getenv("MONGODB_URI")
	log.Println("Running on Port", os.Getenv("PORT"))
	if mongoURI == "" {
		log.Println("Error: MONGODB_URI environment variable is not set")
		return nil
	}

	// Set timeouts directly in client options so background connection attempts fail fast if unreachable
	clientOpts := options.Client().
		ApplyURI(mongoURI).
		SetServerSelectionTimeout(20 * time.Second). // Timeout finding MongoDB server
		SetConnectTimeout(20 * time.Second)          // Timeout establishing TCP socket

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		log.Println("Failed to create MongoDB client:", err)
		return nil
	}

	// Create context for Ping
	ctx, cancel := context.WithTimeout(context.Background(), 20	*time.Second)
	defer cancel()

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Println("Error pinging MongoDB:", err)
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
			log.Println("Failed to close MongoDB client after ping failure:", disconnectErr)
		}
		return nil
	}

	log.Println("MongoDB connected successfully")
	return client
}
