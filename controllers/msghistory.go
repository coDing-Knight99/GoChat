package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/coDing-Knight99/GoChat/middleware"
	"go.mongodb.org/mongo-driver/v2/bson"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Document model for MongoDB mapping
type MessageDoc struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	SenderID   string        `bson:"sender_id" json:"sender_id"`
	ReceiverID string        `bson:"receiver_id" json:"receiver_id"`
	Content    string        `bson:"msg" json:"msg"`
	CreatedAt  time.Time     `bson:"timestamp" json:"timestamp"`
}

type PaginationMeta struct {
	CurrentPage int  `json:"current_page"`
	Limit       int  `json:"limit"`
	HasMore     bool `json:"has_more"`
}

type GetMessagesResponse struct {
	Messages   []MessageDoc   `json:"messages"`
	Pagination PaginationMeta `json:"pagination"`
}

func HandleGetMessagesMongo(collection *mongo.Collection, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Parse & Validate Query Parameters
	query := r.URL.Query()
	user := r.Context().Value(middleware.UserIDKey).(string)
	contact := query.Get("contact")

	if user == "" || contact == "" {
		http.Error(w, `{"error":"'user' and 'contact' query parameters are required"}`, http.StatusBadRequest)
		return
	}

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}

	limit := 50

	skip := int64((page - 1) * 50)

	// 2. Build MongoDB Match Filter
	// Match messages where (sender == user AND receiver == contact) OR (sender == contact AND receiver == user)
	filter := bson.M{
		"$or": []bson.M{
			{"sender_id": user, "receiver_id": contact},
			{"sender_id": contact, "receiver_id": user},
		},
	}

	// 3. Set Find Options (Sort Newest First, Fetch limit + 1 to check for hasMore)
	findOpts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit + 1))

	cursor, err := collection.Find(r.Context(), filter, findOpts)
	if err != nil {
		http.Error(w, `{"error":"failed to retrieve message history"}`, http.StatusInternalServerError)
		return
	}
	defer cursor.Close(r.Context())

	var fetched []MessageDoc
	if err := cursor.All(r.Context(), &fetched); err != nil {
		log.Printf("failed to decode message history for user %q and contact %q: %v", user, contact, err)
		http.Error(w, `{"error":"failed to decode message documents"}`, http.StatusInternalServerError)
		return
	}

	// 4. Determine 'has_more' state
	hasMore := false
	if len(fetched) > limit {
		hasMore = true
		fetched = fetched[:limit] // Trim back to requested limit size
	}

	// 5. Reverse array to return chronological order (oldest -> newest) for UI rendering
	for i, j := 0, len(fetched)-1; i < j; i, j = i+1, j-1 {
		fetched[i], fetched[j] = fetched[j], fetched[i]
	}

	// Ensure non-null JSON array if empty
	if fetched == nil {
		fetched = []MessageDoc{}
	}

	// 6. Send Json Response
	resp := GetMessagesResponse{
		Messages: fetched,
		Pagination: PaginationMeta{
			CurrentPage: page,
			Limit:       limit,
			HasMore:     hasMore,
		},
	}

	json.NewEncoder(w).Encode(resp)
}
