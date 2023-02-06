package main

import (
	"context"
	"log"

	"github.com/StainlessSteelSnake/shurl/internal/config"
	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/server"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
)

func main() {
	cfg := config.NewConfiguration()

	ctx := context.Background()

	str := storage.NewStorage(ctx, cfg.FileStoragePath, cfg.DatabaseDSN)
	if closeFunc := str.CloseFunc(); closeFunc != nil {
		defer closeFunc()
	}

	h := handlers.NewHandler(str, cfg.BaseURL)

	srv := server.NewServer(cfg.ServerAddress, h)
	log.Fatal(srv.ListenAndServe())
}
