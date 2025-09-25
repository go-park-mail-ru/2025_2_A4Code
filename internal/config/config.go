package config

// Будет использоваться дальше для парсинга конфига
type Config struct {
	Env        string `yaml:"env" env-default:"local"`
	HTTPServer `yaml:"http_server"`
}
type HTTPServer struct {
	Address string `yaml:"address" env-default:"localhost:8080"`
}

func MustLoad() *Config {
	var cfg Config
	// Логика...
	return &cfg
}
