package app

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
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
			&Server{http.Server{Addr: "localhost:8080"}},
		},
		{
			"Positive case 2",
			"127.0.0.1:8888",
			&Server{http.Server{Addr: "127.0.0.1:8888"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := new(http.Handler)
			s := NewServer(tt.host, *h)
			require.NotNil(t, s)
			assert.Equal(t, tt.want.Addr, s.Addr)
		})
	}
}
