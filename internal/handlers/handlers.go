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
	AddURL(l, user string) (string, error)
	FindURL(sh string) (string, error)
	GetURLsByUser(string) []string
}

type Handler struct {
	*chi.Mux
	storage Storager
	user    *user
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

type shortAndLongURL struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"original_url"`
}

type shortAndLongURLs []shortAndLongURL

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
		new(user),
	}

	handler.Route("/", func(r chi.Router) {
		r.Get("/{id}", handler.handleCookie(gzipHandler(handler.getLongURL)))
		r.Get("/api/user/urls", handler.handleCookie(gzipHandler(handler.getLongURLsByUser)))
		r.Post("/", handler.handleCookie(gzipHandler(handler.postLongURL)))
		r.Post("/api/shorten", handler.handleCookie(gzipHandler(handler.postLongURLinJSON)))
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

func (h *Handler) getLongURLsByUser(w http.ResponseWriter, r *http.Request) {
	log.Println("Полученный GET-запрос:", r.URL)

	urls := h.storage.GetURLsByUser(h.user.id)
	if len(urls) == 0 {
		log.Println("Для пользователя с идентификатором '" + h.user.id + "' не найдены сохранённые URL")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	log.Println("Для пользователя с идентификатором '"+h.user.id+"' найдено ", len(urls), "сохранённых URL:")

	response := make(shortAndLongURLs, 0)
	for i, short := range urls {
		long, err := h.storage.FindURL(short)
		if err != nil {
			continue
		}

		record := shortAndLongURL{baseURL + short, long}
		log.Println("Запись", i, "короткий URL", record.ShortURL, "длинный URL", record.LongURL)
		response = append(response, record)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(response)
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

	sh, e := h.storage.AddURL(l, h.user.id)
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

	sh, e := h.storage.AddURL(requestBody.URL, h.user.id)
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
