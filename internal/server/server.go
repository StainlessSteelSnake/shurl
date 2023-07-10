// Пакет server отвечает за создание и настройку HTTP-сервера приложения.
package server

import "net/http"

// Server содержит ссылку на настройки и экземпляр HTTP-сервера.
type Server struct {
	http.Server
}

// NewServer создаёт экземпляр HTTP-сервера с заданным адресом и ссылкой на обработчик HTTP-запросов.
func NewServer(host string, handler http.Handler) *Server {
	return &Server{
		http.Server{
			Addr:    host,
			Handler: handler,
		},
	}
}
