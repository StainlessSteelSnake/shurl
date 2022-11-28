package storage

import (
	"errors"
	"strconv"
	"time"
)

type (
	ShortURL string               // короткая ссылка
	LongURL  string               // длинная ссылка
	URLList  map[ShortURL]LongURL // соответствие коротких ссылок длинным ссылкам
)

// storage Хранилище коротких и длинных ссылок
type storage struct {
	container URLList
}

// AddFinder Добавление и поиск ссылок в хранилище
type AddFinder interface {
	Add(LongURL) (ShortURL, error)  // Добавление длинной ссылки в хранилище и получение идентификатора короткой ссылки
	Find(ShortURL) (LongURL, error) // Поиск длинной ссылки по соответствующей короткой ссылке
}

// New Создание экземляра хранилища
func New(ul URLList) AddFinder {
	if ul == nil {
		return &storage{URLList{}}
	} else {
		return &storage{ul}
	}
}

// Add Добавление длинной ссылки в хранилище и получение идентификатора короткой ссылки
func (s *storage) Add(l LongURL) (ShortURL, error) {
	// Получение текущего штампа времени
	t := time.Now()

	// Формирование короткой ссылки путём преобразования числового значения штампа времени (в микросекундах)
	// из десятичного формата 36-ричный формат (десятичные цифры и буквы латинского алфавита)
	sh := ShortURL(strconv.FormatInt(t.UnixMicro(), 36))

	// Проверка наличия полученной короткой ссылки в хранилище
	if _, ok := s.container[sh]; ok {
		return "", errors.New("короткий URL с ID " + string(sh) + " уже существует")
	}

	// Добавление в хранилище соответствия между полученной короткой ссылкой и исходной длинной ссылкой
	s.container[sh] = l
	return sh, nil
}

// Find Поиск длинной ссылки по соответствующей короткой ссылке
func (s *storage) Find(sh ShortURL) (LongURL, error) {
	if l, ok := s.container[sh]; ok {
		return l, nil
	}

	return "", errors.New("короткий URL с ID \" + string(sh) + \" не существует")
}
