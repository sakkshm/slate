package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Environment    string
	DatabaseURL    string
	AppURL         string
	SiteBaseDomain string
	SiteScheme     string
	SitePort       string
	ReservedHosts  []string
	EncryptionKey  string
	JWTSecret      string

	CORSAllowedOrigins []string
	TrustedProxyIPs    []string

	RateGlobalRps  int
	RateAuthRpm    int
	RateWebhookRpm int
	RateBuildRph   int

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

	ArtifactRetentionDays int
	PruneIntervalHours    int
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
		SiteScheme:     getEnvWithDefault("SITE_SCHEME", "http"),
		SitePort:       getEnvWithDefault("SITE_PORT", ""),
		ReservedHosts:  getCSVEnv("SITE_RESERVED_HOSTS", "api,www,app,dashboard,admin,minio,docs,git,status"),

		CORSAllowedOrigins: getCSVEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
		TrustedProxyIPs:    getCSVEnv("TRUSTED_PROXY_IPS", ""),

		RateGlobalRps:  getEnvIntWithDefault("RATE_GLOBAL_RPS", 50),
		RateAuthRpm:    getEnvIntWithDefault("RATE_AUTH_RPM", 10),
		RateWebhookRpm: getEnvIntWithDefault("RATE_WEBHOOK_RPM", 100),
		RateBuildRph:   getEnvIntWithDefault("RATE_BUILD_RPH", 10),

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
		BuildTimeout:     getEnvIntWithDefault("BUILD_TIMEOUT", 300),

		ArtifactRetentionDays: getEnvIntWithDefault("ARTIFACT_RETENTION_DAYS", 30),
		PruneIntervalHours:    getEnvIntWithDefault("PRUNE_INTERVAL_HOURS", 1),
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

func getEnvIntWithDefault(key string, defaultValue int) int {
	value, err := strconv.Atoi(getEnvWithDefault(key, ""))
	if err != nil {
		return defaultValue
	}
	return value
}

func getCSVEnv(key string, defaultValue string) []string {
	value := getEnvWithDefault(key, defaultValue)
	var result []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}
