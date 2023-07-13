// Пакет handlers содержит обработчики http-запросов к сервису.
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/StainlessSteelSnake/shurl/internal/auth"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
)

// Типы данных для обработчиков http-запросов.
type (
	// Handler содержит общие настройки и данные для обработки запросов: ссылку на маршрутизатор,
	// ссылку на хранилище данных и ссылку на обработчик авторизации пользователя.
	Handler struct {
		*chi.Mux
		storage storage.Storager
		auth    auth.Authenticator
	}

	// PostRequestBody содержит поля для обработки тела входящего POST-запроса в формате JSON.
	PostRequestBody struct {
		URL string `json:"url"`
	}

	// PostResponseBody содержит поля для формирования тела ответа в формате JSON на POST-запрос.
	PostResponseBody struct {
		Result string `json:"result"`
	}

	// PostRequestRecord содержит поля для обработки записи входящего
	// POST-запроса в формате JSON на массовую загрузку данных.
	PostRequestRecord struct {
		ID  string `json:"correlation_id"`
		URL string `json:"original_url"`
	}

	// PostResponseRecord содержит поля для обработки записи возвращаемого тела ответа
	// на POST-запрос в формате JSON на массовую загрузку данных.
	PostResponseRecord struct {
		ID       string `json:"correlation_id"`
		ShortURL string `json:"short_url"`
	}

	// DeleteRequestBody содержит список записей из тела запроса на удаление данных.
	DeleteRequestBody []string

	// PostRequestBatch содержит список записей из тела запроса на массовую загрузку данных.
	PostRequestBatch []PostRequestRecord

	// PostResponseBatch содержит список записей из тела ответа на запрос массовой загрузки данных.
	PostResponseBatch []PostResponseRecord

	shortAndLongURL struct {
		ShortURL string `json:"short_url"`
		LongURL  string `json:"original_url"`
	}

	shortAndLongURLs []shortAndLongURL
)

var baseURL string

// NewHandler создаёт верхнеуровневый обработчик HTTP-запросов.
// А также связывает его с хранилищем данных и обработчиком данных авторизации,
// выстраивает цепочки обработки для разных типов запросов и запрашиваемых путей.
func NewHandler(s storage.Storager, bURL string) *Handler {
	baseURL = bURL
	log.Println("Base URL:", baseURL)

	handler := &Handler{
		chi.NewMux(),
		s,
		auth.NewAuth(),
	}

	handler.Route("/", func(r chi.Router) {
		handler.Use(handler.auth.Authenticate)
		handler.Use(gzipHandler)

		r.Get("/{id}", handler.getLongURL)
		r.Get("/api/user/urls", handler.getLongURLsByUser)
		r.Get("/ping", handler.ping)
		r.Post("/", handler.postLongURL)
		r.Post("/api/shorten", handler.postLongURLinJSON)
		r.Post("/api/shorten/batch", handler.postLongURLinJSONbatch)
		r.Delete("/api/user/urls", handler.deleteURLs)
		r.MethodNotAllowed(handler.badRequest)
	})

	handler.Mux.Handle("/debug/pprof/*", http.DefaultServeMux)

	return handler
}

func (h *Handler) badRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "неподдерживаемый запрос: '"+r.RequestURI+"'", http.StatusBadRequest)
}

func (h *Handler) getLongURL(w http.ResponseWriter, r *http.Request) {
	log.Println("Полученный GET-запрос:", r.URL)

	shortURL := strings.Trim(r.URL.Path, "/")
	log.Println("Идентификатор короткого URL, полученный из GET-запроса:", shortURL)

	result, err := h.storage.FindURL(shortURL)
	if err != nil {
		log.Println("Ошибка '", err, "'. Не найден URL с указанным коротким идентификатором:", shortURL)
		http.Error(w, "URL с указанным коротким идентификатором не найден", http.StatusBadRequest)
		return
	}

	if result.Deleted {
		log.Println("URL", result.LongURL, "для короткого идентификатора", shortURL, "был удалён")
		w.WriteHeader(http.StatusGone)
		return
	}

	log.Println("Найден URL", result.LongURL, "для короткого идентификатора", shortURL)
	w.Header().Set("Location", result.LongURL)
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
	for i, shortURL := range urls {
		result, err := h.storage.FindURL(shortURL)
		if err != nil {
			continue
		}

		record := shortAndLongURL{baseURL + shortURL, result.LongURL}
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

	longURL := string(b)
	log.Println("Пришедший в запросе исходный URL:", longURL)
	if len(longURL) == 0 {
		log.Println("Неверный формат URL")
		http.Error(w, "неверный формат URL", http.StatusBadRequest)

		return
	}

	shortURL, err := h.storage.AddURL(longURL, h.auth.GetUserID())
	if err != nil && errors.Is(err, storage.DBError{LongURL: longURL, Duplicate: false, Err: nil}) {
		log.Println("Ошибка '", err, "' при добавлении в БД URL:", longURL)
		http.Error(w, "ошибка при добавлении в БД: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err != nil && errors.Is(err, storage.DBError{LongURL: longURL, Duplicate: true, Err: nil}) {
		log.Println("Найденный короткий идентификатор URL:", shortURL)
		w.WriteHeader(http.StatusConflict)
		err = nil
	}

	if err != nil {
		log.Println("Ошибка '", err, "' при добавлении в БД URL:", longURL)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		log.Println("Созданный короткий идентификатор URL:", shortURL)
		w.WriteHeader(http.StatusCreated)
	}

	_, err = w.Write([]byte(baseURL + shortURL))
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
	shortURL, err := h.storage.AddURL(requestBody.URL, h.auth.GetUserID())
	if err != nil && errors.Is(err, storage.DBError{LongURL: requestBody.URL, Err: nil}) {
		log.Println("Ошибка '", err, "' при добавлении в БД URL:", requestBody.URL)
		http.Error(w, "ошибка при добавлении в БД: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err != nil {
		duplicateFound = true
	}

	log.Println("Созданный короткий идентификатор URL:", shortURL)

	response, err := json.Marshal(PostResponseBody{baseURL + shortURL})
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

	for i, record := range requestBody {
		requestBody[i] = strings.Replace(record, baseURL, "", -1)
	}

	log.Println("Список подлежащих удалению коротких идентификаторов URL:\n", requestBody)

	_ = h.storage.DeleteURLs(requestBody, h.auth.GetUserID())

	w.WriteHeader(http.StatusAccepted)
}
