package models

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct { //struct for active users
	UserID string
	Conn   *websocket.Conn
	Send   chan Message
}

func (c *Client) ReadPump(h *Hub) {
	defer func() { // unregister client on hub and close websocket connection at the end
		h.Unregister <- c
		c.Conn.Close()
	}() // immediately invoked lambda function
	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		log.Println(msg)
		if err != nil {
			// break connection on unexpected error closely cleanly due to defer
			break
		}
		msg.SenderId = c.UserID
		msg.Timestamp = time.Now()
		h.Broadcast <- msg
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()
	for msg := range c.Send {
		err := c.Conn.WriteJSON(msg)
		if err != nil {
			return
		}
	}
}
