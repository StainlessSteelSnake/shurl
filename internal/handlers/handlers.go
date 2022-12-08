package handlers

import (
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"net/http"
	"strings"
)

type Storager interface {
	AddURL(l string) (string, error)
	FindURL(sh string) (string, error)
}

type Handler struct {
	*chi.Mux
	storage Storager
}

func NewHandler(s Storager) *Handler {
	handler := &Handler{
		chi.NewMux(),
		s,
	}

	handler.Route("/", func(r chi.Router) {
		r.Get("/{id}", handler.getLongURL)
		r.Post("/", handler.postLongURL)
		r.MethodNotAllowed(handler.badRequest)
	})

	return handler
}

func (h *Handler) badRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "неподдерживаемый запрос: '"+r.RequestURI+"'", http.StatusBadRequest)
}

func (h *Handler) postLongURL(w http.ResponseWriter, r *http.Request) {
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

	sh, e := h.storage.AddURL(l)
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

func (h *Handler) getLongURL(w http.ResponseWriter, r *http.Request) {
	log.Println("Полученный GET-запрос:", r.URL)

	sh := strings.Trim(r.URL.Path, "/")
	log.Println("Идентификатор короткого URL, полученный из GET-запроса:", sh)

	l, e := h.storage.FindURL(sh)
	if e != nil {
		log.Println("Ошибка '", e, "'. Не найден URL с указанным коротким идентификатором:", sh)
		http.Error(w, "URL с указанным коротким идентификатором не найден", http.StatusBadRequest)
		return
	}
	log.Println("Найден URL", l, "для короткого идентификатора", sh)

	w.Header().Set("Location", l)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
