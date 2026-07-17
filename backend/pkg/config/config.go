package config

import (
	"log"
	"os"
)

type Config struct {
	Environment         string
	DatabaseURL         string
	GithubClientID      string
	GithubClientSecret  string
	GithubAppSlug       string
	GithubAppID         string
	GithubPrivateKey    []byte
	GithubWebhookSecret string
	EncryptionKey       string
	JWTSecret           string
}

func LoadConfig() *Config {

	// Load Github Private Key
	pemFilePath := getRequiredEnv("GITHUB_PRIVATE_KEY_PATH")
	privateKey := loadPEMFile(pemFilePath)

	cfg := &Config{
		Environment:         getRequiredEnv("APP_ENV"),
		DatabaseURL:         getRequiredEnv("DATABASE_URL"),
		GithubClientID:      getRequiredEnv("GITHUB_CLIENT_ID"),
		GithubClientSecret:  getRequiredEnv("GITHUB_CLIENT_SECRET"),
		GithubAppSlug:       getRequiredEnv("GITHUB_APP_SLUG"),
		GithubAppID:         getRequiredEnv("GITHUB_APP_ID"),
		GithubWebhookSecret: getRequiredEnv("GITHUB_WEBHOOK_SECRET"),
		GithubPrivateKey:    privateKey,
		EncryptionKey:       getRequiredEnv("ENCRYPTION_KEY"),
		JWTSecret:           getRequiredEnv("JWT_SECRET"),
	}

	return cfg
}

func loadPEMFile(path string) []byte {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("FATAL CONFIG ERROR: PEM File at path [%s] is required but missing.", path)
	}

	return content
}

func getRequiredEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		log.Fatalf("FATAL CONFIG ERROR: Environment variable [%s] is required but missing.", key)
	}
	return value
}
