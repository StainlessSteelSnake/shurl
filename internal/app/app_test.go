package app

import (
	"testing"
)

func TestStart(t *testing.T) {
	tests := []struct {
		name string
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("Started", tt.name)
		})
	}
}

func Test_newServer(t *testing.T) {
	tests := []struct {
		name string
		host string
		want server
	}{
		{
			"Positive case 1",
			"localhost:8080",
			server{"localhost:8080"},
		},
		{
			"Positive case 2",
			"127.0.0.1:8888",
			server{"127.0.0.1:8888"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if s := newServer(tt.host); *s != tt.want {
				t.Errorf("newServer() = %v, want %v", *s, tt.want)
			}
		})
	}
}
