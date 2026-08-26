package models

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/genai"
)

type Hub struct {
	Clients    map[string]*Client
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	RDB        *redis.Client
}

func NewHub(rdb *redis.Client) *Hub {
	// rdb := redis.NewClient(&redis.Options{
	// 	Addr:     "localhost:6379",
	// 	Password: "",
	// 	DB:       0,
	// })
	return &Hub{
		Clients:    make(map[string]*Client),
		Broadcast:  make(chan Message),
		Register:   make(chan *Client),
		Unregister: make(chan *Client), RDB: rdb,
	}
}

func (h *Hub) APICall(c string, m Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		log.Printf("Failed to create Gemini client: %v", err)
		return
	}
	log.Println(m)
	stream, err := client.Models.GenerateContent(ctx, "gemini-3.5-flash-lite", genai.Text(m.Msg+"Answer in only text format"), nil)
	if err != nil {
		log.Println(err)
		return
	}
	resp := Message{
		ID:         bson.NewObjectID(),
		Msg:        stream.Text(),
		SenderId:   "GoIntellisense",
		ReceiverId: c,
		Timestamp:  time.Now(),
	}
	log.Println(resp)
	h.Broadcast <- resp
}

func (h *Hub) SaveToDB(m Message) {
	ctx := context.Background()
	msgBytes, err := json.Marshal(m)
	if err != nil {
		log.Println("Redis Marshal Error:", err)
		return
	}
	err = h.RDB.LPush(ctx, "chat_msgs", msgBytes).Err()
	if err != nil {
		log.Println("Failed to push to Redis:", err)
	}
}
func (h *Hub) StartHub() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.UserID] = client
			log.Println("Client Registered", h.Clients[client.UserID])
		case client := <-h.Unregister:
			if _, ok := h.Clients[client.UserID]; ok {
				delete(h.Clients, client.UserID)
				close(client.Send)
			}
		case msg := <-h.Broadcast:
			if msg.ReceiverId == "GoIntellisense" {
				go h.APICall(msg.SenderId, msg)
				go h.SaveToDB(msg)
				continue
			}
			receiverClient, Online := h.Clients[msg.ReceiverId]
			if Online {
				select {
				case receiverClient.Send <- msg:
					log.Println("Message Delivered to Client", msg.ReceiverId)
				default:
					close(receiverClient.Send)
					delete(h.Clients, msg.ReceiverId)
				}
			}
			go h.SaveToDB(msg)
		}
	}
}
