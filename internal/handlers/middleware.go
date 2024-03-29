package handlers

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

// Write записывает переданный срез байт во внутреннюю переменную.
func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func gzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			log.Println("Клиент не принимает ответы в gzip")
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			log.Println("Ошибка при формировании ответа в gzip:", err)
			http.Error(w, "ошибка при формировании ответа в gzip: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() {
			if err := gz.Close(); err != nil {
				log.Println(err)
			}
		}()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
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
	defer func() {
		if err := reader.Close(); err != nil {
			log.Println(err)
		}
	}()

	return io.ReadAll(reader)
}
