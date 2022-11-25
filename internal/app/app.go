package app

import (
	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"log"
	"net/http"
)

const host = "localhost:8080"

type server struct {
	host string
}

// new Создание локального хранилища для коротких идентификаторов URL
func newServer() *server {
	return &server{host}
}

// Start Запуск веб-сервера для сервиса обработки коротких ссылок
func Start() {
	// Создаём экземпляр хранилища коротких URL
	server := newServer()
	storager := storage.New()

	// Инициализация обработчика входящих запросов к сервису
	h := http.HandlerFunc(handlers.GlobalHandler(storager, server.host))
	http.Handle("/", h)

	// Запуск HTTP-сервера
	log.Fatal(http.ListenAndServe(server.host, nil))
}
