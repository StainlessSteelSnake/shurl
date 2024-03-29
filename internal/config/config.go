// Пакет config отвечает за первичную настройку работы сервиса.
// В нём задаются настройки подключения к БД, файл для хранения данных, адрес и порт сервера приложения.
package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/caarlos0/env/v6"
)

const (
	defaultServerAddress     = "localhost:8080"
	defaultBaseURL           = "http://" + defaultServerAddress + "/"
	defaultFileStoragePath   = "shurldb.txt"
	defaultDatabaseDSN       = "postgresql://shurl_user:qazxswedc@localhost:5432/shurl"
	defaultGrpcServerAddress = "localhost:3200"
)

// Configuration содержит перечень настроек сервиса.
type Configuration struct {
	ServerAddress     string `env:"SERVER_ADDRESS" json:"server_address"`           // Адрес HTTP-сервера приложения
	BaseURL           string `env:"BASE_URL" json:"base_url"`                       // Корневой URL работающего сервиса
	FileStoragePath   string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`     // Путь к файлу для хранения данных сервиса
	DatabaseDSN       string `env:"DATABASE_DSN" json:"database_dsn"`               // Строка для подключения к базе данных
	EnableHTTPS       bool   `env:"ENABLE_HTTPS" json:"enable_https"`               // Признак "включить поддержку HTTPS"
	ConfigFilePath    string `env:"CONFIG" json:"-"`                                // Путь к файлу с настройками сервиса
	TrustedSubnet     string `env:"TRUSTED_SUBNET" json:"trusted_subnet"`           // IP-подсеть, из которой разрешены запросы статистики сервиса
	GrpcServerAddress string `env:"GRPC_SERVER_ADDRESS" json:"grpc_server_address"` // Адрес gRPC-сервера приложения
}

// NewConfiguration создаёт перечень настроек сервиса.
func NewConfiguration() *Configuration {
	cfg := new(Configuration)

	cfg.fillFromFlags()

	err := cfg.fillFromEnvironment()
	if err != nil {
		log.Println(err)
	}

	if cfg.ConfigFilePath != "" {
		err = cfg.fillFromFile()
		if err != nil {
			log.Println(err)
		}
	}

	if cfg.ServerAddress == "" {
		cfg.ServerAddress = defaultServerAddress
	}

	if cfg.GrpcServerAddress == "" {
		cfg.GrpcServerAddress = defaultGrpcServerAddress
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}

	baseURL := []rune(cfg.BaseURL)
	if baseURL[len(baseURL)-1] != '/' {
		cfg.BaseURL += "/"
	}

	log.Println("Resulting config:", cfg)

	return cfg
}

func (c *Configuration) fillFromFlags() {
	flag.StringVar(&c.ServerAddress, "a", "", "string with HTTP-server address")
	flag.StringVar(&c.GrpcServerAddress, "g", "", "string with gRPC-server address")
	flag.StringVar(&c.BaseURL, "b", "", "string with base URL")
	flag.StringVar(&c.FileStoragePath, "f", "", "string with file storage path")
	flag.StringVar(&c.DatabaseDSN, "d", "", "string with database data source name")
	flag.BoolVar(&c.EnableHTTPS, "s", false, "flag to use HTTPS protocol instead of HTTP")
	flag.StringVar(&c.ConfigFilePath, "c", "", "path to configuration file")
	flag.StringVar(&c.ConfigFilePath, "config", "", "path to configuration file")
	flag.StringVar(&c.TrustedSubnet, "t", "", "trusted subnet that is allowed to check service statistics")

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

func (c *Configuration) fillFromFile() error {
	log.Println("Start reading the file", c.ConfigFilePath)

	content, err := os.ReadFile(c.ConfigFilePath)
	if err != nil {
		return err
	}

	log.Println(string(content))

	var tmpConfig Configuration
	err = json.Unmarshal(content, &tmpConfig)
	if err != nil {
		return err
	}

	log.Println("Configuration from file '", c.ConfigFilePath, "':\n", tmpConfig)

	if tmpConfig.ServerAddress != "" && c.ServerAddress == "" {
		c.ServerAddress = tmpConfig.ServerAddress
	}

	if tmpConfig.GrpcServerAddress != "" && c.GrpcServerAddress == "" {
		c.GrpcServerAddress = tmpConfig.GrpcServerAddress
	}

	if tmpConfig.BaseURL != "" && c.BaseURL == "" {
		c.BaseURL = tmpConfig.BaseURL
	}

	if tmpConfig.FileStoragePath != "" && c.FileStoragePath == "" {
		c.FileStoragePath = tmpConfig.FileStoragePath
	}

	if tmpConfig.DatabaseDSN != "" && c.DatabaseDSN == "" {
		c.DatabaseDSN = tmpConfig.DatabaseDSN
	}

	if tmpConfig.EnableHTTPS && !c.EnableHTTPS {
		c.EnableHTTPS = true
	}

	if tmpConfig.TrustedSubnet != "" && c.TrustedSubnet == "" {
		c.TrustedSubnet = tmpConfig.TrustedSubnet
	}

	return nil
}
