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

const DeletionBatchSize = 20
const DeletionQueueSize = DeletionBatchSize * 2

type RecordURL struct {
	ID  string
	URL string
}

type BatchURLs = []RecordURL

type Storager interface {
	AddURL(string, string) (string, error)
	AddURLs(BatchURLs, string) (BatchURLs, error)
	FindURL(string) (MemoryRecord, error)
	GetURLsByUser(string) []string
	DeleteURLs([]string, string) []string
	CloseFunc() func()
	Ping() error
}

type deleter interface {
	deletionQueueProcess(context.Context)
	delete(context.Context, []string) error
}

type MemoryRecord struct {
	LongURL string
	User    string
	Deleted bool
}

type memoryStorage struct {
	container      map[string]MemoryRecord
	usersURLs      map[string][]string
	locker         sync.RWMutex
	deletionQueue  chan string
	deletionCancel context.CancelFunc
}

func NewStorage(ctx context.Context, filePath string, database string) Storager {
	var storage Storager

	deletionContext, deletionCancel := context.WithCancel(ctx)

	switch {
	case database != "":
		dStorage := newDBStorage(ctx, newMemoryStorage(), database)
		dStorage.deletionCancel = deletionCancel
		dStorage.deletionQueueProcess(deletionContext)
		storage = dStorage

	case filePath != "":
		fStorage := newFileStorage(newMemoryStorage(), filePath)
		fStorage.deletionCancel = deletionCancel
		fStorage.deletionQueueProcess(deletionContext)
		storage = fStorage

	default:
		mStorage := newMemoryStorage()
		mStorage.deletionCancel = deletionCancel
		mStorage.deletionQueueProcess(deletionContext)
		storage = mStorage
	}

	return storage
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{map[string]MemoryRecord{}, map[string][]string{}, sync.RWMutex{}, make(chan string, DeletionQueueSize), nil}
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

	s.container[sh] = MemoryRecord{LongURL: l, Deleted: false, User: user}
	s.usersURLs[user] = append(s.usersURLs[user], sh)
	return sh, nil
}

func (s *memoryStorage) AddURLs(longURLs BatchURLs, user string) (BatchURLs, error) {
	s.locker.Lock()
	defer s.locker.Unlock()

	result := make(BatchURLs, 0, len(longURLs))
	for _, longURL := range longURLs {
		sh, err := s.AddURL(longURL.URL, user)
		if err != nil {
			return result[:0], err
		}

		result = append(result, RecordURL{longURL.ID, sh})
	}

	return result, nil
}

func (s *memoryStorage) FindURL(sh string) (MemoryRecord, error) {
	s.locker.RLock()
	defer s.locker.RUnlock()

	result, ok := s.container[sh]
	if !ok {
		return MemoryRecord{"", "", false}, errors.New("короткий URL с ID \" + string(sh) + \" не существует")
	}

	return result, nil
}

func (s *memoryStorage) GetURLsByUser(u string) []string {
	s.locker.RLock()
	defer s.locker.RUnlock()

	return s.usersURLs[u]
}

func (s *memoryStorage) CloseFunc() func() {
	return nil
}

func (s *memoryStorage) Ping() error {
	return errors.New("БД не была подключена, используется хранилище в памяти")
}

func (s *memoryStorage) DeleteURLs(shortURLs []string, user string) (deleted []string) {
	go func() {
		s.locker.RLock()
		defer s.locker.RUnlock()

		for _, shortURL := range shortURLs {
			mr, ok := s.container[shortURL]
			if !ok {
				continue
			}

			if mr.User != user || mr.Deleted {
				continue
			}

			s.deletionQueue <- shortURL
		}
	}()

	return deleted
}

func (s *memoryStorage) delete(ctx context.Context, deletionBatch []string) error {
	s.locker.Lock()
	defer s.locker.Unlock()

	for _, shortURL := range deletionBatch {
		mr := s.container[shortURL]
		mr.Deleted = true
		s.container[shortURL] = mr
	}

	return nil
}

func (s *memoryStorage) deletionQueueProcess(ctx context.Context) {
	go deletionQueueProcess(ctx, s, s.deletionQueue)
}

func deletionQueueProcess(ctx context.Context, d deleter, deletionQueue <-chan string) {
	deletionBatch := make([]string, DeletionBatchSize)

	for {
		select {
		case sh, ok := <-deletionQueue:
			if !ok {
				return
			}

			deletionBatch = append(deletionBatch, sh)

			if len(deletionBatch) >= DeletionBatchSize {
				err := d.delete(ctx, deletionBatch)
				if err != nil {
					log.Println(err)
				}
				deletionBatch = deletionBatch[:0]
			}

		case <-ctx.Done():
			return
		default:
			if len(deletionBatch) == 0 {
				continue
			}

			err := d.delete(ctx, deletionBatch)
			if err != nil {
				log.Println(err)
			}
			deletionBatch = deletionBatch[:0]
		}
	}
}
