// Пакет config отвечает за первичную настройку работы сервиса.
// В нём задаются настройки подключения к БД, файл для хранения данных, адрес и порт сервера приложения.
package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

const (
	serverAddress   = "localhost:8080"
	baseURL         = "http://" + serverAddress + "/"
	fileStoragePath = "shurldb.txt"
	databaseDSN     = "postgresql://shurl_user:qazxswedc@localhost:5432/shurl"
)

// Configuration содержит перечень настроек сервиса.
type Configuration struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`    // Адрес сервера приложения
	BaseURL         string `env:"BASE_URL"`          // Корневой URL работающего сервиса
	FileStoragePath string `env:"FILE_STORAGE_PATH"` // Путь к файлу для хранения данных сервиса
	DatabaseDSN     string `env:"DATABASE_DSN"`      // Строка для подключения к базе данных
	EnableHTTPS     bool   `env:"ENABLE_HTTPS"`
}

// NewConfiguration создаёт перечень настроек сервиса.
func NewConfiguration() *Configuration {
	cfg := new(Configuration)

	cfg.fillFromFlags()

	err := cfg.fillFromEnvironment()
	if err != nil {
		log.Println(err)
	}

	if cfg.ServerAddress == "" {
		cfg.ServerAddress = serverAddress
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = baseURL
	}

	baseURL := []rune(cfg.BaseURL)
	if baseURL[len(baseURL)-1] != '/' {
		cfg.BaseURL += "/"
	}

	log.Println("Resulting config:", cfg)

	return cfg
}

func (c *Configuration) fillFromFlags() {
	flag.StringVar(&c.ServerAddress, "a", serverAddress, "string with server address")
	flag.StringVar(&c.BaseURL, "b", baseURL, "string with base URL")
	flag.StringVar(&c.FileStoragePath, "f", fileStoragePath, "string with file storage path")
	flag.StringVar(&c.DatabaseDSN, "d", "", "string with database data source name")
	flag.BoolVar(&c.EnableHTTPS, "s", false, "flag to use HTTPS protocol instead of HTTP")

	flag.Parse()

	log.Println("Console flags:", c)
}

func (c *Configuration) fillFromEnvironment() error {
	err := env.Parse(c)
	if err != nil {
		return err
	}

	log.Println("Environment config:", c)

	return nil
}
