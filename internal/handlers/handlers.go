package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Storager interface {
	AddURL(l string) (string, error)
	FindURL(sh string) (string, error)
}

type Handler struct {
	*chi.Mux
	storage Storager
}

type PostRequestBody struct {
	URL string `json:"url"`
}

type PostResponseBody struct {
	Result string `json:"result"`
}

const protocolPrefix string = "http://"

func NewHandler(s Storager) *Handler {
	handler := &Handler{
		chi.NewMux(),
		s,
	}

	handler.Route("/", func(r chi.Router) {
		r.Get("/{id}", handler.getLongURL)
		r.Post("/", handler.postLongURL)
		r.Post("/api/shorten", handler.postLongURLinJSON)
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
	_, e = w.Write([]byte(protocolPrefix + r.Host + "/" + sh))
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

func (h *Handler) postLongURLinJSON(w http.ResponseWriter, r *http.Request) {
	b, e := io.ReadAll(r.Body)
	if e != nil {
		log.Println("Неверный формат данных в запросе")
		http.Error(w, "неверный формат данных в запросе", http.StatusBadRequest)
		return
	}

	requestBody := PostRequestBody{}
	e = json.Unmarshal(b, &requestBody)
	if e != nil {
		log.Println("Неверный формат данных в запросе")
		http.Error(w, "неверный формат данных в запросе", http.StatusBadRequest)
		return
	}

	log.Println("Пришедший в запросе исходный URL:", requestBody.URL)
	if len(requestBody.URL) == 0 {
		log.Println("Неверный формат URL")
		http.Error(w, "неверный формат URL", http.StatusBadRequest)
		return
	}

	sh, e := h.storage.AddURL(requestBody.URL)
	if e != nil {
		log.Println("Ошибка '", e, "' при добавлении в БД URL:", requestBody.URL)
		http.Error(w, "ошибка при добавлении в БД", http.StatusInternalServerError)
		return
	}
	log.Println("Созданный короткий идентификатор URL:", sh)

	response, e := json.Marshal(PostResponseBody{protocolPrefix + r.Host + "/" + sh})
	if e != nil {
		log.Println("Ошибка '", e, "' при формировании ответа:", requestBody.URL)
		http.Error(w, "ошибка при при формировании ответа", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, e = w.Write(response)
	if e != nil {
		log.Println("Ошибка при записи ответа в тело запроса:", e)
	}
}
