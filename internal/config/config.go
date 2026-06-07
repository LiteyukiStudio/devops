package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	APIAddr                     string
	DatabaseURL                 string
	RedisAddr                   string
	BuilderSharedToken          string
	BuilderTaskLeaseSeconds     int64
	BuilderPollIntervalSeconds  int64
	BuilderExecutorImage        string
	BuilderExecutor             string
	BuilderMaxConcurrency       int
	BuilderAPIURL               string
	BuilderAgentID              string
	BuilderAgentName            string
	BuilderWorkspaceRoot        string
	BuilderWorkspaceHostRoot    string
	BuilderNPMRegistry          string
	BuildPrivateEgressCIDRs     []string
	BuildBlockedEgressCIDRs     []string
	DeployRolloutTimeoutSeconds int64
	CertManagerClusterIssuer    string
}

func Load() Config {
	loadEnvFile()

	return Config{
		APIAddr:                     env("API_ADDR", ":8080"),
		DatabaseURL:                 env("DATABASE_URL", "postgres://devops:devops@localhost:5432/devops?sslmode=disable"),
		RedisAddr:                   env("REDIS_ADDR", "localhost:6379"),
		BuilderSharedToken:          env("BUILDER_SHARED_TOKEN", env("BUILDER_TOKEN", "dev-builder-token")),
		BuilderTaskLeaseSeconds:     int64(envInt("BUILDER_TASK_LEASE_SECONDS", 300)),
		BuilderPollIntervalSeconds:  int64(envInt("BUILDER_POLL_INTERVAL_SECONDS", 5)),
		BuilderExecutorImage:        env("BUILDER_EXECUTOR_IMAGE", "moby/buildkit:v0.24.0-rootless"),
		BuilderExecutor:             env("BUILDER_EXECUTOR", "docker"),
		BuilderMaxConcurrency:       envInt("BUILDER_MAX_CONCURRENCY", 1),
		BuilderAPIURL:               env("LITEYUKI_API_URL", env("BUILDER_API_URL", "http://localhost:8080")),
		BuilderAgentID:              env("BUILDER_AGENT_ID", ""),
		BuilderAgentName:            env("BUILDER_AGENT_NAME", "local-builder"),
		BuilderWorkspaceRoot:        env("BUILDER_WORKSPACE_ROOT", ""),
		BuilderWorkspaceHostRoot:    env("BUILDER_WORKSPACE_HOST_ROOT", ""),
		BuilderNPMRegistry:          env("BUILDER_NPM_REGISTRY", ""),
		BuildPrivateEgressCIDRs:     envList("BUILD_PRIVATE_EGRESS_CIDRS"),
		BuildBlockedEgressCIDRs:     append(defaultBuildBlockedEgressCIDRs(), envList("BUILD_BLOCKED_EGRESS_CIDRS")...),
		DeployRolloutTimeoutSeconds: int64(envInt("DEPLOY_ROLLOUT_TIMEOUT_SECONDS", 600)),
		CertManagerClusterIssuer:    env("CERT_MANAGER_CLUSTER_ISSUER", "letsencrypt-http01"),
	}
}

func RuntimeMode() string {
	switch strings.ToLower(os.Getenv("APP_ENV")) {
	case "production", "prod":
		return "production"
	case "development", "dev", "local":
		return "development"
	}

	if strings.Contains(os.Args[0], "go-build") {
		return "development"
	}
	return "production"
}

func loadEnvFile() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		if RuntimeMode() != "development" {
			return
		}
		envFile = ".env.dev"
	}

	if err := godotenv.Load(envFile); err != nil {
		if RuntimeMode() == "development" {
			log.Printf("development mode: env file %s not loaded: %v; using process environment", envFile, err)
		}
		return
	}

	if RuntimeMode() == "development" {
		log.Printf("development mode: loaded env file %s", envFile)
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func defaultBuildBlockedEgressCIDRs() []string {
	return []string{"169.254.169.254/32"}
}
