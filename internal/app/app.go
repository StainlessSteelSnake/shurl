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
	db urlList
}

var server *Server

// new Создание локального хранилища для коротких идентификаторов URL
func new() *Server {
	return &Server{make(urlList)}
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
		postHandler(w, r)
	case "GET":
		getHandler(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

// postHandler Обработка входящих POST-запросов на создание короткого идентификтора для URL
func postHandler(w http.ResponseWriter, r *http.Request) {
	b, e := io.ReadAll(r.Body)
	if e != nil {
		http.Error(w, "неверный формат URL", http.StatusBadRequest)
		return
	}

	l := longURL(b)
	fmt.Println("Long URL in request:", l)

	sh, e := server.add(l)
	if e != nil {
		http.Error(w, "ошибка при добавлении в БД", http.StatusInternalServerError)
		return
	}
	fmt.Println("Short URL ID in response:", sh)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(sh))
}

// getHandler Обработка входящих GET-запросов на получение исходного URL по его короткому идентификатору
func getHandler(w http.ResponseWriter, r *http.Request) {
	sh := shortURL(strings.Trim(r.URL.Path, "/"))
	fmt.Println("Short URL ID in request:", sh)

	l, e := server.find(sh)
	if e != nil {
		http.Error(w, "URL с указанным ID не найден", http.StatusBadRequest)
		return
	}
	fmt.Println("Long URL in response:", l)

	//http.Redirect(w, r, string(l), http.StatusTemporaryRedirect)

	w.Header().Set("Location", string(l))
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// Start Запуск веб-сервера для сервиса обработки коротких ссылок
func Start() {
	// Создаём экземпляр хранилища коротких URL
	server = new()

	// Запускаем HTTP-сервер для обработки входящих запросов
	http.HandleFunc("/", globalHandler)
	http.ListenAndServe(address, nil)
}
