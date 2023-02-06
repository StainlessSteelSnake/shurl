package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"log"
)

const txPreparedInsert = "shurl-insert"
const txPreparedDelete = "shurl-delete"

type databaseStorage struct {
	*memoryStorage
	conn *pgx.Conn
}

type DBError struct {
	LongURL string
	Err     error
}

func (s *databaseStorage) deletionQueueProcess(ctx context.Context) {
	go deletionQueueProcess(ctx, s, s.memoryStorage.deletionQueue)
}

func (s *databaseStorage) delete(ctx context.Context, deletionBatch []string) error {
	s.locker.Lock()
	defer s.locker.Unlock()

	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	_, err = tx.Prepare(ctx, txPreparedDelete, queryDelete)
	if err != nil {
		return err
	}

	for _, sh := range deletionBatch {
		_, err = tx.Exec(ctx, txPreparedDelete, sh)
		if err != nil {
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return s.memoryStorage.delete(ctx, deletionBatch)
}

func newDBStorage(ctx context.Context, m *memoryStorage, database string) *databaseStorage {
	storage := &databaseStorage{memoryStorage: m, conn: nil}

	var err error
	storage.conn, err = pgx.Connect(ctx, database)
	if err != nil {
		log.Println(err)
		return storage
	}

	err = storage.init(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return storage
}

func (s *databaseStorage) init(ctx context.Context) error {

	_, err := s.conn.Exec(ctx, queryCreateTable)
	if err != nil {
		return err
	}

	rows, err := s.conn.Query(ctx, querySelectAll)
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

		s.memoryStorage.container[sh] = MemoryRecord{LongURL: l, Deleted: d, User: u}
		s.memoryStorage.usersURLs[u] = append(s.memoryStorage.usersURLs[u], sh)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	log.Println("Таблицы успешно инициализированы в БД")
	return nil
}

func (e DBError) Error() string {
	return fmt.Sprintf("Найден дубликат для полного URL: %v. Ошибка добавления в БД: %v", e.LongURL, e.Err)
}

func (e DBError) Is(target error) bool {
	err, ok := target.(DBError)
	if !ok {
		return false
	}

	if err.LongURL != e.LongURL {
		return false
	}

	return true
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

	ctx := context.Background()
	var pgErr *pgconn.PgError
	ct, err := s.conn.Exec(ctx, queryInsert, sh, l, user)
	if err != nil && !errors.As(err, &pgErr) {
		return "", err
	}

	if err != nil && pgErr.Code != pgerrcode.UniqueViolation {
		log.Println("Ошибка операции с БД, код:", pgErr.Code, ", сообщение:", pgErr.Error())
		return "", err
	}

	if err != nil {
		log.Println("Ошибка операции с БД, код:", pgErr.Code, ", сообщение:", pgErr.Error())
		duplicateErr := NewStorageDBError(l, err)

		r := s.conn.QueryRow(ctx, querySelectByLongURL, l)
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

func (s *databaseStorage) AddURLs(longURLs BatchURLs, user string) (BatchURLs, error) {
	result := make(BatchURLs, 0, len(longURLs))

	ctx := context.Background()
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return result[:0], err
	}

	defer tx.Rollback(ctx)

	_, err = tx.Prepare(ctx, txPreparedInsert, queryInsert)
	if err != nil {
		return result[:0], err
	}

	for _, longURL := range longURLs {
		sh, err := s.memoryStorage.AddURL(longURL.URL, user)
		if err != nil {
			return result[:0], err
		}

		_, err = tx.Exec(ctx, txPreparedInsert, sh, longURL.URL, user)
		if err != nil {
			return result[:0], err
		}

		result = append(result, RecordURL{ID: longURL.ID, URL: sh})
	}

	s.locker.Lock()
	defer s.locker.Unlock()

	err = tx.Commit(ctx)
	if err != nil {
		return result[:0], err
	}

	return result, nil
}

func (s *databaseStorage) CloseFunc() func() {
	return func() {
		s.deletionCancel()
		close(s.deletionQueue)
		//close(s.errors)

		if s.conn == nil {
			return
		}

		ctx := context.Background()
		err := s.conn.Close(ctx)
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

	ctx := context.Background()
	return s.conn.Ping(ctx)
}
