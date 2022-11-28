package handlers

import (
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"io"
	"log"
	"net/http"
	"strings"
)

// postHandler Обработка входящих POST-запросов на создание короткого идентификтора для URL
func postHandler(s storage.AddFinder, host string, w http.ResponseWriter, r *http.Request) {
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
	sh, e := s.Add(l)
	if e != nil {
		log.Println("Ошибка '", e, "' при добавлении в БД URL:", l)
		http.Error(w, "ошибка при добавлении в БД", http.StatusInternalServerError)
		return
	}
	log.Println("Созданный короткий идентификатор URL:", sh)

	// Формирование ответа: установка кода состояния "успешно создано"
	w.WriteHeader(http.StatusCreated)
	_, e = w.Write([]byte("http://" + host + "/" + string(sh)))
	if e != nil {
		log.Println("Ошибка при записи ответа в тело запроса:", e)
	}
}

// getHandler Обработка входящих GET-запросов на получение исходного URL по его короткому идентификатору
func getHandler(s storage.AddFinder, w http.ResponseWriter, r *http.Request) {
	log.Println("Полученный GET-запрос:", r.URL)

	// Получение идентификатора короткой ссылки из URL запроса
	sh := storage.ShortURL(strings.Trim(r.URL.Path, "/"))
	log.Println("Идентификатор короткого URL, полученный из GET-запроса:", sh)

	// Поиск длинной ссылки в хранилище по идентификатору короткой ссылки
	l, e := s.Find(sh)
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
func GlobalHandler(s storage.AddFinder, host string) func(w http.ResponseWriter, r *http.Request) {
	// Возвращаем функцию-обработчик для использования сервером
	return func(w http.ResponseWriter, r *http.Request) {
		// Определяем метод HTTP и в зависимости от него передаём запрос более узкоспециализированному обработчику
		switch r.Method {
		case "POST": // POST-запрос на формирование короткой ссылки
			log.Println("Пришёл POST-запрос")
			postHandler(s, host, w, r)
		case "GET": // GET-запрос на получение длинной ссылки по переданной короткой ссылке
			log.Println("Пришёл GET-запрос")
			getHandler(s, w, r)
		default: // Другие методы, помимо POST и GET, не поддерживаются
			// Формирование ответа: установка кода состояния "неправильный запрос"
			log.Println("Пришёл неподдерживаемый запрос", r.Method)
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}
