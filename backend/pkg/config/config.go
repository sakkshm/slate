package config

import (
	"log"
	"os"
)

type Config struct {
	Environment         string
	GithubClientID      string
	GithubClientSecret  string
	GithubAppSlug       string
	GithubAppID         string
	GithubPrivateKey    string
	GithubWebhookSecret string
}

func LoadConfig() *Config {

	cfg := &Config{
		Environment:         getRequiredEnv("APP_ENV"),
		GithubClientID:      getRequiredEnv("GITHUB_CLIENT_ID"),
		GithubClientSecret:  getRequiredEnv("GITHUB_CLIENT_SECRET"),
		GithubAppSlug:       getRequiredEnv("GITHUB_APP_SLUG"),
		GithubAppID:         getRequiredEnv("GITHUB_APP_ID"),
		GithubPrivateKey:    getRequiredEnv("GITHUB_PRIVATE_KEY"),
		GithubWebhookSecret: getRequiredEnv("GITHUB_WEBHOOK_SECRET"),
	}

	return cfg
}

func getRequiredEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		log.Fatalf("FATAL CONFIG ERROR: Environment variable [%s] is required but missing.", key)
	}
	return value
}
