package storage

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		want AddFinder
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_storage_Add(t *testing.T) {
	type fields struct {
		container URLList
	}
	type args struct {
		l LongURL
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    ShortURL
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &storage{
				container: tt.fields.container,
			}
			got, err := s.Add(tt.args.l)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Add() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_storage_Find(t *testing.T) {
	type fields struct {
		container URLList
	}
	type args struct {
		sh ShortURL
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    LongURL
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &storage{
				container: tt.fields.container,
			}
			got, err := s.Find(tt.args.sh)
			if (err != nil) != tt.wantErr {
				t.Errorf("Find() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Find() got = %v, want %v", got, tt.want)
			}
		})
	}
}
