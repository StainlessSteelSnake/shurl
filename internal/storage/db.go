package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"log"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
)

const txPreparedName = "shurl-insert"

type databaseStorage struct {
	*memoryStorage
	conn *pgx.Conn
	ctx  context.Context
}

type DBError struct {
	LongURL string
	Err     error
}

func newDBStorage(m *memoryStorage, database string, ctx context.Context) *databaseStorage {
	storage := &databaseStorage{m, nil, ctx}

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
		err = rows.Scan(&sh, &l, &u)
		if err != nil {
			log.Println("Ошибка чтения из БД:", err)
		}

		s.memoryStorage.container[sh] = l
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

	_, err = tx.Prepare(s.ctx, txPreparedName, queryInsert)
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

		_, err = tx.Exec(s.ctx, txPreparedName, sh, l, user)
		if err != nil {
			return result[:0], err
		}

		result = append(result, [2]string{id, sh})
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		return result[:0], err
	}

	return result, nil
}

func (s *databaseStorage) CloseFunc() func() {
	return func() {
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
