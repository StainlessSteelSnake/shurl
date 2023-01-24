package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/StainlessSteelSnake/shurl/internal/auth"

	"github.com/go-chi/chi/v5"
)

type Storager interface {
	AddURL(l, user string) (string, error)
	AddURLs([][2]string, string) ([][2]string, error)
	FindURL(sh string) (string, bool, error)
	GetURLsByUser(string) []string
	DeleteURLs([]string, string) []string
	Ping() error
}

type Handler struct {
	*chi.Mux
	storage Storager
	auth    auth.Authenticator
}

type PostRequestBody struct {
	URL string `json:"url"`
}

type PostResponseBody struct {
	Result string `json:"result"`
}

type PostRequestRecord struct {
	ID  string `json:"correlation_id"`
	URL string `json:"original_url"`
}

type PostResponseRecord struct {
	ID       string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

type DeleteRequestBody []string

type PostRequestBatch []PostRequestRecord

type PostResponseBatch []PostResponseRecord

var baseURL string

type shortAndLongURL struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"original_url"`
}

type shortAndLongURLs []shortAndLongURL

func NewHandler(s Storager, bURL string) *Handler {
	baseURL = bURL

	handler := &Handler{
		chi.NewMux(),
		s,
		auth.NewAuth(),
	}

	handler.Route("/", func(r chi.Router) {
		r.Get("/{id}", handler.auth.Authenticate(gzipHandler(handler.getLongURL)))
		r.Get("/api/user/urls", handler.auth.Authenticate(gzipHandler(handler.getLongURLsByUser)))
		r.Get("/ping", handler.ping)
		r.Post("/", handler.auth.Authenticate(gzipHandler(handler.postLongURL)))
		r.Post("/api/shorten", handler.auth.Authenticate(gzipHandler(handler.postLongURLinJSON)))
		r.Post("/api/shorten/batch", handler.auth.Authenticate(gzipHandler(handler.postLongURLinJSONbatch)))
		r.Delete("/api/user/urls", handler.auth.Authenticate(gzipHandler(handler.deleteURLs)))
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

	l, d, e := h.storage.FindURL(sh)
	if e != nil {
		log.Println("Ошибка '", e, "'. Не найден URL с указанным коротким идентификатором:", sh)
		http.Error(w, "URL с указанным коротким идентификатором не найден", http.StatusBadRequest)
		return
	}

	if d == true {
		log.Println("URL", l, "для короткого идентификатора", sh, "был удалён")
		w.WriteHeader(http.StatusGone)
		return
	}

	log.Println("Найден URL", l, "для короткого идентификатора", sh)
	w.Header().Set("Location", l)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *Handler) getLongURLsByUser(w http.ResponseWriter, r *http.Request) {
	log.Println("Полученный GET-запрос:", r.URL)

	urls := h.storage.GetURLsByUser(h.auth.GetUserID())
	if len(urls) == 0 {
		log.Println("Для пользователя с идентификатором '" + h.auth.GetUserID() + "' не найдены сохранённые URL")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	log.Println("Для пользователя с идентификатором '"+h.auth.GetUserID()+"' найдено ", len(urls), "сохранённых URL:")

	response := make(shortAndLongURLs, 0)
	for i, short := range urls {
		long, _, err := h.storage.FindURL(short)
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
	err := enc.Encode(response)
	if err != nil {
		http.Error(w, "не удалось закодировать в JSON список URL", http.StatusInternalServerError)
	}
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

	var duplicateFound bool
	sh, e := h.storage.AddURL(l, h.auth.GetUserID())
	if e != nil {
		if !strings.Contains(e.Error(), l) {
			log.Println("Ошибка '", e, "' при добавлении в БД URL:", l)
			http.Error(w, "ошибка при добавлении в БД: "+e.Error(), http.StatusInternalServerError)
			return
		}
		duplicateFound = true
	}

	if duplicateFound {
		w.WriteHeader(http.StatusConflict)
		log.Println("Найденный короткий идентификатор URL:", sh)
	} else {
		log.Println("Созданный короткий идентификатор URL:", sh)
		w.WriteHeader(http.StatusCreated)
	}

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

	var duplicateFound bool
	sh, e := h.storage.AddURL(requestBody.URL, h.auth.GetUserID())
	if e != nil {

		if !strings.Contains(e.Error(), requestBody.URL) {
			log.Println("Ошибка '", e, "' при добавлении в БД URL:", requestBody.URL)
			http.Error(w, "ошибка при добавлении в БД: "+e.Error(), http.StatusInternalServerError)
			return
		}
		duplicateFound = true
	}
	log.Println("Созданный короткий идентификатор URL:", sh)

	response, e := json.Marshal(PostResponseBody{baseURL + sh})
	if e != nil {
		log.Println("Ошибка '", e, "' при формировании ответа:", requestBody.URL)
		http.Error(w, "ошибка при при формировании ответа: "+e.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if duplicateFound {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	_, e = w.Write(response)
	if e != nil {
		log.Println("Ошибка при записи ответа в тело запроса:", e)
	}
}

func (h *Handler) ping(w http.ResponseWriter, r *http.Request) {
	err := h.storage.Ping()
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) postLongURLinJSONbatch(w http.ResponseWriter, r *http.Request) {
	b, e := decodeRequest(r)
	if e != nil {
		log.Println("Неверный формат данных в запросе:", e)
		http.Error(w, "неверный формат данных в запросе: "+e.Error(), http.StatusBadRequest)
		return
	}

	requestBody := PostRequestBatch{}
	e = json.Unmarshal(b, &requestBody)
	if e != nil {
		log.Println("Неверный формат данных в запросе:", e)
		http.Error(w, "неверный формат данных в запросе: "+e.Error(), http.StatusBadRequest)
		return
	}

	if len(requestBody) == 0 {
		log.Println("Пустой список URL")
		http.Error(w, "пустой список URL", http.StatusBadRequest)
		return
	}

	var longURLs = make([][2]string, 0, len(requestBody))
	for _, requestRecord := range requestBody {
		longURLs = append(longURLs, [2]string{requestRecord.ID, requestRecord.URL})
	}

	shortURLs, e := h.storage.AddURLs(longURLs, h.auth.GetUserID())
	if e != nil {
		log.Println("Ошибка '", e, "' при добавлении в БД URLs:", longURLs)
		http.Error(w, "ошибка при добавлении в БД URLs: "+e.Error(), http.StatusInternalServerError)
		return
	}

	var responseBody = make(PostResponseBatch, 0, len(shortURLs))
	for _, shortURL := range shortURLs {
		responseRecord := PostResponseRecord{ID: shortURL[0], ShortURL: baseURL + shortURL[1]}
		responseBody = append(responseBody, responseRecord)
	}

	response, e := json.Marshal(responseBody)
	if e != nil {
		log.Println("Ошибка '", e, "' при формировании ответа:", responseBody)
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

func (h *Handler) deleteURLs(w http.ResponseWriter, r *http.Request) {
	b, e := decodeRequest(r)
	if e != nil {
		log.Println("Неверный формат данных в запросе:", e)
		http.Error(w, "неверный формат данных в запросе: "+e.Error(), http.StatusBadRequest)
		return
	}

	requestBody := DeleteRequestBody{}
	e = json.Unmarshal(b, &requestBody)
	if e != nil {
		log.Println("Неверный формат данных в запросе:", e)
		http.Error(w, "неверный формат данных в запросе: "+e.Error(), http.StatusBadRequest)
		return
	}

	if len(requestBody) == 0 {
		log.Println("Пустой список идентификаторов URL")
		http.Error(w, "пустой список идентификаторов URL", http.StatusBadRequest)
		return
	}

	_ = h.storage.DeleteURLs(requestBody, h.auth.GetUserID())

	w.WriteHeader(http.StatusAccepted)
}
