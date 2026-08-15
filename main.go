package main

import (
	"log"
	"net/http"

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
	msgcollection := client.Database("test").Collection("Message")
	userCollection := client.Database("test").Collection("Users")
	var rdb *redis.Client
	rdb = db.ConnectRedis()
	h := models.NewHub(rdb)
	go h.StartHub()
	w := workers.NewDBWorker(rdb, msgcollection)
	go w.Start()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", middleware.VerifyToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("websocket request")
		controllers.HandleWebSocket(h, w, r)
	})))
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Register User")
		controllers.RegisterUser(userCollection, w, r)
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Login User")
		controllers.Login(userCollection, w, r)
	})
	mux.HandleFunc("/checkavailable", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Username  availability check")
		controllers.CheckUsernameAvailability(userCollection, w, r)
	})
	mux.HandleFunc("/conversations", middleware.VerifyToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Fetching Conversations")
		controllers.GetConversations(msgcollection, w, r)
	})))
	mux.HandleFunc("/messages", middleware.VerifyToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Fetching Messages")
		controllers.HandleGetMessagesMongo(msgcollection, w, r)
	})))
	mux.HandleFunc("/users/search", middleware.VerifyToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Search User")
		controllers.SearchUser(userCollection, w, r)
	})))
	mux.HandleFunc("/LoginStatus", controllers.CheckLogin)
	err := http.ListenAndServe(":8080", middleware.EnableCORS(mux))

	if err != nil {
		log.Fatal("ListenAndServeError:", err)
	}
}
