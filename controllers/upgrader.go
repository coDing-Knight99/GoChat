package controllers

import (
	"log"
	"net/http"

	"github.com/coDing-Knight99/GoChat/middleware"
	"github.com/coDing-Knight99/GoChat/models"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleWebSocket(hub *models.Hub, w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	if userID == "" {
		log.Println("Unauthorized UserId missing in Context")
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade Error: ", err)
		return
	}

	client := &models.Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan models.Message, 256),
	}

	hub.Register <- client
	go client.WritePump()
	client.ReadPump(hub)
}
