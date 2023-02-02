package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/StainlessSteelSnake/shurl/internal/storage"

	"github.com/StainlessSteelSnake/shurl/internal/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	*chi.Mux
	storage storage.Storager
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

func NewHandler(s storage.Storager, bURL string) *Handler {
	baseURL = bURL
	log.Println("Base URL:", baseURL)

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

	l, d, err := h.storage.FindURL(sh)
	if err != nil {
		log.Println("Ошибка '", err, "'. Не найден URL с указанным коротким идентификатором:", sh)
		http.Error(w, "URL с указанным коротким идентификатором не найден", http.StatusBadRequest)
		return
	}

	if d {
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
	b, err := decodeRequest(r)
	if err != nil {
		log.Println("Неверный формат данных в запросе:", err)
		http.Error(w, "неверный формат данных в запросе: "+err.Error(), http.StatusBadRequest)
		return
	}

	l := string(b)
	log.Println("Пришедший в запросе исходный URL:", l)
	if len(l) == 0 {
		log.Println("Неверный формат URL")
		http.Error(w, "неверный формат URL", http.StatusBadRequest)

		return
	}

	sh, err := h.storage.AddURL(l, h.auth.GetUserID())
	if err != nil && !strings.Contains(err.Error(), l) {
		log.Println("Ошибка '", err, "' при добавлении в БД URL:", l)
		http.Error(w, "ошибка при добавлении в БД: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err != nil {
		log.Println("Найденный короткий идентификатор URL:", sh)
		w.WriteHeader(http.StatusConflict)
	} else {
		log.Println("Созданный короткий идентификатор URL:", sh)
		w.WriteHeader(http.StatusCreated)
	}

	_, err = w.Write([]byte(baseURL + sh))
	if err != nil {
		log.Println("Ошибка при записи ответа в тело запроса:", err)
	}
}

func (h *Handler) postLongURLinJSON(w http.ResponseWriter, r *http.Request) {
	b, err := decodeRequest(r)
	if err != nil {
		log.Println("Неверный формат данных в запросе:", err)
		http.Error(w, "неверный формат данных в запросе: "+err.Error(), http.StatusBadRequest)
		return
	}

	requestBody := PostRequestBody{}
	err = json.Unmarshal(b, &requestBody)
	if err != nil {
		log.Println("Неверный формат данных в запросе:", err)
		http.Error(w, "неверный формат данных в запросе: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("Пришедший в запросе исходный URL:", requestBody.URL)
	if len(requestBody.URL) == 0 {
		log.Println("Неверный формат URL")
		http.Error(w, "неверный формат URL", http.StatusBadRequest)
		return
	}

	var duplicateFound bool
	sh, err := h.storage.AddURL(requestBody.URL, h.auth.GetUserID())
	if err != nil && !strings.Contains(err.Error(), requestBody.URL) {
		log.Println("Ошибка '", err, "' при добавлении в БД URL:", requestBody.URL)
		http.Error(w, "ошибка при добавлении в БД: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err != nil {
		duplicateFound = true
	}

	log.Println("Созданный короткий идентификатор URL:", sh)

	response, err := json.Marshal(PostResponseBody{baseURL + sh})
	if err != nil {
		log.Println("Ошибка '", err, "' при формировании ответа:", requestBody.URL)
		http.Error(w, "ошибка при при формировании ответа: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if duplicateFound {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	_, err = w.Write(response)
	if err != nil {
		log.Println("Ошибка при записи ответа в тело запроса:", err)
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
	b, err := decodeRequest(r)
	if err != nil {
		log.Println("Неверный формат данных в запросе:", err)
		http.Error(w, "неверный формат данных в запросе: "+err.Error(), http.StatusBadRequest)
		return
	}

	requestBody := PostRequestBatch{}
	err = json.Unmarshal(b, &requestBody)
	if err != nil {
		log.Println("Неверный формат данных в запросе:", err)
		http.Error(w, "неверный формат данных в запросе: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(requestBody) == 0 {
		log.Println("Пустой список URL")
		http.Error(w, "пустой список URL", http.StatusBadRequest)
		return
	}

	var longURLs = make(storage.BatchURLs, 0, len(requestBody))
	for _, requestRecord := range requestBody {
		longURLs = append(longURLs, storage.RecordURL{ID: requestRecord.ID, URL: requestRecord.URL})
	}

	shortURLs, err := h.storage.AddURLs(longURLs, h.auth.GetUserID())
	if err != nil {
		log.Println("Ошибка '", err, "' при добавлении в БД URLs:", longURLs)
		http.Error(w, "ошибка при добавлении в БД URLs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var responseBody = make(PostResponseBatch, 0, len(shortURLs))
	for _, shortURL := range shortURLs {
		responseRecord := PostResponseRecord{ID: shortURL.ID, ShortURL: baseURL + shortURL.URL}
		responseBody = append(responseBody, responseRecord)
	}

	response, err := json.Marshal(responseBody)
	if err != nil {
		log.Println("Ошибка '", err, "' при формировании ответа:", responseBody)
		http.Error(w, "ошибка при при формировании ответа: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(response)
	if err != nil {
		log.Println("Ошибка при записи ответа в тело запроса:", err)
	}
}

func (h *Handler) deleteURLs(w http.ResponseWriter, r *http.Request) {
	log.Println("Обработка запроса на удаление данных")

	b, err := decodeRequest(r)
	if err != nil {
		log.Println("Неверный формат данных в запросе:", err)
		http.Error(w, "неверный формат данных в запросе: "+err.Error(), http.StatusBadRequest)
		return
	}

	requestBody := DeleteRequestBody{}
	err = json.Unmarshal(b, &requestBody)
	if err != nil {
		log.Println("Неверный формат данных в запросе:", err)
		http.Error(w, "неверный формат данных в запросе: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("Тело запроса на удаление данных:\n", requestBody)

	if len(requestBody) == 0 {
		log.Println("Пустой список идентификаторов URL")
		http.Error(w, "пустой список идентификаторов URL", http.StatusBadRequest)
		return
	}

	for i, r := range requestBody {
		requestBody[i] = strings.Replace(r, baseURL, "", -1)
	}

	log.Println("Список подлежащих удалению коротких идентификаторов URL:\n", requestBody)

	_ = h.storage.DeleteURLs(requestBody, h.auth.GetUserID())

	w.WriteHeader(http.StatusAccepted)
}
