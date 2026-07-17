package utils

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"slate-backend/pkg/config"
	"slate-backend/pkg/types"
	"time"

	"github.com/golang-jwt/jwt"
)

func GenerateJWT(userClaim *types.JWTClaim, TTL int64, cfg *config.Config) (string, error) {
	key := cfg.JWTSecret
	now := time.Now()

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       userClaim.ID,
		"username": userClaim.GithubUsername,
		"iat":      now.Unix(),
		"exp":      now.Add(time.Duration(TTL) * time.Second).Unix(),
		"iss":      "slate",
	})

	token, err := t.SignedString([]byte(key))
	if err != nil {
		return "", err
	}
	return token, nil
}

func ParseJWT(tokenString string, cfg *config.Config) (*types.JWTClaim, error) {
	key := cfg.JWTSecret
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is correct
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(key), nil
	})
	if err != nil {
		return nil, err
	}
	if token == nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	idFloat, _ := claims["id"].(float64)
	username, _ := claims["username"].(string)

	if idFloat == 0 || username == "" {
		return nil, errors.New("missing required claims")
	}

	return &types.JWTClaim{
		ID:             int64(idFloat),
		GithubUsername: username,
	}, nil
}

func GenerateGithubJWT(config *config.Config) (string, error) {
	pemBlock, _ := pem.Decode(config.GithubPrivateKey)
	if pemBlock == nil {
		return "", fmt.Errorf("Unable to parse private key")
	}

	key, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return "", err
	}

	now := time.Now()

	t := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(10) * time.Minute).Unix(),
		"iss": config.GithubAppID,
	})

	token, err := t.SignedString(key)
	if err != nil {
		return "", err
	}
	return token, nil
}
