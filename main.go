package main

import (
	"log"
	"net/http"
	"os"

	"github.com/coDing-Knight99/GoChat/controllers"
	"github.com/coDing-Knight99/GoChat/db"
	"github.com/coDing-Knight99/GoChat/middleware"
	"github.com/coDing-Knight99/GoChat/models"
	"github.com/coDing-Knight99/GoChat/workers"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func main() {
	godotenv.Load()
	var client *mongo.Client
	client = db.ConnectDB()
	if client == nil {
		log.Println("Mongo DB Connection Failed")
		return
	}
	collection := client.Database("test").Collection("Message")
	userCollection := client.Database("test").Collection("Users")
	var rdb *redis.Client
	rdb = db.ConnectRedis()
	h := models.NewHub(rdb)
	go h.StartHub()
	w := workers.NewDBWorker(rdb, collection)
	go w.Start()
	http.HandleFunc("/ws", middleware.VerifyToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("websocket request")
		controllers.HandleWebSocket(h, w, r)
	})))
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Register User")
		controllers.RegisterUser(userCollection, w, r)
	})
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Login User")
		controllers.Login(userCollection, w, r)
	})
	err := http.ListenAndServe(os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal("ListenAndServeError:", err)
	}
}
