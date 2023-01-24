package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"
)

type batchURLs = [][2]string

type Storager interface {
	AddURL(string, string) (string, error)
	AddURLs(batchURLs, string) (batchURLs, error)
	FindURL(string) (string, bool, error)
	GetURLsByUser(string) []string
	DeleteURLs([]string, string) []string
	CloseFunc() func()
	Ping() error
}

type memoryRecord struct {
	longURL string
	user    string
	deleted bool
}

type memoryStorage struct {
	container map[string]memoryRecord
	usersURLs map[string][]string
	locker    sync.RWMutex
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
	return &memoryStorage{map[string]memoryRecord{}, map[string][]string{}, sync.RWMutex{}}
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
	s.locker.Lock()
	defer s.locker.Unlock()

	sh, err := generateShortURL()
	if err != nil {
		return "", err
	}

	if _, ok := s.container[sh]; ok {
		return "", errors.New("короткий URL с ID " + string(sh) + " уже существует")
	}

	s.container[sh] = memoryRecord{longURL: l, deleted: false, user: user}
	s.usersURLs[user] = append(s.usersURLs[user], sh)
	return sh, nil
}

func (s *memoryStorage) AddURLs(longURLs batchURLs, user string) (batchURLs, error) {
	s.locker.Lock()
	defer s.locker.Unlock()

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

func (s *memoryStorage) FindURL(sh string) (string, bool, error) {
	s.locker.RLock()
	defer s.locker.RUnlock()

	r, ok := s.container[sh]
	if !ok {
		return "", false, errors.New("короткий URL с ID \" + string(sh) + \" не существует")
	}

	if r.deleted {
		return "", r.deleted, nil
	}

	return r.longURL, r.deleted, nil
}

func (s *memoryStorage) GetURLsByUser(u string) []string {
	s.locker.RLock()
	defer s.locker.RUnlock()

	return s.usersURLs[u]
}

func (s *memoryStorage) DeleteURLs(shortURLs []string, user string) (deleted []string) {
	deleted = make([]string, 0)

	s.locker.Lock()
	defer s.locker.Unlock()

	for _, sh := range shortURLs {
		mr, ok := s.container[sh]
		if !ok {
			continue
		}

		if mr.user != user {
			continue
		}

		s.container[sh] = memoryRecord{longURL: mr.longURL, user: mr.user, deleted: true}
		deleted = append(deleted, sh)
	}

	return deleted
}

func (s *memoryStorage) CloseFunc() func() {
	return nil
}

func (s *memoryStorage) Ping() error {
	return errors.New("БД не была подключена, используется хранилище в памяти")
}
