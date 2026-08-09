package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var accessTokenSecret = []byte(os.Getenv("AccessTokenSecret"))
var refreshTokenSecret = []byte(os.Getenv("refreshTokenSecret"))

type Claims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func GenerateTokens(userID string) (*Tokens, error) {
	now := time.Now()

	accessClaims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessSigned, err := accessToken.SignedString(accessTokenSecret)
	if err != nil {
		return nil, err
	}

	refreshClaims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshSigned, err := refreshToken.SignedString(refreshTokenSecret)
	if err != nil {
		return nil, err
	}

	return &Tokens{
		AccessToken:  accessSigned,
		RefreshToken: refreshSigned,
	}, nil
}

func parseToken(tokenStr string, secret []byte) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid or expired Token")
	}
	return claims.UserID, nil
}

func ValidateAccessToken(tokenStr string) (string, error) {
	return parseToken(tokenStr, accessTokenSecret)
}

func ValidateRefreshToken(tokenStr string) (string, error) {
	return parseToken(tokenStr, refreshTokenSecret)
}
