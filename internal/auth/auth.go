package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
)

const secretKey = "TheKey"
const userIDLength = 5
const cookieAuthentication = "authentication"

type authentication struct {
	userID     string
	cookieSign []byte
	cookieFull string
}

type Authenticator interface {
	Authenticate(http.HandlerFunc) http.HandlerFunc
	GetUserID() string
}

func NewAuth() Authenticator {
	a := authentication{"", make([]byte, 0), ""}
	return &a
}

func getSign(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("не задан user ID пользователя")
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	_, err := h.Write([]byte(id))
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func (a *authentication) authNew() error {
	log.Println("Создание ID для нового пользователя")

	b := make([]byte, userIDLength)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}
	log.Println("Сгенерированы случайные байты для ID нового пользователя:", b)

	a.userID = hex.EncodeToString(b)
	log.Println("Создан ID для нового пользователя:", a.userID)

	a.cookieSign, err = getSign(a.userID)
	if err != nil {
		return err
	}
	log.Println("Сгенерирована подпись в байтах:", a.cookieSign)

	a.cookieFull = a.userID + hex.EncodeToString(a.cookieSign)
	log.Println("Сгенерированы cookie из ID нового пользователя подписи:", a.cookieFull)

	return nil
}

func (a *authentication) authExisting(cookie string) error {
	if cookie == "" {
		return errors.New("не переданы cookieFull для идентификации пользователя")
	}
	log.Println("Получены cookie '"+cookieAuthentication+"':", cookie)

	data, err := hex.DecodeString(cookie)
	if err != nil {
		return err
	}
	log.Println("Cookie расшифрованы в следующие байты:", data)

	if userIDLength*2 < len(cookie) {
		return errors.New("неправильная длина cookie")
	}
	id := cookie[:userIDLength*2]
	log.Println("Из cookie извлечён ID пользователя:", id)
	if id == "" {
		return errors.New("неправильная длина ID пользователя")
	}

	signReceived := data[userIDLength:]
	log.Println("Из cookie извлечена подпись:", signReceived)

	signCalculated, err := getSign(id)
	if err != nil {
		return err
	}
	log.Println("Рассчитана подпись для полученного ID пользователя:", signCalculated)

	if !hmac.Equal(signReceived, signCalculated) {
		return errors.New("в cookie передана неправильная подпись для ID пользователя")
	}

	log.Println("Рассчитанная и полученная подписи для переданного в cookie ID пользователя совпадают")
	a.userID = id
	a.cookieSign = signReceived
	a.cookieFull = cookie
	return nil
}

func (a *authentication) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a == nil {
			return
		}

		cookie, err := r.Cookie(cookieAuthentication)
		if err != nil {
			log.Println("Cookie '" + cookieAuthentication + "' не переданы")
		}

		err = nil
		if cookie != nil {
			err = a.authExisting(cookie.Value)
		}
		if err != nil {
			log.Println("Ошибка при чтении cookie:", err)
		}

		err = nil
		if a.cookieFull == "" {
			err = a.authNew()
		}
		if err != nil {
			log.Println("Ошибка при создании ID пользователя:", err)
		}

		if a.cookieFull != "" {
			http.SetCookie(w, &http.Cookie{Name: cookieAuthentication, Value: a.cookieFull})
		}

		next(w, r)
	}
}

func (a *authentication) GetUserID() string {
	return a.userID
}
