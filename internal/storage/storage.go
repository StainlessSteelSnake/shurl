package storage

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"time"
)

type Storager interface {
	CloseFunc() func()
	AddURL(string) (string, error)
	FindURL(sh string) (string, error)
}

type memoryStorage struct {
	container map[string]string
}

type fileStorage struct {
	*memoryStorage
	file    *os.File
	decoder *json.Decoder
	encoder *json.Encoder
}

type Record struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

func NewStorage(filePath string) Storager {
	if filePath == "" {
		return newMemoryStorage()
	}

	return newFileStorage(newMemoryStorage(), filePath)
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{map[string]string{}}
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

func (s *memoryStorage) FindURL(sh string) (string, error) {
	if l, ok := s.container[sh]; ok {
		return l, nil
	}

	return "", errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}

func (s *memoryStorage) AddURL(l string) (string, error) {
	t := time.Now()
	sh := strconv.FormatInt(t.UnixMicro(), 36)

	if _, ok := s.container[sh]; ok {
		return "", errors.New("короткий URL с ID " + string(sh) + " уже существует")
	}

	s.container[sh] = l
	return sh, nil
}

func (s *fileStorage) AddURL(l string) (string, error) {
	sh, err := s.memoryStorage.AddURL(l)
	if err != nil {
		return "", err
	}

	err = s.saveToFile(&Record{sh, l})
	if err != nil {
		return sh, err
	}

	return sh, nil
}

func (s *memoryStorage) CloseFunc() func() {
	return nil
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
		s.container[r.ShortURL] = r.LongURL
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
