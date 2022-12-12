package storage

import (
	"errors"
	"strconv"
	"time"
)

type Storage struct {
	container map[string]string
}

func NewStorage() *Storage {
	list := map[string]string{}
	return &Storage{list}
}

func (s *Storage) AddURL(l string) (string, error) {
	t := time.Now()
	sh := strconv.FormatInt(t.UnixMicro(), 36)

	if _, ok := s.container[sh]; ok {
		return "", errors.New("короткий URL с ID " + string(sh) + " уже существует")
	}

	s.container[sh] = l
	return sh, nil
}

func (s *Storage) FindURL(sh string) (string, error) {
	if l, ok := s.container[sh]; ok {
		return l, nil
	}

	return "", errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}
