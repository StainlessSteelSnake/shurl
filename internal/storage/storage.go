package storage

import (
	"errors"
	"strconv"
	"time"
)

type urlList = map[string]string

type Storage struct {
	container urlList
}

func NewStorage(list *urlList) *Storage {
	if list != nil && len(*list) > 0 {
		return &Storage{*list}
	}

	return &Storage{urlList{}}
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
