package handlers

import (
	"errors"
	"fmt"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGlobalHandler(t *testing.T) {
	tests := []struct {
		name    string
		storage storage.URLList
		host    string
		request string
		method  string
		want    int
	}{
		{
			name:    "Неуспешный PUT-запрос",
			storage: storage.URLList{"asdfg": "https://ya.ru"},
			host:    "localhost:8080",
			request: "localhost:8080/asdfg",
			method:  http.MethodPut,
			want:    http.StatusBadRequest,
		},
		{
			name:    "Успешный GET-запрос",
			storage: storage.URLList{"asdfg": "https://ya.ru"},
			host:    "localhost:8080",
			request: "localhost:8080/asdfg",
			method:  http.MethodGet,
			want:    http.StatusBadRequest,
		},
		{
			name:    "Неуспешный POST-запрос",
			storage: storage.URLList{"asdfg": "https://ya.ru"},
			host:    "localhost:8080",
			request: "localhost:8080/asdfg",
			method:  http.MethodPost,
			want:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := storage.New()
			for _, ss := range tt.storage {
				s.Add(ss)
			}
			f := GlobalHandler(s, tt.host)

			request := httptest.NewRequest(tt.method, tt.request, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(f)
			h(w, request)
			result := w.Result()
			assert.Equal(t, tt.want, result.StatusCode)
		})
	}
}

type store storage.URLList

func (s store) Add(l storage.LongURL) (storage.ShortURL, error) {
	s["asdf"] = l
	return storage.ShortURL("asdf"), nil
}

func (s store) Find(sh storage.ShortURL) (storage.LongURL, error) {
	if l, ok := s[sh]; ok {
		return l, nil
	}
	return "", errors.New("not found")
}

func Test_getHandler(t *testing.T) {
	tests := []struct {
		name     string
		storage  storage.URLList
		request  string
		wantCode int
		wantURL  string
	}{
		{
			name:     "Неуспешный запрос",
			storage:  storage.URLList{"asdf": "https://ya.ru"},
			request:  "/asdfg",
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос",
			storage:  storage.URLList{"asdf": "https://ya.ru"},
			request:  "/",
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Успешный GET-запрос",
			storage:  storage.URLList{"asdf": "https://ya.ru"},
			request:  "/asdf",
			wantCode: http.StatusTemporaryRedirect,
			wantURL:  "https://ya.ru",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := make(store)
			for _, l := range tt.storage {
				s.Add(l)
			}
			fmt.Println(s)

			w := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.request, nil)
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				getHandler(s, w, r)
			})

			h(w, request)
			result := w.Result()
			assert.Equal(t, tt.wantCode, result.StatusCode)
			assert.Equal(t, tt.wantURL, result.Header.Get("Location"))
		})
	}
}

func Test_postHandler(t *testing.T) {
	type args struct {
		storager storage.AddFinder
		host     string
		w        http.ResponseWriter
		r        *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postHandler(tt.args.storager, tt.args.host, tt.args.w, tt.args.r)
		})
	}
}
