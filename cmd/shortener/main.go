package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/StainlessSteelSnake/shurl/internal/auth"
	"github.com/StainlessSteelSnake/shurl/internal/config"
	"github.com/StainlessSteelSnake/shurl/internal/grpcserv"
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

	_, err := os.Stdout.WriteString("Build version: " + buildVersion + "\n")
	if err != nil {
		log.Fatalln(err)
	}
	_, err = os.Stdout.WriteString("Build date: " + buildDate + "\n")
	if err != nil {
		log.Fatalln(err)
	}
	_, err = os.Stdout.WriteString("Build commit: " + buildCommit + "\n")
	if err != nil {
		log.Fatalln(err)
	}

	cfg := config.NewConfiguration()
	ctx := context.Background()

	var h *handlers.Handler
	var store storage.Storager

	deletionContext, deletionCancel := context.WithCancel(ctx)

	if cfg.DatabaseDSN != "" {
		dbStore := storage.NewDBStorage(ctx, storage.NewMemoryStorage(), cfg.DatabaseDSN)
		dbStore.DeletionCancel = deletionCancel
		dbStore.DeletionQueueProcess(deletionContext)
		store = dbStore

	} else {
		mStore := storage.NewMemoryStorage()
		mStore.DeletionCancel = deletionCancel
		mStore.DeletionQueueProcess(deletionContext)
		store = mStore
	}

	defer store.CloseFunc()

	authenticator := auth.NewAuth()

	h = handlers.NewHandler(store, cfg.BaseURL, authenticator, cfg.TrustedSubnet)

	srv := server.NewServer(cfg.ServerAddress, h)

	grpcServ, err := grpcserv.NewServer(cfg.GrpcServerAddress, cfg.BaseURL, store, authenticator)

	var canTerminate = make(chan struct{})
	var signalChannel = make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	go func() {
		s := <-signalChannel

		log.Println("Signal was received:", s)

		err := srv.Shutdown(ctx)
		if err != nil {
			log.Fatalln("HTTP(S) server shutdown error:", err)
		}

		grpcServ.GracefulStop()

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

		log.Println("Запуск HTTP-сервера с поддержкой TLS")
		err = srv.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			log.Fatalln("HTTPS server ListenAndServeTLS:", err)
		}
	} else {
		log.Println("Запуск HTTP-сервера")
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalln("HTTP server ListenAndServe:", err)
		}
	}

	<-canTerminate
	log.Println("Terminating the server.")
}
