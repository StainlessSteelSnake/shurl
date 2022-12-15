package main

import (
	"log"

	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/server"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"github.com/caarlos0/env/v6"
)

type configuration struct {
	ServerAddress string `env:"SERVER_ADDRESS,required"`
	BaseURL       string `env:"BASE_URL,required"`
}

func main() {
	cfg := configuration{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	str := storage.NewStorage()
	h := handlers.NewHandler(str, cfg.BaseURL)

	srv := server.NewServer(cfg.ServerAddress, h)
	log.Fatal(srv.ListenAndServe())
}
