package storage

import (
	"encoding/json"
	"log"
	"os"
)

type fileStorage struct {
	*memoryStorage
	file    *os.File
	decoder *json.Decoder
	encoder *json.Encoder
}

type Record struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
	Deleted  bool   `json:"deleted,omitempty"`
	UserID   string `json:"user_id"`
}

func newFileStorage(m *memoryStorage, filePath string) *fileStorage {
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
		s.container[r.ShortURL] = memoryRecord{longURL: r.LongURL, deleted: r.Deleted, user: r.UserID}

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

func (s *fileStorage) AddURL(l, user string) (string, error) {
	sh, err := s.memoryStorage.AddURL(l, user)
	if err != nil {
		return "", err
	}

	err = s.saveToFile(&Record{ShortURL: sh, LongURL: l, Deleted: false, UserID: user})
	if err != nil {
		return sh, err
	}

	return sh, nil
}

func (s *fileStorage) AddURLs(longURLs batchURLs, user string) (batchURLs, error) {
	result := make(batchURLs, 0, len(longURLs))
	for _, longURL := range longURLs {
		id := longURL[0]
		l := longURL[1]

		sh, err := s.AddURL(l, user)
		if err != nil {
			return result[:0], err
		}

		err = s.saveToFile(&Record{ShortURL: sh, LongURL: l, Deleted: false, UserID: user})
		if err != nil {
			return result[:0], err
		}

		result = append(result, [2]string{id, sh})
	}

	return result, nil
}

func (s *fileStorage) DeleteURLs(shortURLs []string, user string) (deleted []string) {
	deleted = s.memoryStorage.DeleteURLs(shortURLs, user)

	for _, sh := range deleted {
		err := s.saveToFile(&Record{ShortURL: sh, LongURL: s.container[sh].longURL, Deleted: true, UserID: user})
		if err != nil {
			log.Println("Ошибка при записи удалённой ссылки с id", sh, "в файл")
		}
	}

	return deleted
}

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
