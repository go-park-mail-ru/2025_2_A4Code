package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppConfig *AppConfig
}

type AppConfig struct {
	ConfigPath string
}

func GetConfig() Config {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Error loading .env file")
	}
	return Config{
		AppConfig: GetAppConfig(),
	}
}

func GetAppConfig() *AppConfig {
	return &AppConfig{
		ConfigPath: os.Getenv("CONFIG_PATH"),
	}
}
