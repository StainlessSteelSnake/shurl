package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"strconv"
	"time"
)

type batchURLs = [][2]string

type Storager interface {
	AddURL(string, string) (string, error)
	AddURLs(batchURLs, string) (batchURLs, error)
	FindURL(string) (string, error)
	GetURLsByUser(string) []string
	CloseFunc() func()
	Ping() error
}

type memoryStorage struct {
	container map[string]string
	usersURLs map[string][]string
}

func NewStorage(filePath string, database string, ctx context.Context) Storager {
	if database != "" {
		return newDBStorage(newMemoryStorage(), database, ctx)
	}

	if filePath != "" {
		return newFileStorage(newMemoryStorage(), filePath)
	}

	return newMemoryStorage()
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{map[string]string{}, map[string][]string{}}
}

func generateShortURL() (string, error) {
	t := time.Now()
	result := strconv.FormatInt(t.UnixMicro(), 36)

	b := make([]byte, 2)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	result = result + hex.EncodeToString(b)
	log.Println("Сгененирован короткий URL:", result)

	return result, nil
}

func (s *memoryStorage) AddURL(l, user string) (string, error) {
	sh, err := generateShortURL()
	if err != nil {
		return "", err
	}

	if _, ok := s.container[sh]; ok {
		return "", errors.New("короткий URL с ID " + string(sh) + " уже существует")
	}

	s.container[sh] = l
	s.usersURLs[user] = append(s.usersURLs[user], sh)
	return sh, nil
}

func (s *memoryStorage) AddURLs(longURLs batchURLs, user string) (batchURLs, error) {
	result := make(batchURLs, 0, len(longURLs))
	for _, longURL := range longURLs {
		id := longURL[0]
		l := longURL[1]

		sh, err := s.AddURL(l, user)
		if err != nil {
			return result[:0], err
		}

		result = append(result, [2]string{id, sh})
	}

	return result, nil
}

func (s *memoryStorage) FindURL(sh string) (string, error) {
	if l, ok := s.container[sh]; ok {
		return l, nil
	}

	return "", errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}

func (s *memoryStorage) GetURLsByUser(u string) []string {
	return s.usersURLs[u]
}

func (s *memoryStorage) CloseFunc() func() {
	return nil
}

func (s *memoryStorage) Ping() error {
	return errors.New("БД не была подключена, используется хранилище в памяти")
}