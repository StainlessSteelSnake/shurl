package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/StainlessSteelSnake/shurl/internal/auth"
)

func ExampleHandler_badRequest() {
	h := Handler{storage: nil}

	request := httptest.NewRequest(http.MethodPut, "localhost:8080/dummy/dummy", nil)
	writer := httptest.NewRecorder()

	h.badRequest(writer, request)

	result := writer.Result()
	if err := result.Body.Close(); err != nil {
		log.Fatal(err)
	}
}

func ExampleHandler_getLongURL() {
	tests := []struct {
		name     string
		storage  map[string]string
		user     map[string][]string
		request  string
		wantCode int
		wantURL  string
	}{
		{
			name:     "Неуспешный запрос, ошибка в идентификаторе",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user1": {"https://ya.ru"}},
			request:  "/dummy1",
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, не передан идентификатор",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user2": {"https://ya.ru"}},
			request:  "/",
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Успешный GET-запрос",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user3": {"https://ya.ru"}},
			request:  "/dummy",
			wantCode: http.StatusTemporaryRedirect,
			wantURL:  "https://ya.ru",
		},
	}

	for _, tt := range tests {
		h := Handler{storage: &dummyStorage{tt.storage, tt.user}}

		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, tt.request, nil)

		h.getLongURL(writer, request)
		result := writer.Result()

		if err := result.Body.Close(); err != nil {
			log.Fatal(err)
		}

		fmt.Println(result)
	}
}

func ExampleHandler_postLongURL() {
	tests := []struct {
		name     string
		host     string
		storage  map[string]string
		user     map[string][]string
		longURL  string
		wantCode int
		wantURL  string
	}{
		{
			name:     "Успешный запрос",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy1": "https://ya.ru"},
			user:     map[string][]string{"user1": {"dummy1"}},
			longURL:  "https://ya.ru",
			wantCode: http.StatusCreated,
			wantURL:  "http://localhost:8080/",
		},
		{
			name:     "Неуспешный запрос, в теле не передан URL",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy2": "https://ya.ru"},
			user:     map[string][]string{"user2": {"dummy2"}},
			longURL:  "",
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
	}
	for _, tt := range tests {
		h := Handler{storage: &dummyStorage{tt.storage, tt.user}, auth: auth.NewAuth()}

		writer := httptest.NewRecorder()
		requestBody := strings.NewReader(tt.longURL)

		request := httptest.NewRequest(http.MethodPost, "/", requestBody)
		request.Host = tt.host
		h.postLongURL(writer, request)

		result := writer.Result()

		resultBody, err := io.ReadAll(result.Body)
		if err != nil {
			log.Fatal(err)
		}

		if err = result.Body.Close(); err != nil {
			log.Fatal(err)
		}

		fmt.Println(resultBody)
	}
}
