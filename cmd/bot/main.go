package main

import (
	"log"

	"github.com/Freeeeeet/scheduler_bot/internal/app"
	"github.com/Freeeeeet/scheduler_bot/internal/config"
)

func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := app.NewLogger(cfg.Environment)

	defer logger.Sync()

	logger.Sugar().Infow("Starting scheduler bot",
		"environment", cfg.Environment,
		"token_length", len(cfg.TelegramToken))
}
