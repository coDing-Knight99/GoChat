package models

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
)

type Hub struct {
	Clients    map[string]*Client
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	RDB        *redis.Client
}

func NewHub(rdb *redis.Client	) *Hub {
	// rdb := redis.NewClient(&redis.Options{
	// 	Addr:     "localhost:6379",
	// 	Password: "",
	// 	DB:       0,
	// })
	return &Hub{
		Clients:    make(map[string]*Client),
		Broadcast:  make(chan Message),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		RDB:        rdb,
	}
}



func (h *Hub) StartHub() {
	ctx:=context.Background()
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.UserID] = client
			log.Println("Client Registered",h.Clients[client.UserID])
		case client := <-h.Unregister:
			if _, ok := h.Clients[client.UserID]; ok {
				delete(h.Clients, client.UserID)
				close(client.Send)
			}
		case msg := <-h.Broadcast:
			receiverClient, Online := h.Clients[msg.ReceiverId]
			if Online {
				select {
				case receiverClient.Send <- msg:
					log.Println("Message Delivered to Client",msg.ReceiverId)
				default:
					close(receiverClient.Send)
					delete(h.Clients, msg.ReceiverId)
				}
			}
			go func (m Message){
				msgBytes, err:=json.Marshal(m)
				if err!=nil {
				log.Println("Redis Marshal Error:",err)
				return
				}
				err = h.RDB.LPush(ctx,"chat_msgs",msgBytes).Err()
				if err != nil {
					log.Println("Failed to push to Redis:",err)
				}
			}(msg)
		}
	}
}
