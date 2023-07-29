package main

import (
	"context"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/StainlessSteelSnake/shurl/internal/config"
	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/server"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"golang.org/x/crypto/acme/autocert"
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

	var canTerminate = make(chan struct{})
	var signalChannel = make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	go func() {
		s := <-signalChannel

		log.Println("Signal was received:", s)

		err := srv.Shutdown(ctx)
		if err != nil {
			log.Fatal(err)
		}

		deletionCancel()

		close(canTerminate)
	}()

	if cfg.EnableHTTPS {
		manager := &autocert.Manager{
			Cache:      autocert.DirCache("cache-dir"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("localhost", cfg.ServerAddress),
		}

		srv.TLSConfig = manager.TLSConfig()
		log.Fatal(srv.ListenAndServeTLS("", ""))
	} else {
		log.Fatal(srv.ListenAndServe())
	}

	<-canTerminate
	log.Println("Terminating the server.")
}
