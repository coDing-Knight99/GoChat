package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/coDing-Knight99/GoChat/auth"
)

type contextKey string
const UserIDKey contextKey = "userID"
func VerifyToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string
		if cookie, err := r.Cookie("accessToken"); err == nil {
			tokenStr = cookie.Value
		}
		if tokenStr == "" {
			log.Println("Unauthorized Access Token missing")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"message":"Unauthorized",
			})
			// http.Error(w, "Unauthorized Access Token missing", http.StatusUnauthorized)
			return
		}
		userID, err := auth.ValidateAccessToken(tokenStr)

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"message":"Unauthorized",
			})
			// http.Error(w, "Unauthorized Access Token Expired or Invalid", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
