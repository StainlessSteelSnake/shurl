package handlers

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type storage struct {
	container map[string]string
}

func (s *storage) AddURL(l string) (string, error) {
	s.container[l] = l
	return l, nil
}

func (s *storage) FindURL(sh string) (string, error) {
	if l, ok := s.container[sh]; ok {
		return l, nil
	}
	return "", errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}

func TestGzipWriter_Write(t *testing.T) {
	t.Skip()
}

func Test_handler_gzipHandler(t *testing.T) {
	t.Skip()
}

func Test_decodeRequest(t *testing.T) {
	t.Skip()
}

func Test_handler_badRequest(t *testing.T) {
	tests := []struct {
		name     string
		URL      string
		method   string
		wantCode int
	}{
		{
			"Неправильный запрос (Put)",
			"localhost:8080/dummy/dummy",
			http.MethodPut,
			400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler{storage: nil}

			request := httptest.NewRequest(tt.method, tt.URL, nil)
			writer := httptest.NewRecorder()

			h.badRequest(writer, request)

			result := writer.Result()
			assert.Equal(t, tt.wantCode, result.StatusCode)
			if err := result.Body.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name    string
		storage map[string]string
		host    string
		baseURL string
		request string
		method  string
		want    int
	}{
		{
			name:    "Неуспешный PUT-запрос",
			storage: map[string]string{"dummy": "https://ya.ru"},
			host:    "localhost:8080",
			baseURL: "http://localhost:8080/",
			request: "localhost:8080/dummy",
			method:  http.MethodPut,
			want:    http.StatusBadRequest,
		},
		{
			name:    "Неуспешный GET-запрос",
			storage: map[string]string{"dummy": "https://ya.ru"},
			host:    "localhost:8080",
			baseURL: "http://localhost:8080/",
			request: "localhost:8080/dummy",
			method:  http.MethodGet,
			want:    http.StatusBadRequest,
		},
		{
			name:    "Неуспешный POST-запрос",
			storage: map[string]string{"dummy": "https://ya.ru"},
			host:    "localhost:8080",
			baseURL: "http://localhost:8080/",
			request: "localhost:8080/dummy",
			method:  http.MethodPost,
			want:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &storage{tt.storage}
			h := NewHandler(s, tt.baseURL)

			request := httptest.NewRequest(tt.method, tt.request, nil)
			writer := httptest.NewRecorder()

			h.ServeHTTP(writer, request)

			result := writer.Result()
			assert.Equal(t, tt.want, result.StatusCode)
			if err := result.Body.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_getLongURL(t *testing.T) {
	tests := []struct {
		name     string
		storage  map[string]string
		request  string
		wantCode int
		wantURL  string
	}{
		{
			name:     "Неуспешный запрос, ошибка в идентификаторе",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			request:  "/dummy1",
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, не передан идентификатор",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			request:  "/",
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Успешный GET-запрос",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			request:  "/dummy",
			wantCode: http.StatusTemporaryRedirect,
			wantURL:  "https://ya.ru",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler{storage: &storage{tt.storage}}

			writer := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.request, nil)

			h.getLongURL(writer, request)
			result := writer.Result()
			assert.Equal(t, tt.wantCode, result.StatusCode)
			assert.Equal(t, tt.wantURL, result.Header.Get("Location"))
			if err := result.Body.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_postLongURL(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		storage  map[string]string
		longURL  string
		wantCode int
		wantURL  string
	}{
		{
			name:     "Успешный запрос",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			longURL:  "https://ya.ru",
			wantCode: http.StatusCreated,
			wantURL:  "http://localhost:8080/",
		},
		{
			name:     "Неуспешный запрос, в теле не передан URL",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			longURL:  "",
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler{storage: &storage{tt.storage}}

			writer := httptest.NewRecorder()
			requestBody := strings.NewReader(tt.longURL)

			request := httptest.NewRequest(http.MethodPost, "/", requestBody)
			request.Host = tt.host
			h.postLongURL(writer, request)

			result := writer.Result()
			assert.Equal(t, tt.wantCode, result.StatusCode)

			resultBody, err := io.ReadAll(result.Body)
			if err != nil {
				t.Fatal(err)
			}
			assert.Contains(t, string(resultBody), tt.wantURL)

			if err = result.Body.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_postLongURLinJSON(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		storage  map[string]string
		longURL  string
		wantCode int
		wantURL  string
	}{
		{
			name:     "Успешный запрос",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			longURL:  `{"url":"https://ya.ru"}`,
			wantCode: http.StatusCreated,
			wantURL:  "http://localhost:8080/",
		},
		{
			name:     "Неуспешный запрос, в теле не передан URL",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			longURL:  ``,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, неправильное название поля в JSON",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			longURL:  `{"URL1":"https://ya.ru"}`,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, неправильный формат поля в JSON",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			longURL:  `{url:"https://ya.ru"}`,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, некорректная структура JSON",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			longURL:  `{"url":"https://ya.ru`,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler{storage: &storage{tt.storage}}

			writer := httptest.NewRecorder()
			requestBody := strings.NewReader(tt.longURL)

			request := httptest.NewRequest(http.MethodPost, "/api/shorten", requestBody)
			request.Host = tt.host
			h.postLongURLinJSON(writer, request)

			result := writer.Result()
			assert.Equal(t, tt.wantCode, result.StatusCode)

			resultBody, err := io.ReadAll(result.Body)
			if err != nil {
				t.Fatal(err)
			}
			assert.Contains(t, string(resultBody), tt.wantURL)

			if err = result.Body.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
