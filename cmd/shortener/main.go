package main

import (
	"log"

	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/server"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"github.com/caarlos0/env/v6"
)

const serverAddress = "localhost:8080"

type configuration struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

func newConfig() configuration {
	cfg := configuration{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Println(err)
	}

	log.Println("Environment config:", cfg)

	if cfg.ServerAddress == "" {
		cfg.ServerAddress = serverAddress
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://" + cfg.ServerAddress + "/"
	}

	baseURL := []rune(cfg.BaseURL)
	if baseURL[len(baseURL)-1] != '/' {
		cfg.BaseURL += "/"
	}
	/*
	if cfg.FileStoragePath != "" {
		fileStoragePath := []rune(cfg.FileStoragePath)
		if fileStoragePath[len(fileStoragePath)-1] != '/' {
			cfg.FileStoragePath += "/"
		}
	}

	 */

	log.Println("Resulting config:", cfg)
	return cfg
}

func main() {
	cfg := newConfig()

	str := storage.NewStorage(cfg.FileStoragePath)
	defer str.CloseFile()

	h := handlers.NewHandler(str, cfg.BaseURL)

	srv := server.NewServer(cfg.ServerAddress, h)
	log.Fatal(srv.ListenAndServe())
}
