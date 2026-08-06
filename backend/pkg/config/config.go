package config

import (
	"log"
	"os"
)

type Config struct {
	Environment    string
	DatabaseURL    string
	AppURL         string
	SiteBaseDomain string
	EncryptionKey  string
	JWTSecret      string

	GithubClientID      string
	GithubClientSecret  string
	GithubAppSlug       string
	GithubAppID         string
	GithubPrivateKey    []byte
	GithubWebhookSecret string

	RedisAddr     string
	RedisPassword string

	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string

	DockerSocketPath string
	BuildImageBase   string
	BuildTimeout     int
}

func LoadConfig() *Config {

	// Load Github Private Key
	pemFilePath := getRequiredEnv("GITHUB_PRIVATE_KEY_PATH")
	privateKey := loadPEMFile(pemFilePath)

	cfg := &Config{
		Environment:    getRequiredEnv("APP_ENV"),
		DatabaseURL:    getRequiredEnv("DATABASE_URL"),
		AppURL:         getEnvWithDefault("APP_URL", "http://localhost:5173"),
		EncryptionKey:  getRequiredEnv("ENCRYPTION_KEY"),
		JWTSecret:      getRequiredEnv("JWT_SECRET"),
		SiteBaseDomain: getEnvWithDefault("SITE_BASE_DOMAIN", "slate.sakkshm.me"),

		GithubClientID:      getRequiredEnv("GITHUB_CLIENT_ID"),
		GithubClientSecret:  getRequiredEnv("GITHUB_CLIENT_SECRET"),
		GithubAppSlug:       getRequiredEnv("GITHUB_APP_SLUG"),
		GithubAppID:         getRequiredEnv("GITHUB_APP_ID"),
		GithubWebhookSecret: getRequiredEnv("GITHUB_WEBHOOK_SECRET"),
		GithubPrivateKey:    privateKey,

		RedisAddr:     getEnvWithDefault("REDIS_ADDR", "redis:6379"),
		RedisPassword: getEnvWithDefault("REDIS_PASSWORD", ""),

		MinIOEndpoint:  getEnvWithDefault("MINIO_ENDPOINT", "minio:9000"),
		MinIOAccessKey: getRequiredEnv("MINIO_ACCESS_KEY"),
		MinIOSecretKey: getRequiredEnv("MINIO_SECRET_KEY"),
		MinIOBucket:    getEnvWithDefault("MINIO_BUCKET", "slate-assets"),

		DockerSocketPath: getEnvWithDefault("DOCKER_SOCKET", "/var/run/docker.sock"),
		BuildImageBase:   getEnvWithDefault("BUILD_IMAGE", "slate-base-runner:latest"),
		BuildTimeout:     300,
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

func getEnvWithDefault(key string, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}
	return value
}
