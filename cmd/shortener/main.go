package main

import (
	"context"
	"log"
	_ "net/http/pprof"

	"github.com/StainlessSteelSnake/shurl/internal/config"
	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/server"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
)

func main() {
	cfg := config.NewConfiguration()
	ctx := context.Background()

	var h *handlers.Handler

	deletionContext, deletionCancel := context.WithCancel(ctx)
	if cfg.DatabaseDSN != "" {
		dStorage := storage.NewDBStorage(ctx, storage.NewMemoryStorage(), cfg.DatabaseDSN)
		dStorage.DeletionCancel = deletionCancel
		dStorage.DeletionQueueProcess(deletionContext)
		defer dStorage.CloseFunc()

		h = handlers.NewHandler(dStorage, cfg.BaseURL)
	} else {
		dStorage := storage.NewMemoryStorage()
		dStorage.DeletionCancel = deletionCancel
		dStorage.DeletionQueueProcess(deletionContext)
		defer dStorage.CloseFunc()

		h = handlers.NewHandler(dStorage, cfg.BaseURL)
	}

	/*str := storage.NewStorage(ctx, cfg.FileStoragePath, cfg.DatabaseDSN)
	if closeFunc := str.CloseFunc(); closeFunc != nil {
		defer closeFunc()
	}
	h := handlers.NewHandler(str, cfg.BaseURL)
	*/
	srv := server.NewServer(cfg.ServerAddress, h)
	log.Fatal(srv.ListenAndServe())
}
