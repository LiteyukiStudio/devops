package main

import (
	"log"

	"github.com/LiteyukiStudio/devops/internal/config"
	"github.com/LiteyukiStudio/devops/internal/worker"
)

func main() {
	cfg := config.Load()

	if cfg.RedisAddr == "" {
		log.Println("worker idle: REDIS_ADDR is empty")
		select {}
	}

	if err := worker.Run(cfg.RedisAddr); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
