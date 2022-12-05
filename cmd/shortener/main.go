package main

import (
	"github.com/StainlessSteelSnake/shurl/internal/app"
	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"log"
)

const host = "localhost:8080"

func main() {
	str := storage.NewStorage(map[string]string{})
	h := handlers.NewHandler(str)

	server := app.NewServer(host, h)
	log.Fatal(server.ListenAndServe())
}
