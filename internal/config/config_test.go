package config

import (
	"reflect"
	"testing"
)

func TestConfiguration_fillFromEnvironment(t *testing.T) {
	type fields struct {
		ServerAddress   string
		BaseURL         string
		FileStoragePath string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Configuration{
				ServerAddress:   tt.fields.ServerAddress,
				BaseURL:         tt.fields.BaseURL,
				FileStoragePath: tt.fields.FileStoragePath,
			}
			if err := c.fillFromEnvironment(); (err != nil) != tt.wantErr {
				t.Errorf("fillFromEnvironment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfiguration_fillFromFlags(t *testing.T) {
	type fields struct {
		ServerAddress   string
		BaseURL         string
		FileStoragePath string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Configuration{
				ServerAddress:   tt.fields.ServerAddress,
				BaseURL:         tt.fields.BaseURL,
				FileStoragePath: tt.fields.FileStoragePath,
			}
			c.fillFromFlags()
		})
	}
}

func TestNewConfiguration(t *testing.T) {
	tests := []struct {
		name string
		want *Configuration
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewConfiguration(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConfiguration() = %v, want %v", got, tt.want)
			}
		})
	}
}
