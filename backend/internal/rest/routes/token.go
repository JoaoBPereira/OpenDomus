package routes

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Auth creates and verifies the HS256 JWTs of the API: { username, role, iat, exp }
type Auth struct {
	secret []byte
}

func NewAuth(secret string) *Auth {
	return &Auth{secret: []byte(secret)}
}

// TokenPayload is what a token carries about the user
type TokenPayload struct {
	Username string
	Role     string
}

func (auth *Auth) CreateToken(payload TokenPayload, validFor time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"username": payload.Username,
		"role":     payload.Role,
		"iat":      now.Unix(),
		"exp":      now.Add(validFor).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(auth.secret)
}

func (auth *Auth) VerifyToken(token string) (*TokenPayload, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return auth.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	username, _ := claims["username"].(string)
	role, _ := claims["role"].(string)
	return &TokenPayload{Username: username, Role: role}, nil
}
