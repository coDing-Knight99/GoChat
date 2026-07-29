package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID       bson.ObjectID `json:"Id" bson:"_id,omitempty"`
	UserID   string        `json:"username" bson:"userID"`
	Email    string        `json:"email" bson:"email"`
	Password string        `json:"password" bson:"password"`
}
