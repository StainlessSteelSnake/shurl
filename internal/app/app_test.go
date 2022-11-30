package app

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStart(t *testing.T) {
	t.Skip()
}

func TestNewServer(t *testing.T) {
	tests := []struct {
		name string
		host string
		want *Server
	}{
		{
			"Positive case 1",
			"localhost:8080",
			&Server{Host: "localhost:8080"},
		},
		{
			"Positive case 2",
			"127.0.0.1:8888",
			&Server{Host: "127.0.0.1:8888"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(tt.host, nil)
			assert.Equal(t, tt.want.Host, s.Host)
		})
	}
}
