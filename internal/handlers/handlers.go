package handlers

import (
	"compress/gzip"
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

var baseURL string

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func gzipHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			log.Println("Клиент не принимает ответы в gzip")
			next(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			log.Println("Ошибка при формировании ответа в gzip:", err)
			http.Error(w, "ошибка при формировании ответа в gzip: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	}

}

func decodeRequest(r *http.Request) ([]byte, error) {
	if r.Header.Get("Content-Encoding") != "gzip" {
		log.Println("Тело запроса пришло не в gzip")
		return io.ReadAll(r.Body)
	}

	reader, err := gzip.NewReader(r.Body)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func NewHandler(s Storager, bURL string) *Handler {
	baseURL = bURL

	handler := &Handler{
		chi.NewMux(),
		s,
	}

	handler.Route("/", func(r chi.Router) {
		r.Get("/{id}", gzipHandler(handler.getLongURL))
		r.Post("/", gzipHandler(handler.postLongURL))
		r.Post("/api/shorten", gzipHandler(handler.postLongURLinJSON))
		r.MethodNotAllowed(handler.badRequest)
	})

	return handler
}

func (h *Handler) badRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "неподдерживаемый запрос: '"+r.RequestURI+"'", http.StatusBadRequest)
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

func (h *Handler) postLongURL(w http.ResponseWriter, r *http.Request) {
	b, e := decodeRequest(r)
	if e != nil {
		log.Println("Неверный формат данных в запросе:", e)
		http.Error(w, "неверный формат данных в запросе: "+e.Error(), http.StatusBadRequest)
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
		http.Error(w, "ошибка при добавлении в БД: "+e.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("Созданный короткий идентификатор URL:", sh)

	w.WriteHeader(http.StatusCreated)
	_, e = w.Write([]byte(baseURL + sh))
	if e != nil {
		log.Println("Ошибка при записи ответа в тело запроса:", e)
	}
}

func (h *Handler) postLongURLinJSON(w http.ResponseWriter, r *http.Request) {
	b, e := decodeRequest(r)
	if e != nil {
		log.Println("Неверный формат данных в запросе:", e)
		http.Error(w, "неверный формат данных в запросе: "+e.Error(), http.StatusBadRequest)
		return
	}

	requestBody := PostRequestBody{}
	e = json.Unmarshal(b, &requestBody)
	if e != nil {
		log.Println("Неверный формат данных в запросе:", e)
		http.Error(w, "неверный формат данных в запросе: "+e.Error(), http.StatusBadRequest)
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
		http.Error(w, "ошибка при добавлении в БД: "+e.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("Созданный короткий идентификатор URL:", sh)

	response, e := json.Marshal(PostResponseBody{baseURL + sh})
	if e != nil {
		log.Println("Ошибка '", e, "' при формировании ответа:", requestBody.URL)
		http.Error(w, "ошибка при при формировании ответа: "+e.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, e = w.Write(response)
	if e != nil {
		log.Println("Ошибка при записи ответа в тело запроса:", e)
	}
}
