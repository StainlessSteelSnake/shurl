package main

import (
	"log"

	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/server"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
)

const host = "localhost:8080"

func main() {
	str := storage.NewStorage()
	h := handlers.NewHandler(str)

	server := server.NewServer(host, h)
	log.Fatal(server.ListenAndServe())
}
