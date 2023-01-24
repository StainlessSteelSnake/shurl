package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"log"
	"sync"
)

const txPreparedInsert = "shurl-insert"
const txPreparedDelete = "shurl-delete"

type databaseStorage struct {
	*memoryStorage
	conn           *pgx.Conn
	ctx            context.Context
	deletionQueue  chan string
	deletionCancel context.CancelFunc
	errors         chan error
	locker         sync.Mutex
}

type DBError struct {
	LongURL string
	Err     error
}

const DeletionBatchSize = 20
const DeletionQueueSize = DeletionBatchSize * 2

func (s *databaseStorage) DeleteURLs(shortURLs []string, user string) (deleted []string) {
	deleted = make([]string, 0)

	go func() {
		deleted = s.memoryStorage.DeleteURLs(shortURLs, user)

		for _, sh := range deleted {
			s.deletionQueue <- sh
		}
	}()

	return deleted
}

func (s *databaseStorage) deletionQueueProcess(ctx context.Context) chan error {
	err := make(chan error)

	go func(s *databaseStorage, ctx context.Context) {
		deletionBatch := make([]string, DeletionBatchSize)
		for {
			select {
			case sh, ok := <-s.deletionQueue:
				if !ok {
					return
				}

				deletionBatch = append(deletionBatch, sh)

				if len(deletionBatch) >= DeletionBatchSize {
					s.delete(deletionBatch)
					deletionBatch = deletionBatch[:0]
				}

			case <-ctx.Done():
				return
			default:
				if len(deletionBatch) == 0 {
					continue
				}

				s.delete(deletionBatch)
				deletionBatch = deletionBatch[:0]
			}
		}
	}(s, ctx)

	return err
}

func (s *databaseStorage) errorProcess() {
	go func() {
		for {
			select {
			case err, ok := <-s.errors:
				if !ok {
					return
				}

				log.Println(err)
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

func (s *databaseStorage) delete(deletionBatch []string) {
	s.locker.Lock()
	defer s.locker.Unlock()

	tx, err := s.conn.Begin(s.ctx)
	if err != nil {
		s.errors <- err
		return
	}

	defer tx.Rollback(s.ctx)

	_, err = tx.Prepare(s.ctx, txPreparedDelete, queryDelete)
	if err != nil {
		s.errors <- err
		return
	}

	for _, sh := range deletionBatch {
		_, err = tx.Exec(s.ctx, txPreparedDelete, sh)
		if err != nil {
			s.errors <- err
			return
		}
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		s.errors <- err
		return
	}
}

func newDBStorage(m *memoryStorage, database string, ctx context.Context) *databaseStorage {
	storage := &databaseStorage{memoryStorage: m, conn: nil, ctx: ctx, deletionQueue: nil}

	var err error
	storage.conn, err = pgx.Connect(ctx, database)
	if err != nil {
		log.Println(err)
		return storage
	}

	err = storage.init()
	if err != nil {
		log.Fatal(err)
	}

	storage.deletionQueue = make(chan string, DeletionQueueSize)

	var deletionCtx context.Context
	deletionCtx, storage.deletionCancel = context.WithCancel(ctx)

	storage.errors = storage.deletionQueueProcess(deletionCtx)

	return storage
}

func (s *databaseStorage) init() error {

	_, err := s.conn.Exec(s.ctx, queryCreateTable)
	if err != nil {
		return err
	}

	rows, err := s.conn.Query(s.ctx, querySelectAll)
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var sh, l, u string
		var d bool
		err = rows.Scan(&sh, &l, &u, &d)
		if err != nil {
			log.Println("Ошибка чтения из БД:", err)
		}

		s.memoryStorage.container[sh] = memoryRecord{longURL: l, deleted: d, user: u}
		s.memoryStorage.usersURLs[u] = append(s.memoryStorage.usersURLs[u], sh)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	log.Println("Таблицы успешно инициализированы в БД")
	return nil
}

func (e *DBError) Error() string {
	return fmt.Sprintf("Найден дубликат для полного URL: %v. Ошибка добавления в БД: %v", e.LongURL, e.Err)
}

func NewStorageDBError(longURL string, err error) error {
	return &DBError{
		LongURL: longURL,
		Err:     err,
	}
}

func (s *databaseStorage) AddURL(l, user string) (string, error) {
	sh, err := s.memoryStorage.AddURL(l, user)
	if err != nil {
		return "", err
	}

	s.locker.Lock()
	defer s.locker.Unlock()

	ct, err := s.conn.Exec(s.ctx, queryInsert, sh, l, user)
	if err != nil {
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) {
			return "", err
		}

		log.Println("Ошибка операции с БД, код:", pgErr.Code, ", сообщение:", pgErr.Error())

		if pgErr.Code != pgerrcode.UniqueViolation {
			return "", err
		}

		duplicateErr := NewStorageDBError(l, err)

		r := s.conn.QueryRow(s.ctx, querySelectByLongURL, l)
		err = r.Scan(&sh)
		if err != nil {
			return "", NewStorageDBError(l, err)
		}

		log.Println("Найдена ранее сохранённая запись")
		return sh, duplicateErr
	}

	log.Println("Добавлено строк:", ct.RowsAffected())
	return sh, nil
}

func (s *databaseStorage) AddURLs(longURLs batchURLs, user string) (batchURLs, error) {
	result := make(batchURLs, 0, len(longURLs))

	tx, err := s.conn.Begin(s.ctx)
	if err != nil {
		return result[:0], err
	}

	defer tx.Rollback(s.ctx)

	_, err = tx.Prepare(s.ctx, txPreparedInsert, queryInsert)
	if err != nil {
		return result[:0], err
	}

	for _, longURL := range longURLs {
		id := longURL[0]
		l := longURL[1]

		sh, err := s.memoryStorage.AddURL(l, user)
		if err != nil {
			return result[:0], err
		}

		_, err = tx.Exec(s.ctx, txPreparedInsert, sh, l, user)
		if err != nil {
			return result[:0], err
		}

		result = append(result, [2]string{id, sh})
	}

	s.locker.Lock()
	defer s.locker.Unlock()

	err = tx.Commit(s.ctx)
	if err != nil {
		return result[:0], err
	}

	return result, nil
}

func (s *databaseStorage) CloseFunc() func() {
	return func() {
		s.deletionCancel()
		close(s.deletionQueue)
		close(s.errors)

		if s.conn == nil {
			return
		}

		err := s.conn.Close(s.ctx)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func (s *databaseStorage) Ping() error {
	if s.conn == nil {
		return s.memoryStorage.Ping()
	}
	return s.conn.Ping(s.ctx)
}
