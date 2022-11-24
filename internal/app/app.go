package app

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const address = "localhost:8080"

type (
	shortURL string
	longURL  string
)
type urlList map[shortURL]longURL

type Server struct {
	db      urlList
	address string
}

var server *Server

// new Создание локального хранилища для коротких идентификаторов URL
func newServer() *Server {
	return &Server{make(urlList), address}
}

// add Создание короткого идентификатора для URL
func (s *Server) add(l longURL) (shortURL, error) {
	t := time.Now()

	sh := shortURL(strconv.FormatInt(t.UnixMicro(), 36))

	if _, ok := s.db[sh]; ok {
		return "", errors.New("короткий URL с ID " + string(sh) + " уже существует")
	}

	s.db[sh] = l
	return sh, nil
}

// find Поиск полного URL по его короткому идентификатору
func (s *Server) find(sh shortURL) (longURL, error) {
	if l, ok := s.db[sh]; ok {
		return l, nil
	}

	return "", errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}

// globalHandler Обработка всех входящих запросов, вне зависимости от метода
func globalHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		fmt.Println("Пришёл POST-запрос")
		postHandler(w, r)
	case "GET":
		fmt.Println("Пришёл GET-запрос")
		getHandler(w, r)
	default:
		fmt.Println("Пришёл неподдерживаемый запрос", r.Method)
		w.WriteHeader(http.StatusBadRequest)
	}
}

// postHandler Обработка входящих POST-запросов на создание короткого идентификтора для URL
func postHandler(w http.ResponseWriter, r *http.Request) {
	b, e := io.ReadAll(r.Body)
	if e != nil {
		fmt.Println("Неверный формат URL")
		http.Error(w, "неверный формат URL", http.StatusBadRequest)

		return
	}

	l := longURL(b)
	fmt.Println("Пришедший в запросе исходный URL:", l)

	sh, e := server.add(l)
	if e != nil {
		fmt.Println("Ошибка '", e, "' при добавлении в БД URL:", l)
		http.Error(w, "ошибка при добавлении в БД", http.StatusInternalServerError)
		return
	}
	fmt.Println("Созданный короткий идентификатор URL:", sh)

	w.WriteHeader(http.StatusCreated)
	_, e = w.Write([]byte("http://" + server.address + "/" + string(sh)))
	if e != nil {
		fmt.Println("Ошибка при записи ответа в тело запроса:", e)
	}
}

// getHandler Обработка входящих GET-запросов на получение исходного URL по его короткому идентификатору
func getHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Полученный GET-запрос:", r.URL)

	sh := shortURL(strings.Trim(r.URL.Path, "/"))
	fmt.Println("Идентификатор короткого URL, полученный из GET-запроса:", sh)

	l, e := server.find(sh)
	if e != nil {
		fmt.Println("Ошибка '", e, "'. Не найден URL с указанным коротким идентификатором:", sh)
		http.Error(w, "URL с указанным коротким идентификатором не найден", http.StatusBadRequest)
		return
	}
	fmt.Println("Найден URL", l, "для короткого идентификатора", sh)

	//http.Redirect(w, r, string(l), http.StatusTemporaryRedirect)

	w.Header().Set("Location", string(l))
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// Start Запуск веб-сервера для сервиса обработки коротких ссылок
func Start() {
	// Создаём экземпляр хранилища коротких URL
	server = newServer()

	// Запускаем HTTP-сервер для обработки входящих запросов
	http.HandleFunc("/", globalHandler)
	e := http.ListenAndServe(server.address, nil)
	fmt.Println("Ошибка в работе веб-сервера:", e)
}
