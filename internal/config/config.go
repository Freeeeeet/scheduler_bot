package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string `mapstructure:"TELEGRAM_TOKEN"`
	DBDSN         string `mapstructure:"DB_DSN"`
	Environment   string `mapstructure:"ENV"`
}

func Load() (*Config, error) {
	// Пытаемся загрузить .env файл (игнорируем ошибку, если файла нет)
	if err := godotenv.Load(".env"); err != nil {
		log.Println("⚠️  No .env file found, using environment variables")
	} else {
		log.Println("✅ Loaded configuration from .env file")
	}

	// Читаем напрямую из переменных окружения (после godotenv.Load они там)
	cfg := &Config{
		DBDSN:         os.Getenv("DB_DSN"),
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		Environment:   os.Getenv("ENV"),
	}

	// Устанавливаем дефолтные значения
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	// Проверяем обязательные поля
	if cfg.DBDSN == "" {
		return nil, fmt.Errorf("DB_DSN is required but not set")
	}

	// Дебаг: показываем что загружено (без пароля)
	log.Printf("Config loaded\n")

	return cfg, nil
}

func (c *Config) GetDBDSN() string {
	return c.DBDSN
}
