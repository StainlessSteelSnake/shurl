// Пакет storage отвечает за хранилище данных и различные способы его организации.
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

// Константы для обработки очередей на удаление записей.
const (
	// DeletionBatchSize задаёт максимальный размер пакета для массового удаления данных.
	DeletionBatchSize = 20
	// DeletionQueueSize задаёт максимальный размер очереди записей, подлежащих удалению.
	DeletionQueueSize = DeletionBatchSize * 2
)

// Типы данных для работы хранилища.
type (
	// RecordURL содержит запись для списка массового сокращения длинных URL.
	RecordURL struct {
		ID  string // Идентификатор записи в исходном запросе
		URL string // Длинный URL, который подлежит сокращению
	}

	// BatchURLs содержит список URL, подлежащих сокращению
	BatchURLs = []RecordURL

	// Storager обеспечивает экземпляр хранилища основными функциями.
	Storager interface {
		AddURL(string, string) (string, error)        // Добавление длинного URL в хранилище и его сокращение.
		AddURLs(BatchURLs, string) (BatchURLs, error) // Добавление списка длинных URL в хранилище и их сокращение.
		FindURL(string) (MemoryRecord, error)         // Поиск длинного URL в хранилище по его сокращённому варианту.
		GetURLsByUser(string) []string                // Поиск в хранилище всех URL, добавленных текущим пользователем.
		DeleteURLs([]string, string) []string         // Удаление из хранилища списка URL.
		CloseFunc() func()                            // Закрытие соединения с хранилищем (для файла или БД).
		Ping() error                                  // Проверка установки соединения с БД.
	}

	deleter interface {
		DeletionQueueProcess(context.Context)
		delete(context.Context, []string) error
	}

	// MemoryRecord содержит соответствие исходного длинного URL и пользователя, добавившего его.
	// А также пометку об удаление этого URL из хранилища.
	MemoryRecord struct {
		LongURL string
		User    string
		Deleted bool
	}

	// MemoryStorage обеспечивает хранилище в памяти для соответствий исходных длинных URL и соответствующих им коротких URL.
	// А также хранит информацию об URL, добавленных определёнными пользователми,
	// обеспечивает блокировку хранилища при конкурентном доступе,
	// содержит ссылку на очередь для удаления записей и функцию для отмены контекста операций удаления.
	MemoryStorage struct {
		container      map[string]MemoryRecord
		usersURLs      map[string][]string
		locker         sync.RWMutex
		deletionQueue  chan string
		DeletionCancel context.CancelFunc
	}
)

// NewStorage создаёт реализацию хранилища в памяти, в файле или в БД, в зависимости от переданных настроек.
func NewStorage(ctx context.Context, filePath string, database string) Storager {
	var storage Storager

	deletionContext, deletionCancel := context.WithCancel(ctx)

	switch {
	case database != "":
		dStorage := NewDBStorage(ctx, NewMemoryStorage(), database)
		dStorage.DeletionCancel = deletionCancel
		dStorage.DeletionQueueProcess(deletionContext)
		storage = dStorage

	case filePath != "":
		fStorage := newFileStorage(NewMemoryStorage(), filePath)
		fStorage.DeletionCancel = deletionCancel
		fStorage.DeletionQueueProcess(deletionContext)
		storage = fStorage

	default:
		mStorage := NewMemoryStorage()
		mStorage.DeletionCancel = deletionCancel
		mStorage.DeletionQueueProcess(deletionContext)
		storage = mStorage
	}

	return storage
}

// NewMemoryStorage создаёт реализацию хранилища в памяти приложения.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		container:      map[string]MemoryRecord{},
		usersURLs:      map[string][]string{},
		deletionQueue:  make(chan string, DeletionQueueSize),
		DeletionCancel: nil,
	}
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

// AddURL добавляет исходный длинный URL в хранилище в памяти, связывая его с созданным коротким URL.
func (s *MemoryStorage) AddURL(l, user string) (string, error) {
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

// AddURLs добавляет несколько исходных длинных URL в хранилище в памяти, связывая их с соответствующими созданными короткими URL.
func (s *MemoryStorage) AddURLs(longURLs BatchURLs, user string) (BatchURLs, error) {
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

// FindURL ищет в хранилище в памяти исходный длинный URL по заданному короткому URL.
func (s *MemoryStorage) FindURL(sh string) (MemoryRecord, error) {
	s.locker.RLock()
	defer s.locker.RUnlock()

	result, ok := s.container[sh]
	if !ok {
		return MemoryRecord{"", "", false}, errors.New("короткий URL с ID \" + string(sh) + \" не существует")
	}

	return result, nil
}

// GetURLsByUser ищет в хранилище в памяти исходные длинные URL по заданному идентификатору пользователя, добавившего их.
func (s *MemoryStorage) GetURLsByUser(u string) []string {
	s.locker.RLock()
	defer s.locker.RUnlock()

	return s.usersURLs[u]
}

// CloseFunc не возвращает никакую функцию, поскольку соединение с БД не устанавливается для хранилища в памяти.
func (s *MemoryStorage) CloseFunc() func() {
	return nil
}

// Ping возвращает сообщение об ошибке, поскольку соединение с БД не устанавливается для хранилища в памяти.
func (s *MemoryStorage) Ping() error {
	return errors.New("БД не была подключена, используется хранилище в памяти")
}

// DeleteURLs добавляет заданные короткие URL в очередь на удаление из хранилища в памяти.
func (s *MemoryStorage) DeleteURLs(shortURLs []string, user string) (deleted []string) {
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

func (s *MemoryStorage) delete(ctx context.Context, deletionBatch []string) error {
	//s.locker.Lock()
	//defer s.locker.Unlock()

	for _, shortURL := range deletionBatch {
		mr := s.container[shortURL]
		mr.Deleted = true
		s.container[shortURL] = mr
	}

	return nil
}

// DeletionQueueProcess обрабатывает очередь запросов на удаление, вызывая обработчик каждой записи в отдельном потоке.
func (s *MemoryStorage) DeletionQueueProcess(ctx context.Context) {
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
