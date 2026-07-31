package controllers

import (
	"context"
	"encoding/json"
	"errors"
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
	w.Header().Set("Content-Type", "application/json")
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Bad Request",
		})
		return
	}
	var alreadyTaken models.User
	er1 := userCollection.FindOne(context.Background(), bson.M{"userID": user.UserID}).Decode(&alreadyTaken)
	if er1 == nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Username is already taken",
		})
		return
	}
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		log.Println("Error hashing password")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal Server Error",
		})
		return
	}
	user.Password = string(hash)
	_, er := userCollection.InsertOne(context.Background(), user)
	if er != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal Server Error",
		})
		return
	}

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
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Println("Invalid Request Body")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Bad Request",
		})
		return
	}
	var user models.User
	err = userCollection.FindOne(context.Background(), bson.M{"userID": logCred.UserID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "User Not Found",
		})
		return
	}
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal Server Error",
		})
		return
	}
	er := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(logCred.PassKey),
	)

	if er != nil {
		log.Println("Incorrect Password")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Incorrect Password",
		})
		return
	}

	var tokens *auth.Tokens
	tokens, err = auth.GenerateTokens(logCred.UserID)
	if err != nil {
		log.Println("Error Generating Tokens")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal Server Error",
		})
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
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login Successful",
	})
}

type queryusername struct {
	Username string `json:"username"`
}

func CheckUsernameAvailability(userCollection *mongo.Collection, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var qusername queryusername
	er := json.NewDecoder(r.Body).Decode(&qusername)
	log.Println("Json", qusername)
	if er != nil {
		log.Println("Invalid Json", qusername.Username)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Bad Request",
		})
		return
	}

	err := userCollection.FindOne(r.Context(), bson.M{"userID": qusername.Username}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		log.Println("Username available")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"available": "true",
		})
		return
	}

	if err != nil {
		log.Println("Database Error")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal Server Error",
		})
		return
	}

	log.Println("Username not available")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"available": "false",
	})
}
