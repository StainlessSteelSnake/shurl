package handlers

import (
	"fmt"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"io"
	"net/http"
	"strings"
)

// postHandler Обработка входящих POST-запросов на создание короткого идентификтора для URL
func postHandler(storager storage.AddFinder, host string, w http.ResponseWriter, r *http.Request) {
	b, e := io.ReadAll(r.Body)
	if e != nil {
		fmt.Println("Неверный формат URL")
		http.Error(w, "неверный формат URL", http.StatusBadRequest)

		return
	}

	l := storage.LongURL(b)
	fmt.Println("Пришедший в запросе исходный URL:", l)

	sh, e := storager.Add(l)
	if e != nil {
		fmt.Println("Ошибка '", e, "' при добавлении в БД URL:", l)
		http.Error(w, "ошибка при добавлении в БД", http.StatusInternalServerError)
		return
	}
	fmt.Println("Созданный короткий идентификатор URL:", sh)

	w.WriteHeader(http.StatusCreated)
	_, e = w.Write([]byte("http://" + host + "/" + string(sh)))
	if e != nil {
		fmt.Println("Ошибка при записи ответа в тело запроса:", e)
	}
}

// getHandler Обработка входящих GET-запросов на получение исходного URL по его короткому идентификатору
func getHandler(storager storage.AddFinder, w http.ResponseWriter, r *http.Request) {
	fmt.Println("Полученный GET-запрос:", r.URL)

	sh := storage.ShortURL(strings.Trim(r.URL.Path, "/"))
	fmt.Println("Идентификатор короткого URL, полученный из GET-запроса:", sh)

	l, e := storager.Find(sh)
	if e != nil {
		fmt.Println("Ошибка '", e, "'. Не найден URL с указанным коротким идентификатором:", sh)
		http.Error(w, "URL с указанным коротким идентификатором не найден", http.StatusBadRequest)
		return
	}
	fmt.Println("Найден URL", l, "для короткого идентификатора", sh)

	//http.Redirect(w, r, string(l), http.StatusTemporaryRedirect)

	w.Header().Set("Location", string(l))
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// globalHandler Обработка всех входящих запросов, вне зависимости от метода
func GlobalHandler(storager storage.AddFinder, host string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			fmt.Println("Пришёл POST-запрос")
			postHandler(storager, host, w, r)
		case "GET":
			fmt.Println("Пришёл GET-запрос")
			getHandler(storager, w, r)
		default:
			fmt.Println("Пришёл неподдерживаемый запрос", r.Method)
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}
