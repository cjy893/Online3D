package utils

import (
	"errors"
	"myapp/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte(config.Conf.JWTSecret))
}

func ParseToken(tokenString string) (uint, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Conf.JWTSecret), nil
	})
	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, jwt.ErrSignatureInvalid
	}

	sub, ok := claims["sub"].(float64)
	if !ok {
		return 0, errors.New("invalid subject type")
	}

	return uint(sub), nil
}
