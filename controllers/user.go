package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/coDing-Knight99/GoChat/auth"
	"github.com/coDing-Knight99/GoChat/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)



func RegisterUser(userCollection *mongo.Collection, w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var alreadyTaken models.User
	er1 := userCollection.FindOne(context.Background(), bson.M{"userID": user.UserID}).Decode(&alreadyTaken)
	if er1 == nil {
		http.Error(w, "Username is already taken", http.StatusConflict)
		return
	}
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		log.Println("Error hashing password")
		http.Error(w, "Error Hashing Password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hash)
	_, er := userCollection.InsertOne(context.Background(), user)
	if er != nil {
		http.Error(w, "Error registering User", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User registered successfully",
	})
}

type loginCreds struct {
	UserID  string `json:"username" bson:"userID"`
	PassKey string `json:"password" bson:"password"`
}

func Login(userCollection *mongo.Collection, w http.ResponseWriter, r *http.Request) {
	logCred := &loginCreds{}
	err := json.NewDecoder(r.Body).Decode(logCred)
	if err != nil {
		log.Println("Invalid Request Body")
		http.Error(w, "Invalid Request Body", http.StatusBadRequest)
		return
	}
	var user models.User
	err = userCollection.FindOne(context.Background(), bson.M{"userID": logCred.UserID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	er := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(logCred.PassKey),
	)

	if er != nil {
		log.Println("Incorrect Password")
		http.Error(w, "Incorrect Password", http.StatusForbidden)
		return
	}

	var tokens *auth.Tokens
	tokens, err = auth.GenerateTokens(logCred.UserID)
	if err != nil {
		log.Println("Error Generating Tokens")
		http.Error(w, "Error generating Tokens", http.StatusInternalServerError)
		return
	}

	var sameSite http.SameSite
	if os.Getenv("Environment") == "Development" {
		sameSite = http.SameSiteLaxMode
	} else {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    tokens.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("Environment") == "Production",
		SameSite: sameSite,
		MaxAge:   60 * 15,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    tokens.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("Environment") == "Production",
		SameSite: sameSite,
		MaxAge:   60 * 60 * 24,
	})
	log.Println("Login Successful")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login Successful",
	})
}
