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

type handler struct {
	*chi.Mux
	Storage storage.URLAddFinder
}

func (h *handler) badRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	http.Error(w, "неподдерживаемый запрос: '"+r.RequestURI+"'", http.StatusBadRequest)
}

func (h *handler) postLongURL(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	b, e := io.ReadAll(r.Body)
	if e != nil {
		log.Println("Неверный формат URL")
		http.Error(w, "неверный формат URL", http.StatusBadRequest)
		return
	}

	l := string(b)
	log.Println("Пришедший в запросе исходный URL:", l)
	if len(l) == 0 {
		log.Println("Неверный формат URL")
		http.Error(w, "неверный формат URL", http.StatusBadRequest)

		return
	}

	sh, e := h.Storage.AddURL(l)
	if e != nil {
		log.Println("Ошибка '", e, "' при добавлении в БД URL:", l)
		http.Error(w, "ошибка при добавлении в БД", http.StatusInternalServerError)
		return
	}
	log.Println("Созданный короткий идентификатор URL:", sh)

	w.WriteHeader(http.StatusCreated)
	_, e = w.Write([]byte("http://" + r.Host + "/" + sh))
	if e != nil {
		log.Println("Ошибка при записи ответа в тело запроса:", e)
	}
}

func (h *handler) getLongURL(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	log.Println("Полученный GET-запрос:", r.URL)

	sh := strings.Trim(r.URL.Path, "/")
	log.Println("Идентификатор короткого URL, полученный из GET-запроса:", sh)

	l, e := h.Storage.FindURL(sh)
	if e != nil {
		log.Println("Ошибка '", e, "'. Не найден URL с указанным коротким идентификатором:", sh)
		http.Error(w, "URL с указанным коротким идентификатором не найден", http.StatusBadRequest)
		return
	}
	log.Println("Найден URL", l, "для короткого идентификатора", sh)

	w.Header().Set("Location", l)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *handler) route() {
	h.Use(middleware.RequestID)
	h.Use(middleware.Logger)
	h.Use(middleware.Recoverer)

	h.Route("/", func(r chi.Router) {
		r.Get("/{id}", h.getLongURL)
		r.Post("/", h.postLongURL)
		r.MethodNotAllowed(h.badRequest)
	})
}

func NewHandler(s storage.URLAddFinder) http.Handler {
	handler := &handler{
		Mux:     chi.NewMux(),
		Storage: s,
	}

	handler.route()
	return handler
}
