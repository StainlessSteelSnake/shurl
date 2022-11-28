package handlers

import (
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"log"
	"net/http"
	"strings"
)

// Handler Обработчик запросов для сервера и ссылка на хранилище URL
type handler struct {
	*chi.Mux
	Storage storage.AddFinder
}

// NewHandler Создание экземпляра обработчика запросов, со ссылкой на хранилище URL
func newHandler(s storage.AddFinder) *handler {
	return &handler{
		Mux:     chi.NewMux(),
		Storage: s,
	}
}

// badRequest
func (h *handler) badRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "неподдерживаемый запрос: '"+r.RequestURI+"'", http.StatusBadRequest)
}

// postLongURL Обработка входящих POST-запросов на создание короткого идентификтора для URL
func (h *handler) postLongURL(w http.ResponseWriter, r *http.Request) {

	// Считывание содержимого тела POST-запроса
	b, e := io.ReadAll(r.Body)
	if e != nil {
		log.Println("Неверный формат URL")
		http.Error(w, "неверный формат URL", http.StatusBadRequest)
		return
	}

	// Преобразование считанного содержимого к типу длинной ссылки в хранилище
	l := storage.LongURL(b)
	log.Println("Пришедший в запросе исходный URL:", l)
	if len(l) == 0 {
		log.Println("Неверный формат URL")
		http.Error(w, "неверный формат URL", http.StatusBadRequest)

		return
	}

	// Сохранение длинной ссылки в хранилище и получение её короткого идентификатора
	sh, e := h.Storage.Add(l)
	if e != nil {
		log.Println("Ошибка '", e, "' при добавлении в БД URL:", l)
		http.Error(w, "ошибка при добавлении в БД", http.StatusInternalServerError)
		return
	}
	log.Println("Созданный короткий идентификатор URL:", sh)

	// Формирование ответа: установка кода состояния "успешно создано"
	w.WriteHeader(http.StatusCreated)
	_, e = w.Write([]byte("http://" + r.Host + "/" + string(sh)))
	if e != nil {
		log.Println("Ошибка при записи ответа в тело запроса:", e)
	}
}

// getLongURL Обработка входящих GET-запросов на получение исходного URL по его короткому идентификатору
func (h *handler) getLongURL(w http.ResponseWriter, r *http.Request) {
	log.Println("Полученный GET-запрос:", r.URL)

	// Получение идентификатора короткой ссылки из URL запроса
	sh := storage.ShortURL(strings.Trim(r.URL.Path, "/"))
	log.Println("Идентификатор короткого URL, полученный из GET-запроса:", sh)

	// Поиск длинной ссылки в хранилище по идентификатору короткой ссылки
	l, e := h.Storage.Find(sh)
	if e != nil {
		log.Println("Ошибка '", e, "'. Не найден URL с указанным коротким идентификатором:", sh)
		http.Error(w, "URL с указанным коротким идентификатором не найден", http.StatusBadRequest)
		return
	}
	log.Println("Найден URL", l, "для короткого идентификатора", sh)

	// Формирование ответа:
	// - установка заголовка "Location" в значение найденной длинной ссылки;
	// - установка кода состояния "временно перемещено"
	w.Header().Set("Location", string(l))
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// GlobalHandler Обработка всех входящих запросов, вне зависимости от метода
func GlobalHandler(s storage.AddFinder) http.Handler {
	// Создание экземпляр обработчика для последующего использования сервером.
	// Экзмепляр обработчика содержит в себе ссылку на интерфейс хранилища записей соответствия коротких и полных URL.
	router := newHandler(s)

	// Использование прослойки для присвоения запросам уникальных идентификаторов
	router.Use(middleware.RequestID)

	// Использование прослойки для логирования запросов
	router.Use(middleware.Logger)

	// Использование прослойки для восстановления сервера в случае сбоев (срабатывания паники)
	router.Use(middleware.Recoverer)

	// Маршрутизация запросов GET и POST. Обработка неподдерживаемых запросов.
	router.Route("/", func(r chi.Router) {
		r.Get("/{id}", router.getLongURL)
		r.Post("/", router.postLongURL)
		r.MethodNotAllowed(router.badRequest)
	})

	return router
}
