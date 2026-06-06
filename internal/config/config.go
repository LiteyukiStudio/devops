package config

import (
	"log"
	"os"

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

func loadEnvFile() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		return
	}

	if err := godotenv.Load(envFile); err != nil {
		log.Printf("load env file %s: %v", envFile, err)
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
