package main

import (
	"log"

	"github.com/LiteyukiStudio/devops/internal/api"
	"github.com/LiteyukiStudio/devops/internal/config"
	"github.com/LiteyukiStudio/devops/internal/database"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	router := api.NewRouter(db)

	log.Printf("api listening on %s", cfg.APIAddr)
	if err := router.Run(cfg.APIAddr); err != nil {
		log.Fatalf("run api: %v", err)
	}
}
