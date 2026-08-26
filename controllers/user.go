package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/coDing-Knight99/GoChat/auth"
	"github.com/coDing-Knight99/GoChat/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

func CheckLogin(w http.ResponseWriter,r *http.Request){
	w.Header().Set("Content-Type","application/json")
	cookie,err := r.Cookie("accessToken")
	var accessToken string
	if err==nil {
		accessToken = cookie.Value
	}
	if accessToken=="" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{
			"isLogin":false,
		})
		return
	} 
	_, err1 := auth.ValidateAccessToken(accessToken)

	if err1!=nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{
			"isLogin":false,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{
		"isLogin":true,
	})
}

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
		MaxAge:   60 * 60,
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
	Username string `json:"username" bson:"userID"`
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

type searchusername struct {
	Id       bson.ObjectID `json:"ID" bson:"_id"`
	Username string        `json:"username" bson:"userID"`
}

func SearchUser(userCollection *mongo.Collection, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var q queryusername
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		log.Println("Invalid Json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid Json",
		})
		return
	}
	safePrefix := regexp.QuoteMeta(q.Username)

	filter := bson.M{
		"userID": bson.Regex{
			Pattern: "^" + safePrefix,
			Options: "i", // case insensitive
		},
	}

	findOpts := options.Find().SetLimit(10).SetProjection(bson.M{"_id": 1, "userID": 1})

	cursor, err := userCollection.Find(r.Context(), filter, findOpts)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal Server Error",
		})
		return
	}

	defer cursor.Close(r.Context())

	var users []searchusername
	if err := cursor.All(r.Context(), &users); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal Server Error",
		})
		return
	}
	log.Println(users)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string][]searchusername{
		"users": users,
	})
}

func LogoutUser(w http.ResponseWriter,r *http.Request){
	var sameSite http.SameSite
	if os.Getenv("Environment") == "Development" {
		sameSite = http.SameSiteLaxMode
	} else {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("Environment") == "Production",
		SameSite: sameSite,
		MaxAge:  -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("Environment") == "Production",
		SameSite: sameSite,
		MaxAge:   60 * 60 * 24,
	})
	w.WriteHeader(http.StatusOK)
}
