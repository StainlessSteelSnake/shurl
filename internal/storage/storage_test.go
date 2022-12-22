package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStorage(t *testing.T) {
	t.Skip()
}

func Test_fileStorage_CloseFunc(t *testing.T) {
	t.Skip()
}

func Test_fileStorage_loadFromFile(t *testing.T) {
	t.Skip()
}

func Test_fileStorage_openFile(t *testing.T) {
	t.Skip()
}

func Test_fileStorage_saveToFile(t *testing.T) {
	t.Skip()
}

func Test_memoryStorage_CloseFunc(t *testing.T) {
	t.Skip()
}

func Test_newMemoryStorage(t *testing.T) {
	tests := []struct {
		name string
		want *memoryStorage
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, newMemoryStorage(), "newMemoryStorage()")
		})
	}
}

func Test_newFileStorage(t *testing.T) {
	type args struct {
		m        *memoryStorage
		filePath string
	}
	tests := []struct {
		name string
		args args
		want *fileStorage
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, newFileStorage(tt.args.m, tt.args.filePath), "newFileStorage(%v, %v)", tt.args.m, tt.args.filePath)
		})
	}
}

func Test_memoryStorage_AddURL(t *testing.T) {
	tests := []struct {
		name       string
		s          memoryStorage
		URL        string
		iterations int
		err        error
	}{
		{
			"Успешное добавление 1 элемента",
			memoryStorage{map[string]string{}},
			"http://ya.ru",
			1,
			nil,
		},
		{
			"Успешное добавление дублирующих элементов",
			memoryStorage{map[string]string{}},
			"http://ya.ru",
			3,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 1; i++ {
				sh, err := tt.s.AddURL(tt.URL)
				assert.NoError(t, err)
				assert.NotEmpty(t, sh)
			}
		})
	}
}

func Test_memoryStorage_FindURL(t *testing.T) {
	tests := []struct {
		name    string
		s       memoryStorage
		URL     string
		wantURL string
		OK      bool
	}{
		{
			"Неуспешная попытка поиска в пустом хранилище",
			memoryStorage{map[string]string{}},
			"dummy",
			"",
			false,
		},
		{
			"Успешная попытка поиска в списке из 1 элемента",
			memoryStorage{map[string]string{"dummy": "http://ya.ru"}},
			"dummy",
			"http://ya.ru",
			true,
		},
		{
			"Успешная попытка поиска в списке из 3 элементов",
			memoryStorage{map[string]string{"dummy": "http://ya.ru", "dummy1": "http://mail.ru", "dummy2": "http://google.ru"}},
			"dummy1",
			"http://mail.ru",
			true,
		},
		{
			"Неуспешная попытка поиска в непустом списке",
			memoryStorage{map[string]string{"dummy": "http://ya.ru"}},
			"dummy1",
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := tt.s.FindURL(tt.URL)
			assert.Equal(t, tt.OK, err == nil)
			assert.Equal(t, tt.wantURL, l)
		})
	}
}

func Test_fileStorage_AddURL(t *testing.T) {
	tests := []struct {
		name    string
		s       fileStorage
		URL     string
		wantURL string
		OK      bool
	}{
		{
			"Неуспешная попытка поиска в пустом хранилище",
			fileStorage{&memoryStorage{map[string]string{}}, nil, nil, nil},
			"dummy",
			"",
			false,
		},
		{
			"Успешная попытка поиска в списке из 1 элемента",
			fileStorage{&memoryStorage{map[string]string{"dummy": "http://ya.ru"}}, nil, nil, nil},
			"dummy",
			"http://ya.ru",
			true,
		},
		{
			"Успешная попытка поиска в списке из 3 элементов",
			fileStorage{&memoryStorage{map[string]string{"dummy": "http://ya.ru", "dummy1": "http://mail.ru", "dummy2": "http://google.ru"}}, nil, nil, nil},
			"dummy1",
			"http://mail.ru",
			true,
		},
		{
			"Неуспешная попытка поиска в непустом списке",
			fileStorage{&memoryStorage{map[string]string{"dummy": "http://ya.ru"}}, nil, nil, nil},
			"dummy1",
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := tt.s.FindURL(tt.URL)
			assert.Equal(t, tt.OK, err == nil)
			assert.Equal(t, tt.wantURL, l)
		})
	}
}
