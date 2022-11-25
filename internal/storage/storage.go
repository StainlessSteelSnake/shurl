package storage

import (
	"errors"
	"strconv"
	"time"
)

type (
	ShortURL string
	LongURL  string
	URLList  map[ShortURL]LongURL
)

type storage struct {
	container URLList
}

type AddFinder interface {
	Add(LongURL) (ShortURL, error)
	Find(ShortURL) (LongURL, error)
}

func New() AddFinder {
	return &storage{URLList{}}
}

// Add Создание короткого идентификатора для URL
func (s *storage) Add(l LongURL) (ShortURL, error) {
	t := time.Now()

	sh := ShortURL(strconv.FormatInt(t.UnixMicro(), 36))

	if _, ok := s.container[sh]; ok {
		return "", errors.New("короткий URL с ID " + string(sh) + " уже существует")
	}

	s.container[sh] = l
	return sh, nil
}

// Find Поиск полного URL по его короткому идентификатору
func (s *storage) Find(sh ShortURL) (LongURL, error) {
	if l, ok := s.container[sh]; ok {
		return l, nil
	}

	return "", errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}
