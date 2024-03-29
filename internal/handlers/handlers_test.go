package handlers

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/StainlessSteelSnake/shurl/internal/auth"
	"github.com/StainlessSteelSnake/shurl/internal/storage"

	"github.com/stretchr/testify/assert"
)

type dummyStorage struct {
	container map[string]string
	usersURLs map[string][]string
}

func (s *dummyStorage) AddURL(l, user string) (string, error) {
	s.container[l] = l
	s.usersURLs[user] = append(s.usersURLs[user], l)
	return l, nil
}

func (s *dummyStorage) FindURL(sh string) (storage.MemoryRecord, error) {
	if l, ok := s.container[sh]; ok {
		return storage.MemoryRecord{LongURL: l, User: "", Deleted: false}, nil
	}
	return storage.MemoryRecord{LongURL: "", User: "", Deleted: false}, errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}

func (s *dummyStorage) GetURLsByUser(u string) []string {
	return s.usersURLs[u]
}

func (s *dummyStorage) GetStatistics() (int, int) {
	return 1, 1
}

func (s *dummyStorage) Ping() error {
	return nil
}

func (s *dummyStorage) CloseFunc() func() {
	return nil
}

func (s *dummyStorage) AddURLs(b storage.BatchURLs, user string) (storage.BatchURLs, error) {
	for _, record := range b {
		s.container[record.ID] = record.URL
		s.usersURLs[user] = append(s.usersURLs[user], record.URL)

	}
	return b, nil
}

func (s *dummyStorage) DeleteURLs(urls []string, user string) []string {
	return urls
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

func BenchmarkNewHandler(b *testing.B) {
	tests := []struct {
		name    string
		storage map[string]string
		user    map[string][]string
		host    string
		baseURL string
		request string
		method  string
		want    int
	}{
		{
			name:    "Неуспешный PUT-запрос",
			storage: map[string]string{"dummy": "https://ya.ru"},
			user:    map[string][]string{"user1": {"https://ya.ru"}},
			host:    "localhost:8080",
			baseURL: "http://localhost:8080/",
			request: "localhost:8080/dummy",
			method:  http.MethodPut,
			want:    http.StatusBadRequest,
		},
		{
			name:    "Неуспешный GET-запрос",
			storage: map[string]string{"dummy": "https://ya.ru"},
			user:    map[string][]string{"user2": {"https://ya.ru"}},
			host:    "localhost:8080",
			baseURL: "http://localhost:8080/",
			request: "localhost:8080/dummy",
			method:  http.MethodGet,
			want:    http.StatusBadRequest,
		},
		{
			name:    "Неуспешный POST-запрос",
			storage: map[string]string{"dummy": "https://ya.ru"},
			user:    map[string][]string{"user3": {"https://ya.ru"}},
			host:    "localhost:8080",
			baseURL: "http://localhost:8080/",
			request: "localhost:8080/dummy",
			method:  http.MethodPost,
			want:    http.StatusBadRequest,
		},
	}

	for i := 0; i < b.N; i++ {
		for _, tt := range tests {
			b.Run(tt.name, func(b *testing.B) {
				s := &dummyStorage{tt.storage, tt.user}
				h := NewHandler(s, tt.baseURL, auth.NewAuth(), "")

				request := httptest.NewRequest(tt.method, tt.request, nil)
				writer := httptest.NewRecorder()

				h.ServeHTTP(writer, request)

				result := writer.Result()
				if err := result.Body.Close(); err != nil {
					b.Fatal(err)
				}
			})
		}
	}
}

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name    string
		storage map[string]string
		user    map[string][]string
		host    string
		baseURL string
		request string
		method  string
		want    int
	}{
		{
			name:    "Неуспешный PUT-запрос",
			storage: map[string]string{"dummy": "https://ya.ru"},
			user:    map[string][]string{"user1": {"https://ya.ru"}},
			host:    "localhost:8080",
			baseURL: "http://localhost:8080/",
			request: "localhost:8080/dummy",
			method:  http.MethodPut,
			want:    http.StatusBadRequest,
		},
		{
			name:    "Неуспешный GET-запрос",
			storage: map[string]string{"dummy": "https://ya.ru"},
			user:    map[string][]string{"user2": {"https://ya.ru"}},
			host:    "localhost:8080",
			baseURL: "http://localhost:8080/",
			request: "localhost:8080/dummy",
			method:  http.MethodGet,
			want:    http.StatusBadRequest,
		},
		{
			name:    "Неуспешный POST-запрос",
			storage: map[string]string{"dummy": "https://ya.ru"},
			user:    map[string][]string{"user3": {"https://ya.ru"}},
			host:    "localhost:8080",
			baseURL: "http://localhost:8080/",
			request: "localhost:8080/dummy",
			method:  http.MethodPost,
			want:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &dummyStorage{tt.storage, tt.user}
			h := NewHandler(s, tt.baseURL, auth.NewAuth(), "")

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
		t.Run(tt.name, func(t *testing.T) {
			h := Handler{storage: &dummyStorage{tt.storage, tt.user}}

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

func Benchmark_getLongURL(b *testing.B) {
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

	for i := 0; i < b.N; i++ {
		for _, tt := range tests {
			b.Run(tt.name, func(b *testing.B) {
				h := Handler{storage: &dummyStorage{tt.storage, tt.user}}

				writer := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, tt.request, nil)

				h.getLongURL(writer, request)
				result := writer.Result()
				if err := result.Body.Close(); err != nil {
					b.Fatal(err)
				}
			})
		}
	}
}

func Test_postLongURL(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			h := Handler{storage: &dummyStorage{tt.storage, tt.user}, auth: auth.NewAuth()}

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

func Benchmark_postLongURL(b *testing.B) {
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

	for i := 0; i < b.N; i++ {
		for _, tt := range tests {
			b.Run(tt.name, func(b *testing.B) {
				h := Handler{storage: &dummyStorage{tt.storage, tt.user}, auth: auth.NewAuth()}

				writer := httptest.NewRecorder()
				requestBody := strings.NewReader(tt.longURL)

				request := httptest.NewRequest(http.MethodPost, "/", requestBody)
				request.Host = tt.host
				h.postLongURL(writer, request)

				result := writer.Result()

				_, err := io.ReadAll(result.Body)
				if err != nil {
					b.Fatal(err)
				}

				if err = result.Body.Close(); err != nil {
					b.Fatal(err)
				}
			})
		}
	}
}

func Test_postLongURLinJSON(t *testing.T) {
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
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user1": {"https://ya.ru"}},
			longURL:  `{"url":"https://ya.ru"}`,
			wantCode: http.StatusCreated,
			wantURL:  "http://localhost:8080/",
		},
		{
			name:     "Неуспешный запрос, в теле не передан URL",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user2": {"https://ya.ru"}},
			longURL:  ``,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, неправильное название поля в JSON",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user3": {"https://ya.ru"}},
			longURL:  `{"URL1":"https://ya.ru"}`,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, неправильный формат поля в JSON",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user4": {"https://ya.ru"}},
			longURL:  `{url:"https://ya.ru"}`,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, некорректная структура JSON",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user5": {"https://ya.ru"}},
			longURL:  `{"url":"https://ya.ru`,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler{storage: &dummyStorage{tt.storage, tt.user}, auth: auth.NewAuth()}

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

func Benchmark_postLongURLinJSON(b *testing.B) {
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
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user1": {"https://ya.ru"}},
			longURL:  `{"url":"https://ya.ru"}`,
			wantCode: http.StatusCreated,
			wantURL:  "http://localhost:8080/",
		},
		{
			name:     "Неуспешный запрос, в теле не передан URL",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user2": {"https://ya.ru"}},
			longURL:  ``,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, неправильное название поля в JSON",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user3": {"https://ya.ru"}},
			longURL:  `{"URL1":"https://ya.ru"}`,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, неправильный формат поля в JSON",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user4": {"https://ya.ru"}},
			longURL:  `{url:"https://ya.ru"}`,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
		{
			name:     "Неуспешный запрос, некорректная структура JSON",
			host:     "localhost:8080",
			storage:  map[string]string{"dummy": "https://ya.ru"},
			user:     map[string][]string{"user5": {"https://ya.ru"}},
			longURL:  `{"url":"https://ya.ru`,
			wantCode: http.StatusBadRequest,
			wantURL:  "",
		},
	}

	for i := 0; i < b.N; i++ {
		for _, tt := range tests {
			b.Run(tt.name, func(b *testing.B) {
				h := Handler{storage: &dummyStorage{tt.storage, tt.user}, auth: auth.NewAuth()}

				writer := httptest.NewRecorder()
				requestBody := strings.NewReader(tt.longURL)

				request := httptest.NewRequest(http.MethodPost, "/api/shorten", requestBody)
				request.Host = tt.host
				h.postLongURLinJSON(writer, request)

				result := writer.Result()

				_, err := io.ReadAll(result.Body)
				if err != nil {
					b.Fatal(err)
				}

				if err = result.Body.Close(); err != nil {
					b.Fatal(err)
				}
			})
		}
	}
}
