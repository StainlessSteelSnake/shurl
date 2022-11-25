package app

import (
	"testing"
)

func TestStart(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Start()
		})
	}
}

func Test_newServer(t *testing.T) {
	tests := []struct {
		name string
		want server
	}{
		{
			"Positive case",
			server{"localhost:8080"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if s := newServer(); *s != tt.want {
				t.Errorf("newServer() = %v, want %v", *s, tt.want)
			}
		})
	}
}
