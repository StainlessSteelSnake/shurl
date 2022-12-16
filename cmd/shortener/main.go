package main

import (
	"flag"
	"log"

	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/server"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"github.com/caarlos0/env/v6"
)

const serverAddress = "localhost:8080"
const baseURL = "http://" + serverAddress + "/"
const fileStoragePath = "shurldb.txt"

type configuration struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

func newConfiguration() *configuration {
	cfg := new(configuration)

	flag.StringVar(&cfg.ServerAddress, "a", serverAddress, "string with server address")
	flag.StringVar(&cfg.BaseURL, "b", baseURL, "string with base URL")
	flag.StringVar(&cfg.FileStoragePath, "f", fileStoragePath, "string with file storage path")
	flag.Parse()
	log.Println("Console flags:", cfg)

	err := env.Parse(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Environment config:", cfg)

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

func main() {
	cfg := newConfiguration()

	str := storage.NewStorage(cfg.FileStoragePath)
	defer str.CloseFile()

	h := handlers.NewHandler(str, cfg.BaseURL)

	srv := server.NewServer(cfg.ServerAddress, h)
	log.Fatal(srv.ListenAndServe())
}
