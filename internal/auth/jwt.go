package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	secret []byte
}

func NewJWT(secret string) *JWT {
	return &JWT{secret: []byte(secret)}
}

func (j *JWT) Sign(userID uint64) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(j.secret)
}

func (j *JWT) Verify(tokenStr string) (uint64, error) {
	t, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil || !t.Valid {
		return 0, errors.New("invalid token")
	}

	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid claims")
	}

	sub, ok := claims["sub"]
	if !ok {
		return 0, errors.New("missing sub")
	}

	// jwt MapClaims numbers are float64
	idf, ok := sub.(float64)
	if !ok {
		return 0, errors.New("invalid sub type")
	}
	return uint64(idf), nil
}
