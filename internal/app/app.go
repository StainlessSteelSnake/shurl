package app

import (
	"github.com/StainlessSteelSnake/shurl/internal/handlers"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"log"
	"net/http"
)

// host Доменное имя и порт сервера по-умолчанию
const host = "localhost:8080"

// server Текущие настройки и состояние сервера
type server struct {
	host string // Доменное имя и порт сервера
}

// newServer Получение настроек сервера по-умолчанию
func newServer() *server {
	return &server{host} // Доменное имя и порт сервера по-умолчанию
}

// Start Запуск веб-сервера для сервиса обработки коротких ссылок
func Start() {
	// Получение настройки сервера по-умолчанию
	server := newServer()

	// Получение экземпляра хранилища коротких ссылок
	storager := storage.New()

	// Инициализация обработчика входящих запросов к сервису
	h := http.HandlerFunc(handlers.GlobalHandler(storager, server.host))
	http.Handle("/", h)

	// Запуск HTTP-сервера
	log.Fatal(http.ListenAndServe(server.host, nil))
}
