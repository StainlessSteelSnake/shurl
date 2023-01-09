package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

const serverAddress = "localhost:8080"
const baseURL = "http://" + serverAddress + "/"
const fileStoragePath = "shurldb.txt"
const databaseDSN = "postgresql://shurl_user:qazxswedc@localhost:5432/shurl"

type Configuration struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

func (c *Configuration) fillFromFlags() {
	flag.StringVar(&c.ServerAddress, "a", serverAddress, "string with server address")
	flag.StringVar(&c.BaseURL, "b", baseURL, "string with base URL")
	flag.StringVar(&c.FileStoragePath, "f", fileStoragePath, "string with file storage path")
	flag.StringVar(&c.DatabaseDSN, "d", databaseDSN, "string with database data source name")

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

	if cfg.DatabaseDSN == "" {
		cfg.DatabaseDSN = databaseDSN
	}

	log.Println("Resulting config:", cfg)

	return cfg
}
