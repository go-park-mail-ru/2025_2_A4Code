package config

import (
	e "2025_2_a4code/internal/lib/wrapper"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	AppConfig   *AppConfig
	DBConfig    *DBConfig
	MinioConfig *MinioConfig
}

type AppConfig struct {
	Host                string `yaml:"host"`
	AuthPort            string `yaml:"auth_port"`
	MessagesPort        string `yaml:"messages_port"`
	ProfilePort         string `yaml:"profile_port"`
	FilePort            string `yaml:"file_port"`
	GatewayPort         string `yaml:"gateway_port"`
	GatewayMetricsPort  string `yaml:"gateway_metrics_port"`
	FileMetricsPort     string `yaml:"file_metrics_port"`
	AuthMetricsPort     string `yaml:"auth_metrics_port"`
	MessagesMetricsPort string `yaml:"messages_metrics_port"`
	ProfileMetricsPort  string `yaml:"profile_metrics_port"`
	Secret              string `yaml:"secret"`
}

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`
}

type MinioConfig struct {
	Host            string `yaml:"host"`
	Port            string `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	BucketName      string `yaml:"bucket_name"`
	FilesBucketName string `yaml:"files_bucket_name"`
	Endpoint        string `yaml:"endpoint"`
	UseSSL          bool   `yaml:"use_ssl"`
	PublicEndpoint  string `yaml:"public_endpoint"`
	PublicUseSSL    bool   `yaml:"public_use_ssl"`
}

func GetConfig() (Config, error) {
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
		return Config{}, e.Wrap("Can not read config file: ", err)
	}

	var yamlStruct struct {
		App   AppConfig   `yaml:"app"`
		DB    DBConfig    `yaml:"db"`
		Minio MinioConfig `yaml:"minio"`
	}

	if err := yaml.Unmarshal(data, &yamlStruct); err != nil {
		return Config{}, e.Wrap("Can not parse config file: ", err)
	}

	return Config{
		AppConfig:   &yamlStruct.App,
		DBConfig:    &yamlStruct.DB,
		MinioConfig: &yamlStruct.Minio,
	}, nil
}
