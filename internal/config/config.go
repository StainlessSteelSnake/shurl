package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

const serverAddress = "localhost:8080"
const baseURL = "http://" + serverAddress + "/"
const fileStoragePath = "shurldb.txt"

type Configuration struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

func (c *Configuration) fillFromFlags() {
	flag.StringVar(&c.ServerAddress, "a", serverAddress, "string with server address")
	flag.StringVar(&c.BaseURL, "b", baseURL, "string with base URL")
	flag.StringVar(&c.FileStoragePath, "f", fileStoragePath, "string with file storage path")

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

	log.Println("Resulting config:", cfg)

	return cfg
}
