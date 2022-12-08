package storage

import (
	"errors"
	"strconv"
	"time"
)

type Storager struct {
	container map[string]string
}

func NewStorage() *Storager {
	list := map[string]string{}
	return &Storager{list}
}

func (s *Storager) AddURL(l string) (string, error) {
	t := time.Now()
	sh := strconv.FormatInt(t.UnixMicro(), 36)

	if _, ok := s.container[sh]; ok {
		return "", errors.New("короткий URL с ID " + string(sh) + " уже существует")
	}

	s.container[sh] = l
	return sh, nil
}

func (s *Storager) FindURL(sh string) (string, error) {
	if l, ok := s.container[sh]; ok {
		return l, nil
	}

	return "", errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}
