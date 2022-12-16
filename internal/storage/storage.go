package storage

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"time"
)

type Storage struct {
	container map[string]string
	file      *os.File
	decoder   *json.Decoder
	encoder   *json.Encoder
}

type Record struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

func NewStorage(filePath string) *Storage {
	storage := &Storage{map[string]string{}, nil, nil, nil}

	if filePath == "" {
		return storage
	}

	err := storage.openFile(filePath)
	if err != nil {
		log.Fatal(err)
		return storage
	}

	err = storage.loadFromFile()
	if err != nil {
		log.Println(err)
	}

	return storage
}

func (s *Storage) openFile(f string) error {
	var err error

	s.file, err = os.OpenFile(f, os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0777)
	if err != nil {
		return err
	}

	s.decoder = json.NewDecoder(s.file)
	s.encoder = json.NewEncoder(s.file)

	return nil
}

func (s *Storage) loadFromFile() error {
	if s.decoder == nil {
		return nil
	}

	r := Record{}
	for s.decoder.More() {
		err := s.decoder.Decode(&r)
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

func (s *Storage) saveToFile(r Record) error {
	if s.encoder == nil {
		return nil
	}

	err := s.encoder.Encode(r)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) CloseFile() {
	if s.file != nil {
		err := s.file.Close()
		if err != nil {
			log.Fatal(err)
		} else {
			log.Println("файл", s.file.Name(), "был успешно закрыт")
		}

	}
}

func (s *Storage) AddURL(l string) (string, error) {
	t := time.Now()
	sh := strconv.FormatInt(t.UnixMicro(), 36)

	if _, ok := s.container[sh]; ok {
		return "", errors.New("короткий URL с ID " + string(sh) + " уже существует")
	}

	s.container[sh] = l

	err := s.saveToFile(Record{sh, l})
	if err != nil {
		return sh, err
	}

	return sh, nil
}

func (s *Storage) FindURL(sh string) (string, error) {
	if l, ok := s.container[sh]; ok {
		return l, nil
	}

	return "", errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}
