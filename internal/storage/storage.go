package storage

import (
	"errors"
	"strconv"
	"time"
)

type URLList map[string]string

type storage struct {
	container URLList
}

type URLAddFinder interface {
	AddURL(string) (string, error)
	FindURL(string) (string, error)
}

func NewStorage(ul URLList) URLAddFinder {
	if ul != nil {
		return &storage{ul}
	}

	return &storage{URLList{}}
}

func (s *storage) AddURL(l string) (string, error) {
	t := time.Now()
	sh := strconv.FormatInt(t.UnixMicro(), 36)

	if _, ok := s.container[sh]; ok {
		return "", errors.New("короткий URL с ID " + string(sh) + " уже существует")
	}

	s.container[sh] = l
	return sh, nil
}

func (s *storage) FindURL(sh string) (string, error) {
	if l, ok := s.container[sh]; ok {
		return l, nil
	}

	return "", errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}
