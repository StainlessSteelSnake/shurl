package storage

import (
	"encoding/json"
	"log"
	"os"
)

type fileStorage struct {
	*MemoryStorage
	file    *os.File
	decoder *json.Decoder
	encoder *json.Encoder
}

// Record описывает структуру отдельной записи хранилища в файле.
type Record struct {
	ShortURL string `json:"short_url"`         // Короткий URL
	LongURL  string `json:"long_url"`          // Исходный длинный URL
	Deleted  bool   `json:"deleted,omitempty"` // Признак удаления записи
	UserID   string `json:"user_id"`           // Идентификатор пользователя, добавившего исходный длинный URL
}

func newFileStorage(m *MemoryStorage, filePath string) *fileStorage {
	storage := &fileStorage{m, nil, nil, nil}

	if filePath == "" {
		return storage
	}

	err := storage.openFile(filePath)
	if err != nil {
		log.Println(err)
		return storage
	}

	err = storage.loadFromFile()
	if err != nil {
		log.Println(err)
	}

	return storage
}

func (s *fileStorage) openFile(f string) error {
	var err error

	s.file, err = os.OpenFile(f, os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0777)
	if err != nil {
		return err
	}

	s.decoder = json.NewDecoder(s.file)
	s.encoder = json.NewEncoder(s.file)

	return nil
}

func (s *fileStorage) loadFromFile() error {
	if s.decoder == nil {
		return nil
	}

	r := new(Record)
	for s.decoder.More() {
		err := s.decoder.Decode(r)
		if err != nil {
			return err
		}
		if r.ShortURL == "" || r.LongURL == "" {
			continue
		}
		s.container[r.ShortURL] = MemoryRecord{LongURL: r.LongURL, Deleted: r.Deleted, User: r.UserID}

		if r.UserID == "" {
			continue
		}
		s.usersURLs[r.UserID] = append(s.usersURLs[r.UserID], r.ShortURL)
	}

	return nil
}

func (s *fileStorage) saveToFile(r *Record) error {
	if s.encoder == nil {
		return nil
	}

	err := s.encoder.Encode(r)
	if err != nil {
		return err
	}

	return nil
}

// AddURL добавляет исходный длинный URL в хранилище в файле, связывая его с созданным коротким URL.
func (s *fileStorage) AddURL(l, user string) (string, error) {
	sh, err := s.MemoryStorage.AddURL(l, user)
	if err != nil {
		return "", err
	}

	err = s.saveToFile(&Record{ShortURL: sh, LongURL: l, Deleted: false, UserID: user})
	if err != nil {
		return sh, err
	}

	return sh, nil
}

// AddURLs добавляет несколько исходных длинных URL в хранилище в файле, связывая их с соответствующими созданными короткими URL.
func (s *fileStorage) AddURLs(longURLs BatchURLs, user string) (BatchURLs, error) {
	result := make(BatchURLs, 0, len(longURLs))
	for _, longURL := range longURLs {
		sh, err := s.AddURL(longURL.URL, user)
		if err != nil {
			return result[:0], err
		}

		err = s.saveToFile(&Record{ShortURL: sh, LongURL: longURL.URL, Deleted: false, UserID: user})
		if err != nil {
			return result[:0], err
		}

		result = append(result, RecordURL{ID: longURL.ID, URL: sh})
	}

	return result, nil
}

// DeleteURLs добавляет заданные короткие URL в очередь на удаление из хранилища в файле.
func (s *fileStorage) DeleteURLs(shortURLs []string, user string) (deleted []string) {
	deleted = s.MemoryStorage.DeleteURLs(shortURLs, user)

	for _, sh := range deleted {
		err := s.saveToFile(&Record{ShortURL: sh, LongURL: s.container[sh].LongURL, Deleted: true, UserID: user})
		if err != nil {
			log.Println("Ошибка при записи удалённой ссылки с id", sh, "в файл")
		}
	}

	return deleted
}

// CloseFunc возвращает функцию для закрытия файла, используемого для хранения информации о коротких и длинных URL.
func (s *fileStorage) CloseFunc() func() {
	return func() {
		if s.file == nil {
			return
		}

		err := s.file.Close()
		if err != nil {
			log.Println(err)
			return
		}

		log.Println("файл", s.file.Name(), "был успешно закрыт")
	}
}
