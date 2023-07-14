package main

import (
	"context"
	"log"
	_ "net/http/pprof"
	"os"

	"github.com/StainlessSteelSnake/shurl/internal/config"
	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/server"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
)

var (
	buildVersion, buildDate, buildCommit string
)

func main() {
	if buildVersion == "" {
		buildVersion = "N/A"
	}

	if buildDate == "" {
		buildDate = "N/A"
	}

	if buildCommit == "" {
		buildCommit = "N/A"
	}

	os.Stdout.WriteString("Build version: " + buildVersion + "\n")
	os.Stdout.WriteString("Build date: " + buildDate + "\n")
	os.Stdout.WriteString("Build commit: " + buildCommit + "\n")

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
