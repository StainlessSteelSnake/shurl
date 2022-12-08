package app

import "net/http"

type Server struct {
	http.Server
}

func NewServer(host string, handler http.Handler) *Server {
	return &Server{
		http.Server{
			Addr:    host,
			Handler: handler,
		},
	}
}
