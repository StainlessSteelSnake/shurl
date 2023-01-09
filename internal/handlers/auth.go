package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
)

type user struct {
	id     string
	sign   []byte
	cookie string
}

const secretKey = "TheKey"
const userIdLength = 5
const cookieUser = "user"

func signId(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("не задан id пользователя")
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	_, err := h.Write([]byte(id))
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func newUser() (*user, error) {
	log.Println("Создание нового пользователя")
	b := make([]byte, userIdLength)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	log.Println("Сгенерированы байты:", b)

	u := new(user)
	u.id = hex.EncodeToString(b)
	log.Println("Сгенерирован user id:", u.id)

	u.sign, err = signId(u.id)
	if err != nil {
		return nil, err
	}
	log.Println("Сгенерирована подпись в байтах:", u.sign)

	u.cookie = u.id + hex.EncodeToString(u.sign)
	log.Println("Сгенерированы cookie:", u.cookie)

	return u, nil
}

func (u *user) get(cookie string) error {
	if cookie == "" {
		return errors.New("не переданы cookie для идентификации пользователя")
	}

	log.Println("Получены cookie '"+cookieUser+"':", cookie)
	data, err := hex.DecodeString(cookie)
	if err != nil {
		return err
	}
	log.Println("Cookie расшифрованы в байты:", data)

	id := cookie[:userIdLength*2]
	log.Println("Извлечён id пользователя из cookie:", id)
	if id == "" {
		return errors.New("неправильная длина идентификатора пользователя")
	}

	signReceived := data[userIdLength:]
	log.Println("Извлечена подпись из cookie:", signReceived)

	signCalculated, err := signId(id)
	if err != nil {
		return err
	}
	log.Println("Рассчитана подпись для извлечённого id пользователя:", signCalculated)

	if !hmac.Equal(signReceived, signCalculated) {
		return errors.New("неправильная подпись идентификатора пользователя")
	}

	log.Println("Рассчитанная подпись для полученного id пользователя и полученная в cookie подписи совпадают")
	u.id = id
	u.sign = signReceived
	u.cookie = cookie
	return nil
}

func (h *Handler) handleCookie(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.user.id = ""
		h.user.sign = []byte{}
		h.user.cookie = ""

		cookie, err := r.Cookie(cookieUser)
		if err != nil {
			log.Println("Cookie '" + cookieUser + "' не переданы")
		}

		if cookie != nil {
			err = h.user.get(cookie.Value)
			if err != nil {
				log.Println("Ошибка при чтении cookie:", err)
			}
		}

		if h.user.cookie == "" {
			h.user, err = newUser()
			if err != nil {
				log.Println("Ошибка при создании id пользователя:", err)
			}
		}

		if h.user.cookie != "" {
			http.SetCookie(w, &http.Cookie{Name: cookieUser, Value: h.user.cookie})
		}

		next(w, r)
	}
}
