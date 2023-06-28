package main

import (
	_ "net/http/pprof"

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

	str := storage.NewStorage(cfg.FileStoragePath, cfg.DatabaseDSN, ctx)
	if closeFunc := str.CloseFunc(); closeFunc != nil {
		defer closeFunc()
	}

	h := handlers.NewHandler(str, cfg.BaseURL)

	srv := server.NewServer(cfg.ServerAddress, h)
	log.Fatal(srv.ListenAndServe())
}
