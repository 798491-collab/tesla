package auth

import (
	"errors"
	"fmt"
	"tesla-server/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Type     string `json:"type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
}

var jwtSecret []byte

func Init(cfg *config.JWTConfig) {
	jwtSecret = []byte(cfg.Secret)
}

func GenerateToken(userID uint64, username string, expiresIn int) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Type:     "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresIn) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tesla-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func GenerateTokenPair(userID uint64, username string, accessExpiresIn, refreshExpiresIn int) (*TokenPair, error) {
	now := time.Now()

	accessClaims := Claims{
		UserID:   userID,
		Username: username,
		Type:     "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(accessExpiresIn) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "tesla-server",
		},
	}

	refreshClaims := Claims{
		UserID:   userID,
		Username: username,
		Type:     "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(refreshExpiresIn) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "tesla-server",
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

	accessTokenString, err := accessToken.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshTokenString, err := refreshToken.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    accessExpiresIn,
		ExpiresAt:    now.Add(time.Duration(accessExpiresIn) * time.Second).Unix(),
	}, nil
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
