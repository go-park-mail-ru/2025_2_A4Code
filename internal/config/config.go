package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	AppConfig *AppConfig
	DBConfig  *DBConfig
}

type AppConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`
}

func GetConfig() Config {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Message loading .env file")
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/prod.yml" // значение по умолчанию
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Message reading config file %s: %v", configPath, err)
	}

	var yamlStruct struct {
		App AppConfig `yaml:"app"`
		DB  DBConfig  `yaml:"db"`
	}

	if err := yaml.Unmarshal(data, &yamlStruct); err != nil {
		log.Fatalf("Message parsing YAML config: %v", err)
	}

	return Config{
		AppConfig: &yamlStruct.App,
		DBConfig:  &yamlStruct.DB,
	}
	// return Config{
	// 	AppConfig: GetAppConfig(),
	// }
}

// func GetAppConfig() *AppConfig {
// 	return &AppConfig{
// 		ConfigPath: os.Getenv("CONFIG_PATH"),
// 	}
// }
