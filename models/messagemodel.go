package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Message struct {
	ID         bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Msg        string        `json:"msg" bson:"msg"`
	SenderId   string        `json:"sender_id" bson:"sender_id"`
	ReceiverId string        `json:"receiver_id" bson:"receiver_id"`
	Timestamp  time.Time     `json:"timestamp" bson:"timestamp"`
}
