package storage

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const txPreparedInsert = "shurl-insert"
const txPreparedDelete = "shurl-delete"

type DatabaseStorage struct {
	*MemoryStorage
	conn *pgx.Conn
}

type DBError struct {
	LongURL   string
	Duplicate bool
	Err       error
}

func (s *DatabaseStorage) DeletionQueueProcess(ctx context.Context) {
	go deletionQueueProcess(ctx, s, s.MemoryStorage.deletionQueue)
}

func (s *DatabaseStorage) delete(ctx context.Context, deletionBatch []string) error {
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

	return s.MemoryStorage.delete(ctx, deletionBatch)
}

func NewDBStorage(ctx context.Context, m *MemoryStorage, database string) *DatabaseStorage {
	storage := &DatabaseStorage{MemoryStorage: m, conn: nil}

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

func (s *DatabaseStorage) init(ctx context.Context) error {

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

		s.MemoryStorage.container[sh] = MemoryRecord{LongURL: l, Deleted: d, User: u}
		s.MemoryStorage.usersURLs[u] = append(s.MemoryStorage.usersURLs[u], sh)
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

	if err.LongURL != e.LongURL || err.Duplicate != e.Duplicate {
		return false
	}

	return true
}

func NewStorageDBError(longURL string, duplicate bool, err error) error {
	return &DBError{
		LongURL:   longURL,
		Duplicate: duplicate,
		Err:       err,
	}
}

func (s *DatabaseStorage) AddURL(l, user string) (string, error) {

	sh, err := s.MemoryStorage.AddURL(l, user)
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
		duplicateErr := NewStorageDBError(l, true, err)

		r := s.conn.QueryRow(ctx, querySelectByLongURL, l)
		err = r.Scan(&sh)
		if err != nil {
			return "", NewStorageDBError(l, false, err)
		}

		log.Println("Найдена ранее сохранённая запись")
		return sh, duplicateErr
	}

	log.Println("Добавлено строк:", ct.RowsAffected())
	return sh, nil
}

func (s *DatabaseStorage) AddURLs(longURLs BatchURLs, user string) (BatchURLs, error) {
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
		sh, err := s.MemoryStorage.AddURL(longURL.URL, user)
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

func (s *DatabaseStorage) CloseFunc() func() {
	return func() {
		s.DeletionCancel()
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

func (s *DatabaseStorage) Ping() error {
	if s.conn == nil {
		return s.MemoryStorage.Ping()
	}

	ctx := context.Background()
	return s.conn.Ping(ctx)
}
