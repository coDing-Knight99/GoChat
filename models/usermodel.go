package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `json:"Id" bson:"_id,omitempty"`
	Username  string             `json:"username" bson:"username,omitempty"`
	IsOnline  bool               `json:"is_online" bson:"is_online"` 
	CreatedAt time.Time        `json:"created_at" bson:"created_at"`
}
