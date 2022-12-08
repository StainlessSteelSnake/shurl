package storage

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewStorage(t *testing.T) {
	t.Skip()
}

func Test_storage_AddURL(t *testing.T) {
	tests := []struct {
		name       string
		s          Storager
		URL        string
		iterations int
		err        error
	}{
		{
			"Успешное добавление 1 элемента",
			Storager{map[string]string{}},
			"http://ya.ru",
			1,
			nil,
		},
		{
			"Успешное добавление дублирующих элементов",
			Storager{map[string]string{}},
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

func Test_storage_FindURL(t *testing.T) {
	tests := []struct {
		name    string
		s       Storager
		URL     string
		wantURL string
		OK      bool
	}{
		{
			"Неуспешная попытка поиска в пустом хранилище",
			Storager{map[string]string{}},
			"dummy",
			"",
			false,
		},
		{
			"Успешная попытка поиска в списке из 1 элемента",
			Storager{map[string]string{"dummy": "http://ya.ru"}},
			"dummy",
			"http://ya.ru",
			true,
		},
		{
			"Успешная попытка поиска в списке из 3 элементов",
			Storager{map[string]string{"dummy": "http://ya.ru", "dummy1": "http://mail.ru", "dummy2": "http://google.ru"}},
			"dummy1",
			"http://mail.ru",
			true,
		},
		{
			"Неуспешная попытка поиска в непустом списке",
			Storager{map[string]string{"dummy": "http://ya.ru"}},
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
