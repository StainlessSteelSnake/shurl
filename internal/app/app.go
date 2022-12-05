package app

import "net/http"

type Server struct {
	Host string
	http.Server
}

func NewServer(host string, handler *http.Handler) *Server {
	return &Server{
		host,
		http.Server{
			Addr:    host,
			Handler: *handler,
		},
	}
}
