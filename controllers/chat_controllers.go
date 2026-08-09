package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coDing-Knight99/GoChat/middleware"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ConversationSummary struct {
	ContactID   string    `bson:"_id" json:"contact_id"`
	LastMessage string    `bson:"last_message" json:"last_message"`
	LastTime    time.Time `bson:"last_time" json:"last_time"`
	SenderID    string    `bson:"sender_id" json:"sender_id"`
}

func GetConversations(collection *mongo.Collection, w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	// Extract query parameter using net/http
	currentUser := r.Context().Value(middleware.UserIDKey).(string)
	if currentUser == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "user query parameter is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		// 1. Match messages where currentUser is either sender or receiver
		bson.D{{Key: "$match", Value: bson.M{
			"$or": []bson.M{
				{"sender_id": currentUser},
				{"receiver_id": currentUser},
			},
		}}},

		// 2. Sort by newest messages first
		bson.D{{Key: "$sort", Value: bson.D{{Key: "timestamp", Value: -1}}}},

		// 3. Group by contact (the other party)
		bson.D{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"$cond": bson.M{
					"if":   bson.M{"$eq": []interface{}{"$sender_id", currentUser}},
					"then": "$receiver_id",
					"else": "$sender_id",
				},
			},
			"last_message": bson.M{"$first": "$msg"},
			"last_time":    bson.M{"$first": "$timestamp"},
			"sender_id":    bson.M{"$first": "$sender_id"},
		}}},

		// 4. Sort contacts by their most recent activity
		bson.D{{Key: "$sort", Value: bson.D{{Key: "last_time", Value: -1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to fetch conversations"})
		return
	}
	defer cursor.Close(ctx)

	var conversations []ConversationSummary
	if err = cursor.All(ctx, &conversations); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to parse conversations"})
		return
	}

	if conversations == nil {
		conversations = []ConversationSummary{}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(conversations)
}
