package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	APIAddr     string
	DatabaseURL string
	RedisAddr   string
}

func Load() Config {
	loadEnvFile()

	return Config{
		APIAddr:     env("API_ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://devops:devops@localhost:5432/devops?sslmode=disable"),
		RedisAddr:   env("REDIS_ADDR", "localhost:6379"),
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
