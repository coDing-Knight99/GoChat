package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("Go-Chat")

type Claims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

func GenerateToken	(userID string) (string,error){
	expirationTime := time.Now().Add(24*time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenStr string) (string,error) {
	claims := &Claims{}
	token,err := jwt.ParseWithClaims(tokenStr,claims,func(token *jwt.Token) (interface{},error){
		if _,ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil,errors.New("unexpected signing method")
		}
		return jwtSecret,nil
	})
	if err!=nil {
		return "",err
	}

	if !token.Valid {
		return "",errors.New("invalid token")
	}
	return claims.UserID,nil
}