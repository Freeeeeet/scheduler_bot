package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	TelegramToken string `mapstructure:"TELEGRAM_TOKEN"`
	DBDSN         string `mapstructure:"DB_DSN"`
	Environment   string `mapstructure:"ENV"`
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("ENV", "development")

	viper.AutomaticEnv()

	var cfg Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
